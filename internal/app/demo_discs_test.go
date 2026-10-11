package app

import (
	"net/url"
	"testing"
)

func TestDiscScreenshotFixtureSourcesMatchResolution(t *testing.T) {
	t.Setenv("SCOPEARR_DEMO_DISCS", "true")
	s := testStore(t)
	if err := SeedDemo(s); err != nil {
		t.Fatal(err)
	}
	for _, service := range []string{"radarr", "sonarr"} {
		out, err := s.Query(url.Values{"service": {service}}, "matches", true)
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range out.Files {
			if f.Resolution == "" {
				continue
			}
			if f.Source != demoDiscSource(f.Resolution) {
				t.Fatalf("%s %s %s", f.Title, f.Resolution, f.Source)
			}
		}
	}
}
