package app

import (
	"net/url"
	"strings"
	"testing"
)

func TestProcessingColumnMigrationIdempotent(t *testing.T) {
	cfg := DefaultSettings()
	for _, service := range []string{"radarr", "sonarr"} {
		cfg.ColumnOrder[service] = []string{"title", "info", "source", "codec", "resolution", "year", "size", "hdr", "bitDepth", "audio", "channels", "custom"}
		cfg.VisibleColumns[service] = []string{"title", "source", "codec"}
	}
	if e := normalizeVisibleColumns(&cfg); e != nil {
		t.Fatal(e)
	}
	for _, service := range []string{"radarr", "sonarr"} {
		if strings.Join(cfg.VisibleColumns[service], ",") != "title,source,processing,codec" || cfg.ColumnOrder[service][3] != "processing" {
			t.Fatal(cfg)
		}
	}
	// Once migrated, users can hide or move Processing without it reappearing.
	cfg.VisibleColumns["radarr"] = []string{"title", "source"}
	cfg.ColumnOrder["radarr"] = []string{"processing", "title", "info", "source", "codec", "resolution", "year", "size", "hdr", "bitDepth", "audio", "channels", "custom"}
	if e := normalizeVisibleColumns(&cfg); e != nil {
		t.Fatal(e)
	}
	if containsColumn(cfg.VisibleColumns["radarr"], "processing") || cfg.ColumnOrder["radarr"][0] != "processing" {
		t.Fatal(cfg)
	}
}
func TestSourceAndProcessingSortIndependently(t *testing.T) {
	s := testStore(t)
	files := []File{sample(1, "Alpha"), sample(2, "Beta"), sample(3, "Gamma")}
	for i := range files {
		files[i].Source = "BluRay"
		files[i].Processing = []string{"Remux", "", "BDRip"}[i]
	}
	if e := s.Replace("radarr", files, SyncStatus{}); e != nil {
		t.Fatal(e)
	}
	for _, c := range []struct {
		field, dir string
		want       []int
	}{
		{"source", "asc", []int{1, 2, 3}}, {"processing", "asc", []int{3, 1, 2}}, {"processing", "desc", []int{1, 3, 2}},
	} {
		r, e := s.Query(url.Values{"service": {"radarr"}, "sort": {c.field}, "dir": {c.dir}}, "matches", false)
		if e != nil {
			t.Fatal(e)
		}
		for i, f := range r.Files {
			if f.ID != c.want[i] {
				t.Fatal(c, r)
			}
		}
	}
}
