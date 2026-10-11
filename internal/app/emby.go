package app

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// Emby supplements cached per-track metadata. No playback, probing or refresh endpoints exist here.
type EmbyClient struct {
	transport *ArrClient
	mappings  []pathMapping
}

func NewEmby(c ServiceConfig) *EmbyClient {
	mappings, _ := parsePathMappings(c.PathMappings)
	return &EmbyClient{transport: NewArr(c), mappings: mappings}
}
func (c *EmbyClient) get(ctx context.Context, endpoint string, q url.Values, out any) error {
	if endpoint != "System/Info" && endpoint != "Items" {
		return errors.New("Emby endpoint is not allowed")
	}
	base := strings.TrimRight(c.transport.Base, "/")
	if !strings.HasSuffix(strings.ToLower(base), "/emby") {
		base += "/emby"
	}
	u, err := url.Parse(base + "/" + endpoint)
	if err != nil {
		return errors.New("Invalid Emby address")
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
	Type                 string
	Index                int
	Codec                string
	Profile              string
	Language             string
	DisplayLanguage      string
	Channels             int
	ChannelLayout        string
	Title                string
	IsDefault            bool
	IsExternal           bool
	ExtendedVideoType    string
	ExtendedVideoSubType string
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
		source    embySource
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
		q := url.Values{"Recursive": {"true"}, "IncludeItemTypes": {itemType}, "Fields": {"Path,MediaSources,MediaStreams"}, "EnableImages": {"false"}, "EnableUserData": {"false"}, "SortBy": {"SortName"}, "SortOrder": {"Ascending"}, "StartIndex": {strconv.Itoa(start)}, "Limit": {"250"}}
		var page struct {
			Items            []embyItem
			TotalRecordCount *int
		}
		if err := c.get(ctx, "Items", q, &page); err != nil {
			return err
		}
		if page.Items == nil || page.TotalRecordCount == nil || *page.TotalRecordCount < 0 {
			return errors.New("Emby did not provide a complete page count")
		}
		total := *page.TotalRecordCount
		if expectedTotal < 0 {
			expectedTotal = total
		} else if total != expectedTotal {
			return errors.New("Emby library changed during sync; sync again")
		}
		if len(page.Items) > 250 || start+len(page.Items) > total || (len(page.Items) == 0 && start < total) {
			return errors.New("Incomplete Emby page")
		}
		for _, item := range page.Items {
			if item.Id == "" || seenItems[item.Id] {
				return errors.New("Emby pages contain missing or duplicate IDs")
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
				if old, ok := matches[key]; ok {
					samePath := strings.ReplaceAll(old.source.Path, "\\", "/") == strings.ReplaceAll(source.Path, "\\", "/")
					if old.ambiguous || !samePath || !reflect.DeepEqual(old.source.MediaStreams, source.MediaStreams) {
						old.ambiguous = true
						matches[key] = old
					}
				} else {
					matches[key] = candidate{source: source}
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
		if len(indices) != 1 || (ok && candidate.ambiguous) {
			for _, i := range indices {
				files[i].EmbyMatch = "ambiguous"
			}
			continue
		}
		if !ok {
			continue
		}
		applyEmby(&files[indices[0]], candidate.source.MediaStreams)
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
func applyEmby(f *File, streams []embyStream) {
	tracks := []AudioTrack{}
	codecs, languages, hdr := map[string]bool{}, map[string]bool{}, map[string]bool{}
	indices := map[int]bool{}
	for _, stream := range streams {
		if strings.EqualFold(stream.Type, "Audio") && !stream.IsExternal {
			if indices[stream.Index] {
				f.EmbyMatch = "ambiguous"
				return
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
	if len(tracks) == 0 {
		f.EmbyMatch = "no_tracks"
	} else {
		f.EmbyMatch = "matched"
	}
	sort.Slice(tracks, func(i, j int) bool { return tracks[i].Index < tracks[j].Index })
	f.AudioTracks = tracks
	if len(tracks) > 0 {
		f.AudioStreamCount = len(tracks)
	}
	if f.Evidence == nil {
		f.Evidence = map[string]Evidence{}
	}
	join := func(set map[string]bool, separator string) string {
		values := []string{}
		for v := range set {
			values = append(values, v)
		}
		sort.Strings(values)
		return strings.Join(values, separator)
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
		f.Evidence[field] = evidence
		return value
	}
	// Verified per-track metadata is more precise than flattened API or filename audio traits.
	f.Audio = set("audio", join(codecs, " + "), " + ", f.Audio)
	f.Language = set("language", join(languages, ", "), ", ", f.Language)
	if len(hdr) > 0 {
		embyHDR := join(hdr, " + ")
		// HDR is additive: the API may expose one profile of a backwards-compatible stream.
		for _, v := range strings.Split(f.HDR, " + ") {
			if v != "" {
				hdr[v] = true
			}
		}
		value := join(hdr, " + ")
		e := f.Evidence["hdr"]
		e.Emby = embyHDR
		e.Origin = "combined"
		f.Evidence["hdr"] = e
		f.HDR = value
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
