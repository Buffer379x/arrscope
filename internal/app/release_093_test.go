package app

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"
	"testing"
)

func TestAllMediaChannelLayouts093(t *testing.T) {
	tracks := []AudioTrack{
		{Index: 1, Codec: "DTS-HD MA", Language: "Deutsch", Channels: 6, ChannelLayout: "5.1", Default: true},
		{Index: 2, Codec: "TrueHD", Language: "Englisch", Channels: 8, ChannelLayout: "7.1", Default: true},
		{Index: 3, Codec: "AC-3", Language: "Englisch", Channels: 6, ChannelLayout: "5.1", Default: true},
		{Index: 4, Codec: "DTS-HD MA", Language: "Englisch", Channels: 6, ChannelLayout: "5.1", Default: true},
		{Index: 5, Codec: "AC-3", Language: "Englisch", Channels: 2, ChannelLayout: "stereo"},
	}
	if got := mediaChannels(tracks); got != "2.0 + 5.1 + 7.1" {
		t.Fatal(got)
	}
	if got := mediaChannels([]AudioTrack{{Channels: 6}}); got != "6 ch" {
		t.Fatal("guessed LFE layout", got)
	}
	f := sample(1, "Example.mkv")
	f.setMediaMatch("emby", MediaMatch{Status: "matched", Tracks: tracks})
	resolveMediaServers(&f)
	if f.Channels != "2.0 + 5.1 + 7.1" || len(f.AudioTracks) != 5 || f.AudioTracks[0].Language != "Deutsch" || f.AudioTracks[0].Codec != "DTS-HD MA" || f.AudioTracks[1].Codec != "TrueHD" {
		t.Fatal(f)
	}
	f = sample(2, "Example.5.1.mkv")
	f.setMediaMatch("emby", MediaMatch{Status: "matched", Tracks: tracks})
	resolveMediaServers(&f)
	if f.Channels != "5.1" || f.Evidence["channels"].Emby != "2.0 + 5.1 + 7.1" || f.Evidence["channels"].Origin != "filename" {
		t.Fatal(f)
	}
}

func TestChannelsFiltersAndMatching093(t *testing.T) {
	s := testStore(t)
	a, b := sample(1, "Episode1.mkv"), sample(2, "Episode2.mkv")
	for _, f := range []*File{&a, &b} {
		f.Service = "sonarr"
		f.Series = "Example"
		f.Season = 1
	}
	a.Channels = "2.0 + 5.1 + 7.1"
	a.EmbyMatch = "matched"
	a.APIInfo = nil
	a.Evidence = nil
	b.Channels = "2.0"
	b.EmbyMatch = "missing"
	b.APIInfo = nil
	b.Evidence = nil
	if err := s.Replace("sonarr", []File{a, b}, SyncStatus{}); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		q     url.Values
		total int
	}{
		{url.Values{"channels": {"5.1"}}, 1},
		{url.Values{"channels": {"5.1"}, "channelsOp": {"not"}}, 1},
		{url.Values{"channels": {"2.0", "7.1"}, "channelsOp": {"all"}}, 1},
		{url.Values{"mediaMissing": {"true"}}, 1},
		{url.Values{"mediaMissing": {"true"}, "fileId": {"1"}}, 0},
	} {
		tc.q.Set("service", "sonarr")
		r, err := s.Query(tc.q, "matches", false)
		if err != nil || r.Total != tc.total {
			t.Fatal(tc, r, err)
		}
	}
	facets, err := s.Facets("sonarr")
	if err != nil || !containsString(facets["channels"], "7.1") || containsString(facets["channels"], a.Channels) {
		t.Fatal(facets, err)
	}
}

func TestHiddenTagsSettingsAndCounts093(t *testing.T) {
	s := testStore(t)
	cfg := DefaultSettings()
	cfg.HiddenTags = map[string]bool{"source:__unknown__": true, "codec:x264": false}
	if err := normalizeTagTextColors(&cfg); err != nil || len(cfg.HiddenTags) != 1 {
		t.Fatal(cfg, err)
	}
	if err := s.Put("settings", cfg); err != nil {
		t.Fatal(err)
	}
	var saved Settings
	if err := s.Get("settings", &saved); err != nil || !saved.HiddenTags["source:__unknown__"] {
		t.Fatal(saved, err)
	}
	f := sample(1, "Example.mkv")
	if err := s.Replace("radarr", []File{f}, SyncStatus{}); err != nil {
		t.Fatal(err)
	}
	r, err := s.Query(url.Values{"service": {"radarr"}, "source": {"__unknown__"}}, "matches", false)
	if err != nil || r.Total != 1 {
		t.Fatal(r, err)
	}
	counts, err := traitCounts(context.Background(), s.DB, "source")
	if err != nil || counts["__unknown__"] != 1 {
		t.Fatal(counts, err)
	}
	// Invalid keys cannot create arbitrary UI settings; defaults remain visible.
	for _, key := range []string{"password:x", "source:", strings.Repeat("x", 201)} {
		saved.HiddenTags = map[string]bool{key: true}
		if normalizeTagTextColors(&saved) == nil {
			t.Fatal(key)
		}
	}
	raw, _ := json.Marshal(cfg)
	var restored Settings
	json.Unmarshal(raw, &restored)
	if !restored.HiddenTags["source:__unknown__"] {
		t.Fatal(restored)
	}
	cfg.HiddenTags["codec:x265"] = true
	if err := s.Put("settings", cfg); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteCatalogTag(context.Background(), "codec", "x265"); err != nil {
		t.Fatal(err)
	}
	if err := s.Get("settings", &restored); err != nil || restored.HiddenTags["codec:x265"] || !restored.HiddenTags["source:__unknown__"] {
		t.Fatal(restored, err)
	}
}

func TestAPICodecTagPresentation093(t *testing.T) {
	s := testStore(t)
	f := sample(1, "Example.mkv")
	Enrich(&f, APIInfo{VideoCodec: "vp9"})
	if err := s.Replace("radarr", []File{f}, SyncStatus{}); err != nil {
		t.Fatal(err)
	}
	tags, err := s.DisplayTags(context.Background())
	if err != nil || !containsString(tags["codec"], "vp9") {
		t.Fatal(tags, err)
	}
}
