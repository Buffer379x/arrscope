package app

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

// Emby supplements cached per-track metadata. No playback, probing or refresh endpoints exist here.
type EmbyClient struct {
	transport *ArrClient
	mappings  []pathMapping
	provider  string
}

func NewEmby(c ServiceConfig) *EmbyClient {
	mappings, _ := parsePathMappings(c.PathMappings)
	return &EmbyClient{transport: NewArr(c), mappings: mappings, provider: "emby"}
}
func NewJellyfin(c ServiceConfig) *EmbyClient {
	client := NewEmby(c)
	client.provider = "jellyfin"
	return client
}
func (c *EmbyClient) label() string {
	if c.provider == "jellyfin" {
		return "Jellyfin"
	}
	return "Emby"
}
func (c *EmbyClient) get(ctx context.Context, endpoint string, q url.Values, out any) error {
	if endpoint != "System/Info" && endpoint != "Items" {
		return errors.New(strings.ReplaceAll("Emby endpoint is not allowed", "Emby", c.label()))
	}
	base := strings.TrimRight(c.transport.Base, "/")
	if c.provider != "jellyfin" && !strings.HasSuffix(strings.ToLower(base), "/emby") {
		base += "/emby"
	}
	u, err := url.Parse(base + "/" + endpoint)
	if err != nil {
		return errors.New(strings.ReplaceAll("Invalid Emby address", "Emby", c.label()))
	}
	u.RawQuery = q.Encode()
	return c.transport.readJSON(ctx, u, "X-Emby-Token", out)
}
func (c *EmbyClient) Test(ctx context.Context) (string, error) {
	var info struct{ Version string }
	err := c.get(ctx, "System/Info", nil, &info)
	return info.Version, err
}

type embyStream struct {
	Type                      string
	Index                     int
	Codec                     string
	Profile                   string
	Language                  string
	DisplayLanguage           string
	Channels                  int
	ChannelLayout             string
	Title                     string
	IsDefault                 bool
	IsExternal                bool
	ExtendedVideoType         string
	ExtendedVideoSubType      string
	VideoRangeType            string
	DvBlSignalCompatibilityId *int
	Hdr10PlusPresentFlag      bool
}
type embySource struct {
	Id           string
	Path         string
	Size         int64
	IsRemote     bool
	MediaStreams []embyStream
}
type embyItem struct {
	Id           string
	Type         string
	Path         string
	Size         int64
	MediaStreams []embyStream
	MediaSources []embySource
}
type fileIdentity struct {
	name string
	size int64
}

func pathBasename(path string) string {
	parts := strings.Split(strings.ReplaceAll(path, "\\", "/"), "/")
	return parts[len(parts)-1]
}

