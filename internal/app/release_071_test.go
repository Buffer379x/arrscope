package app

import (
	"net/url"
	"testing"
	"time"
)

func TestAudioDashboardCountsMatchCompoundFilters(t *testing.T) {
	s := testStore(t)
	files := []File{sample(1, "One.mkv"), sample(2, "Two.mkv"), sample(3, "Three.mkv"), sample(4, "Four.mkv")}
	values := []string{"DTS + AC-3", "DTS + MP3 + DTS", "", "AAC"}
	for i := range files {
		files[i].APIInfo = nil
		files[i].Evidence = nil
		files[i].Audio = values[i]
	}
	if e := s.Replace("radarr", files, SyncStatus{}); e != nil {
		t.Fatal(e)
	}
	series := sample(9, "Series.mkv")
	series.APIInfo = nil
	series.Evidence = nil
	series.Audio = "DTS + AC-3"
	if e := s.Replace("sonarr", []File{series}, SyncStatus{}); e != nil {
		t.Fatal(e)
	}
	stats, e := s.Statistics()
	if e != nil {
		t.Fatal(e)
	}
	if stats.FileTotals["all"] != 5 || stats.FileTotals["radarr"] != 4 || stats.FileTotals["sonarr"] != 1 {
		t.Fatal(stats.FileTotals)
	}
	for _, scope := range []string{"radarr", "sonarr"} {
		for _, bucket := range stats.Distributions[scope]["audio"] {
			out, e := s.Query(url.Values{"service": {scope}, "audio": {bucket.Value}}, "matches", true)
			if e != nil || out.Total != bucket.Count {
				t.Fatalf("%s %q: chart %d filter %d %v", scope, bucket.Value, bucket.Count, out.Total, e)
			}
		}
	}
	counts := map[string]int{}
	for _, b := range stats.Distributions["all"]["audio"] {
		counts[b.Value] = b.Count
	}
	if counts["DTS"] != 3 || counts["AC-3"] != 2 || counts["MP3"] != 1 {
		t.Fatal(counts)
	}
	stats.FileTotals["all"] = 99
	next, e := s.Statistics()
	if e != nil || next.FileTotals["all"] != 5 {
		t.Fatal("cached totals mutated", next, e)
	}
}

func TestStorageAreasDoNotSumOverlappingUnraidShares(t *testing.T) {
	cfg := DefaultSettings()
	cfg.Radarr = ServiceConfig{Enabled: true, URL: "http://radarr"}
	cfg.Sonarr = ServiceConfig{Enabled: true, URL: "http://sonarr"}
	cfg.Widgets.Storage = map[string][]StorageSelection{
		"radarr": {{"/movies", ""}, {"/documentaries", "documentaries"}, {"/downloads", "downloads"}},
		"sonarr": {{"/shows", ""}, {"/documentaries", "documentaries"}, {"/downloads", "downloads"}},
	}
	now := time.Now()
	later := now.Add(time.Minute)
	tib := float64(int64(1) << 40)
	snap := map[string]StorageSnapshot{
		"radarr": {Source: tagEndpoint(cfg.Radarr), SuccessAt: &now, Disks: []DiskSpace{{Path: "/movies", Total: int64(110.9 * tib), Free: int64(34.1 * tib)}, {Path: "/documentaries", Total: int64(20 * tib), Free: int64(8.3 * tib)}, {Path: "/downloads", Total: int64(1.8 * tib), Free: 970000000000}}},
		"sonarr": {Source: tagEndpoint(cfg.Sonarr), SuccessAt: &later, Disks: []DiskSpace{{Path: "/shows", Total: int64(56.4 * tib), Free: int64(11.8 * tib)}, {Path: "/documentaries", Total: int64(20 * tib), Free: int64(8.2 * tib)}, {Path: "/downloads", Total: int64(1.8 * tib), Free: 969000000000}}},
	}
	all := storageAreas(cfg, snap, "all")
	if len(all) != 4 {
		t.Fatal("duplicate shared paths", all)
	}
	for _, a := range all {
		if a.Used != a.Total-a.Free || a.Error != "" {
			t.Fatal(a)
		}
		switch a.Key {
		case "radarr:/movies":
			if a.Total != int64(110.9*tib) {
				t.Fatal("shares were added", a)
			}
		case "shared:documentaries":
			if a.Free != int64(8.2*tib) || len(a.Paths) != 2 {
				t.Fatal("not newest whole sample", a)
			}
		case "shared:downloads":
			if a.Free != 969000000000 {
				t.Fatal(a)
			}
		}
	}
	if len(storageAreas(cfg, snap, "radarr")) != 3 || len(storageAreas(cfg, snap, "sonarr")) != 3 {
		t.Fatal("scope selection")
	}
	broken := snap["sonarr"]
	broken.Error = "timeout"
	snap["sonarr"] = broken
	for _, a := range storageAreas(cfg, snap, "all") {
		if a.Key == "sonarr:/shows" && (a.Total == 0 || a.Error == "") {
			t.Fatal("stale reading lost or unmarked", a)
		}
	}
	cfg.Sonarr.URL = "http://other"
	for _, a := range storageAreas(cfg, snap, "sonarr") {
		if a.Total != 0 || a.Error == "" {
			t.Fatal("old endpoint data exposed", a)
		}
	}
	cfg.Sonarr.Enabled = false
	if len(storageAreas(cfg, snap, "all")) != 3 {
		t.Fatal("disabled service not excluded")
	}
}
