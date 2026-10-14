package app

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

func mediaProviderOK(provider string) bool { return provider == "emby" || provider == "jellyfin" }
func invalidMediaConfig(c ServiceConfig) bool {
	return c.Interval != 0 || c.TagSync.Active() || c.TagImport || c.TagImportInterval != 0 || c.SyncStart != "" || c.WebhookEnabled
}

// Provider identity and metadata remain separate. Never construct fictional
// language/codec pairings by combining disagreeing server track inventories.
func resolveMediaServers(f *File) {
	if f.MediaServers.count() == 0 {
		return
	}
	engine := f.comparisonEngine
	if engine == nil {
		engine = defaultRecognitionEngine
	}
	if f.APIInfo != nil {
		api := *f.APIInfo
		EnrichWithRules(f, api, engine)
	}
	f.AudioTracks = nil
	f.MediaOrigin = ""
	f.EmbyMatch = "missing"
	var selected *MediaMatch
	providers := []string{}
	statuses := map[string]bool{}
	for _, provider := range []string{"emby", "jellyfin"} {
		value := f.MediaServers.get(provider)
		if value == nil {
			continue
		}
		statuses[value.Status] = true
		if value.Status != "matched" && value.Status != "no_tracks" {
			continue
		}
		if selected != nil && trackInventory(selected.Tracks) != trackInventory(value.Tracks) {
			f.EmbyMatch = "conflicting"
			updateCompleteness(f)
			return
		}
		if selected == nil {
			copy := *value
			selected = &copy
		} else {
			selected.HDR = mergeHDR(selected.HDR, value.HDR)
		}
		providers = append(providers, provider)
	}
	if selected != nil {
		codecs, languages := map[string]bool{}, map[string]bool{}
		for _, track := range selected.Tracks {
			if track.Codec != "" {
				codecs[track.Codec] = true
			}
			if track.Language != "" {
				languages[track.Language] = true
			}
		}
		metadata := embyMetadata{tracks: selected.Tracks, audio: joinEmbyTraits(codecs, " + "), language: joinEmbyTraits(languages, ", "), hdr: selected.HDR}
		applyEmbyMetadata(f, metadata)
		f.MediaOrigin = strings.Join(providers, " + ")
		for field, e := range f.Evidence {
			if e.Emby != "" {
				e.Origin = "mediaServer"
				f.Evidence[field] = e
			}
		}
	} else {
		// Most actionable failure wins for the combined status; individual results
		// are still available in file details.
		for _, status := range []string{"unconfigured", "missing_size", "missing", "pending", "ambiguous"} {
			if statuses[status] {
				f.EmbyMatch = status
			}
		}
	}
	updateCompleteness(f)
}
func trackInventory(tracks []AudioTrack) string {
	values := []string{}
	for _, t := range tracks {
		raw, _ := json.Marshal([]any{t.Language, t.Codec, t.Channels})
		values = append(values, string(raw))
	}
	sort.Strings(values)
	return strings.Join(values, "|")
}

var mediaStateFilters = map[string]string{"mediaMatched": "matched", "mediaMissing": "missing", "mediaAmbiguous": "ambiguous", "mediaConflicting": "conflicting", "mediaNoTracks": "no_tracks", "mediaPending": "pending", "mediaUnconfigured": "unconfigured"}

func appendMediaFilters(q url.Values, w *[]string, a *[]any) error {
	positive, negative := []string{}, []string{}
	keys := []string{}
	for key := range mediaStateFilters {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if v := q.Get(key); v != "" {
			if v != "true" && v != "false" {
				return fmt.Errorf("Invalid media server match filter")
			}
			states := []string{mediaStateFilters[key]}
			if key == "mediaMissing" {
				states = append(states, "missing_size")
			}
			if v == "true" {
				positive = append(positive, states...)
			} else {
				negative = append(negative, states...)
			}
		}
	}
	if len(positive) > 0 {
		*w = append(*w, "coalesce(nullif(f.emby_match,''),'unconfigured') IN ("+strings.TrimRight(strings.Repeat("?,", len(positive)), ",")+")")
		for _, v := range positive {
			*a = append(*a, v)
		}
	}
	if len(negative) > 0 {
		*w = append(*w, "coalesce(nullif(f.emby_match,''),'unconfigured') NOT IN ("+strings.TrimRight(strings.Repeat("?,", len(negative)), ",")+")")
		for _, v := range negative {
			*a = append(*a, v)
		}
	}
	return nil
}