func (c *EmbyClient) Enrich(ctx context.Context, service string, files []File) error {
	defer func() {
		for i := range files {
			if value := files[i].MediaServers.get(c.provider); value != nil {
				value.Status = files[i].EmbyMatch
			} else {
				files[i].setMediaMatch(c.provider, MediaMatch{Status: files[i].EmbyMatch})
			}
			resolveMediaServers(&files[i])
			updateCompleteness(&files[i])
		}
	}()
	if !serviceOK(service) {
		return errors.New("Invalid service")
	}
	wanted := map[fileIdentity][]int{}
	for i := range files {
		files[i].EmbyMatch = "missing"

		if files[i].Size <= 0 {
			files[i].EmbyMatch = "missing_size"
			continue
		}
		key := fileIdentity{files[i].Filename, files[i].Size}
		wanted[key] = append(wanted[key], i)
	}
	if len(wanted) == 0 {
		return nil
	}
	type candidate struct {
		path      string
		signature [32]byte
		metadata  embyMetadata
		ambiguous bool
	}
	matches := map[fileIdentity]candidate{}
	itemType := "Movie"
	if service == "sonarr" {
		itemType = "Episode"
	}
	start, expectedTotal := 0, -1
	seenItems := map[string]bool{}
	for {
		q := url.Values{"Recursive": {"true"}, "IncludeItemTypes": {itemType}, "Fields": {"Path,MediaSources,MediaStreams"}, "EnableTotalRecordCount": {"true"}, "EnableImages": {"false"}, "EnableUserData": {"false"}, "SortBy": {"SortName"}, "SortOrder": {"Ascending"}, "StartIndex": {strconv.Itoa(start)}, "Limit": {"250"}}
		var page struct {
			Items            []embyItem
			TotalRecordCount *int
		}
		if err := c.get(ctx, "Items", q, &page); err != nil {
			return err
		}
		if page.Items == nil || page.TotalRecordCount == nil || *page.TotalRecordCount < 0 {
			return errors.New(strings.ReplaceAll("Emby did not provide a complete page count", "Emby", c.label()))
		}
		total := *page.TotalRecordCount
		if expectedTotal < 0 {
			expectedTotal = total
		} else if total != expectedTotal {
			return errors.New(strings.ReplaceAll("Emby library changed during sync; sync again", "Emby", c.label()))
		}
		if len(page.Items) > 250 || start+len(page.Items) > total || (len(page.Items) == 0 && start < total) {
			return errors.New(strings.ReplaceAll("Incomplete Emby page", "Emby", c.label()))
		}
		for _, item := range page.Items {
			if item.Id == "" || seenItems[item.Id] {
				return errors.New(strings.ReplaceAll("Emby pages contain missing or duplicate IDs", "Emby", c.label()))
			}
			seenItems[item.Id] = true
			if item.Type != itemType {
				continue
			}
			sources := item.MediaSources
			// Item-level stream metadata belongs only to the exact item path, never another version.
			if len(sources) == 0 && item.Path != "" {
				sources = []embySource{{Path: item.Path, Size: item.Size, MediaStreams: item.MediaStreams}}
			}
			for _, source := range sources {
				if source.Size <= 0 && source.Path == item.Path {
					source.Size = item.Size
				}
				if source.IsRemote || source.Path == "" || source.Size <= 0 {
					continue
				}
				key := fileIdentity{pathBasename(source.Path), source.Size}
				if _, ok := wanted[key]; !ok {
					continue
				}
				if len(c.mappings) > 0 {
					matchesPath := false
					for _, index := range wanted[key] {
						matchesPath = matchesPath || mappedPathMatches(files[index].SourcePath, source.Path, c.mappings)
					}
					if !matchesPath {
						continue
					}
				}
				if len(source.MediaStreams) == 0 && source.Path == item.Path {
					source.MediaStreams = item.MediaStreams
				}
				raw, err := json.Marshal(source.MediaStreams)
				if err != nil {
					return err
				}
				signature := sha256.Sum256(raw)
				if old, ok := matches[key]; ok {
					samePath := strings.ReplaceAll(old.path, "\\", "/") == strings.ReplaceAll(source.Path, "\\", "/")
					if old.ambiguous || !samePath || old.signature != signature {
						old.ambiguous = true
						matches[key] = old
					}
				} else {
					matches[key] = candidate{path: source.Path, signature: signature, metadata: parseEmbyMetadata(source.MediaStreams)}
				}
			}
		}
		start += len(page.Items)
		if start == total {
			break
		}
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	for key, indices := range wanted {
		candidate, ok := matches[key]
		// Release raw stream records as verified tracks replace them.
		delete(matches, key)
		if len(indices) != 1 || (ok && candidate.ambiguous) {
			for _, i := range indices {
				files[i].EmbyMatch = "ambiguous"
			}
			continue
		}
		if !ok {
			continue
		}
		metadata := candidate.metadata
		status := "matched"
		if metadata.ambiguous {
			status = "ambiguous"
		} else if len(metadata.tracks) == 0 {
			status = "no_tracks"
		}
		files[indices[0]].setMediaMatch(c.provider, MediaMatch{Status: status, Tracks: metadata.tracks, HDR: metadata.hdr})
		files[indices[0]].EmbyMatch = status
	}
	return nil
}
func trackCodec(stream embyStream) string {
	codec := normAudio(stream.Codec)
	if strings.EqualFold(stream.Codec, "dts") {
		switch strings.ToLower(stream.Profile) {
		case "ma", "dts-hd ma", "dts-hd master audio":
			return "DTS-HD MA"
		case "hra", "dts-hd hra", "dts-hd high resolution audio":
			return "DTS-HD"
		}
	}
	return codec
}

// Retain only normalized traits; raw stream structs are much larger than tracks.
// A separate raw-data fingerprint still detects conflicting duplicate sources.
type embyMetadata struct {
	tracks               []AudioTrack
	audio, language, hdr string
	ambiguous            bool
}

func joinEmbyTraits(set map[string]bool, separator string) string {
	values := []string{}
	for v := range set {
		values = append(values, v)
	}
	sort.Strings(values)
	return strings.Join(values, separator)
}
func parseEmbyMetadata(streams []embyStream) embyMetadata {
	tracks := []AudioTrack{}
	codecs, languages, hdr := map[string]bool{}, map[string]bool{}, map[string]bool{}
	indices := map[int]bool{}
	for _, stream := range streams {
		if strings.EqualFold(stream.Type, "Audio") && !stream.IsExternal {
			if indices[stream.Index] {
				return embyMetadata{ambiguous: true}
			}
			indices[stream.Index] = true
			language := stream.Language
			if language == "" {
				language = stream.DisplayLanguage
			}
			language = normLanguage(language)
			codec := trackCodec(stream)
			tracks = append(tracks, AudioTrack{Index: stream.Index, Language: language, Codec: codec, Channels: stream.Channels, ChannelLayout: stream.ChannelLayout, Title: stream.Title, Default: stream.IsDefault})
			if codec != "" {
				codecs[codec] = true
			}
			if language != "" {
				languages[language] = true
			}
		}
		if strings.EqualFold(stream.Type, "Video") {
			switch strings.ToLower(stream.VideoRangeType) {
			case "hdr10":
				hdr["HDR10"] = true
			case "hdr10plus":
				hdr["HDR10+"] = true
			case "hlg":
				hdr["HLG"] = true
			case "dovi":
				hdr["Dolby Vision"] = true
			case "doviwithhdr10":
				hdr["Dolby Vision"] = true
				hdr["HDR10"] = true
			case "doviwithhdr10plus":
				hdr["Dolby Vision"] = true
				hdr["HDR10+"] = true
			case "doviwithhlg":
				hdr["Dolby Vision"] = true
				hdr["HLG"] = true
			}
			if stream.Hdr10PlusPresentFlag {
				hdr["HDR10+"] = true
			}
			switch strings.ToLower(stream.ExtendedVideoType) {
			case "hdr10":
				hdr["HDR10"] = true
			case "hdr10plus":
				hdr["HDR10+"] = true
			case "hyperloggamma":
				hdr["HLG"] = true
			case "dolbyvision":
				hdr["Dolby Vision"] = true
			}
			// Only explicit base-layer subtype values are used, not a guessed DV profile fallback.
			switch strings.ToLower(stream.ExtendedVideoSubType) {
			case "hdr10":
				hdr["HDR10"] = true
			case "hdr10plus0":
				hdr["HDR10+"] = true
			case "hyperloggamma":
				hdr["HLG"] = true
			}
		}
	}
	sort.Slice(tracks, func(i, j int) bool { return tracks[i].Index < tracks[j].Index })
	return embyMetadata{tracks: tracks, audio: joinEmbyTraits(codecs, " + "), language: joinEmbyTraits(languages, ", "), hdr: joinEmbyTraits(hdr, " + ")}
}
func applyEmby(f *File, streams []embyStream) {
	applyEmbyMetadata(f, parseEmbyMetadata(streams))
}
func applyEmbyMetadata(f *File, metadata embyMetadata) {
	if metadata.ambiguous {
		f.EmbyMatch = "ambiguous"
		return
	}
	tracks := metadata.tracks
	if len(tracks) == 0 {
		f.EmbyMatch = "no_tracks"
	} else {
		f.EmbyMatch = "matched"
	}
	f.AudioTracks = tracks
	if len(tracks) > 0 {
		f.AudioStreamCount = len(tracks)
	}
	if f.Evidence == nil {
		f.Evidence = map[string]Evidence{}
	}
	set := func(field, value, separator, fallback string) string {
		evidence := f.Evidence[field]
		evidence.Emby = value
		if value == "" {
			f.Evidence[field] = evidence
			return fallback
		}
		evidence.Origin = "emby"
		evidence.Conflict = evidence.Filename != "" && value != "" && !containsAllTraits(value, evidence.Filename, separator)
		if field == "audio" {
			engine := f.comparisonEngine
			if engine == nil {
				engine = defaultRecognitionEngine
			}
			evidence.Conflict = evidence.Filename != "" && !engine.containsAudio(value, evidence.Filename, evidence.Rules)
		}
		f.Evidence[field] = evidence
		return value
	}
	// Verified per-track metadata is more precise than flattened API or filename audio traits.
	f.Audio = set("audio", metadata.audio, " + ", f.Audio)
	f.Language = set("language", metadata.language, ", ", f.Language)
	if metadata.hdr != "" {
		// Preserve explicit filename/API HDR alongside verified Emby profiles.
		e := f.Evidence["hdr"]
		e.Emby = metadata.hdr
		e.Origin = "combined"
		f.Evidence["hdr"] = e
		profiles := map[string]bool{}
		for _, profile := range strings.Split(f.HDR+" + "+metadata.hdr, " + ") {
			if profile != "" {
				profiles[profile] = true
			}
		}
		f.HDR = joinEmbyTraits(profiles, " + ")
	}
	f.Conflict = false
	for _, e := range f.Evidence {
		f.Conflict = f.Conflict || e.Conflict
	}
	updateCompleteness(f)
}

func audioTrackSummary(tracks []AudioTrack, locale ...string) string {
	labelLanguage := "en"
	if len(locale) > 0 {
		labelLanguage = locale[0]
	}
	values := []string{}
	for _, t := range tracks {
		language, codec := t.Language, t.Codec
		if language == "" {
			language = localizedMessage(labelLanguage, "Unknown")
		}
		if codec == "" {
			codec = localizedMessage(labelLanguage, "Unknown")
		}
		channel := ""
		if t.ChannelLayout != "" {
			channel = " · " + t.ChannelLayout
		} else if t.Channels > 0 {
			channel = " · " + strings.ReplaceAll(localizedMessage(labelLanguage, "{count} channels"), "{count}", strconv.Itoa(t.Channels))
		}
		values = append(values, fmt.Sprintf("#%d %s → %s%s", t.Index, language, codec, channel))
	}
	return strings.Join(values, " | ")
}
