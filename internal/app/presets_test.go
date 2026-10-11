package app

import (
	"context"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
)

func TestPresetEffectiveEquivalenceAndDistinctFilters(t *testing.T) {
	pairs := [][2]string{
		{"resolution=1080p&resolution=2160p&codec=x265", "codec=x265&resolution=2160p&resolution=1080p&resolution=1080p&resolutionOp=eq&page=4&sort=year&dir=desc"},
		{"language=de&language=eng&languageOp=all", "language=Englisch&language=Deutsch&languageOp=all&sourceOp=not"},
		{"yearMin=02010&sizeMin=0001024", "yearMin=2010&sizeMin=1024"},
		{"hdr=HDR10&hdrOp=notAll", "hdr=HDR10&hdrOp=not"},
		{"q=FOO++bar+foo", "q=bar+foo"},
	}
	for _, pair := range pairs {
		a, err := canonicalPresetQuery(pair[0], "radarr")
		if err != nil {
			t.Fatal(err)
		}
		b, err := canonicalPresetQuery(pair[1], "radarr")
		if err != nil {
			t.Fatal(err)
		}
		if a != b {
			t.Fatalf("different effective keys: %s / %s", a, b)
		}
	}
	for _, pair := range [][2]string{{"codec=x265", "codec=x265&codecOp=not"}, {"hdr=HDR10&hdr=Dolby+Vision&hdrOp=all", "hdr=HDR10&hdr=Dolby+Vision&hdrOp=notAll"}, {"conflict=true", "conflict=false"}} {
		a, _ := canonicalPresetQuery(pair[0], "radarr")
		b, _ := canonicalPresetQuery(pair[1], "radarr")
		if a == b {
			t.Fatal("different filters collapsed")
		}
	}
}

func TestConcurrentPresetCreationAndExistingRecords(t *testing.T) {
	s := testStore(t)
	var created, duplicate atomic.Int32
	var group sync.WaitGroup
	for i := 0; i < 16; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			p, other, err := s.SavePreset(context.Background(), Preset{Name: "Same filters", Service: "radarr", Query: "resolution=1080p&codec=x265"})
			if err != nil {
				t.Error(err)
				return
			}
			if other != nil {
				duplicate.Add(1)
			} else if p.ID > 0 {
				created.Add(1)
			}
		}()
	}
	group.Wait()
	if created.Load() != 1 || duplicate.Load() != 15 {
		t.Fatalf("created %d, duplicate %d", created.Load(), duplicate.Load())
	}
	if _, dup, err := s.SavePreset(context.Background(), Preset{Name: "TV", Service: "sonarr", Query: "resolution=1080p&codec=x265"}); err != nil || dup != nil {
		t.Fatal("service scopes should be separate", err)
	}
	if _, err := s.DB.Exec("INSERT INTO presets(name,service,query) VALUES('Legacy','radarr','language=German&languageOp=all')"); err != nil {
		t.Fatal(err)
	}
	_, dup, err := s.SavePreset(context.Background(), Preset{Name: "New name", Service: "radarr", Query: url.Values{"language": {"Deutsch"}, "languageOp": {"eq"}}.Encode()})
	if err != nil || dup == nil || dup.Name != "Legacy" {
		t.Fatal("legacy alias was not detected", err)
	}
	var query string
	s.DB.QueryRow("SELECT query FROM presets WHERE name='Legacy'").Scan(&query)
	if query != "language=German&languageOp=all" {
		t.Fatal("legacy preset was rewritten")
	}
}
