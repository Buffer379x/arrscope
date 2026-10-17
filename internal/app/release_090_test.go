package app

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

func TestDashboardSettingsLegacyAndPersistence(t *testing.T) {
	cfg := DefaultSettings()
	cfg.PageSize = 0
	cfg.Widgets.Order = nil
	cfg.Widgets.Visible = nil
	if e := normalizeSettings(&cfg); e != nil {
		t.Fatal(e)
	}
	if cfg.PageSize != 100 || !reflect.DeepEqual(cfg.Widgets.Order, defaultDashboardOrder()) || len(cfg.Widgets.Visible) != 6 {
		t.Fatal(cfg)
	}
	cfg.PageSize = 3
	cfg.Widgets.Order = []string{"storage", "language", "audio", "source", "codec", "resolution"}
	cfg.Widgets.Visible = []string{}
	store := testStore(t)
	if e := store.Put("settings", cfg); e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	server, e := NewServer(ctx, store, nil, "test")
	if e != nil {
		t.Fatal(e)
	}
	defer server.Sync.Shutdown()
	got := server.Config()
	if got.PageSize != 3 || !reflect.DeepEqual(got.Widgets.Order, cfg.Widgets.Order) || len(got.Widgets.Visible) != 0 {
		t.Fatal("choices lost", got)
	}
	for _, size := range []int{-1, 1001} {
		c := DefaultSettings()
		c.PageSize = size
		if normalizeSettings(&c) == nil {
			t.Fatal("invalid page size", size)
		}
	}
	for _, order := range [][]string{{"source", "source"}, {"unknown"}} {
		c := DefaultSettings()
		c.Widgets.Order = order
		if normalizeSettings(&c) == nil {
			t.Fatal("invalid widgets", order)
		}
	}
}
func TestConfiguredPaginationMatchesAndSeasonContext(t *testing.T) {
	s := testStore(t)
	files := []File{}
	for i := 1; i <= 7; i++ {
		f := sample(i, fmt.Sprintf("Movie.%02d.1080p.mkv", i))
		f.Title = fmt.Sprintf("Movie %02d", i)
		f.Series = "Series"
		f.Season = 1
		files = append(files, f)
	}
	for _, service := range []string{"radarr", "sonarr"} {
		if e := s.Replace(service, files, SyncStatus{}); e != nil {
			t.Fatal(e)
		}
		q := url.Values{"service": {service}, "_pageSize": {"3"}, "page": {"2"}}
		r, e := s.Query(q, "matches", false)
		if e != nil {
			t.Fatal(e)
		}
		if r.PageSize != 3 || r.Total != 7 || len(r.Files) != 3 || r.Files[0].ID != 4 {
			t.Fatal(r)
		}
		q.Set("page", "3")
		r, e = s.Query(q, "matches", false)
		if e != nil || len(r.Files) != 1 || r.Files[0].ID != 7 {
			t.Fatal(r, e)
		}
		all, e := s.Query(q, "matches", true)
		if e != nil || len(all.Files) != 7 {
			t.Fatal("export/bulk truncated", all, e)
		}
	}
	q := url.Values{"service": {"sonarr"}, "_pageSize": {"3"}, "page": {"2"}, "q": {"Movie 04"}}
	r, e := s.Query(q, "season", false)
	if e != nil {
		t.Fatal(e)
	}
	if r.Total != 1 || r.DisplayTotal != 7 || len(r.Files) != 3 || !r.Files[0].Matched || r.Files[1].Matched {
		t.Fatal("context pagination", r)
	}
	for _, bad := range []string{"0", "-1", "1001", "3.5", "invalid"} {
		q.Set("_pageSize", bad)
		if _, e := s.Query(q, "matches", false); e == nil {
			t.Fatal("bad page size accepted", bad)
		}
	}
}

func TestHTTPUsesSavedPageSizeAndExportsAll(t *testing.T) {
	store := testStore(t)
	files := []File{}
	for i := 1; i <= 7; i++ {
		files = append(files, sample(i, fmt.Sprintf("Movie.%02d.1080p.mkv", i)))
	}
	if e := store.Replace("radarr", files, SyncStatus{}); e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	server, e := NewServer(ctx, store, nil, "test")
	if e != nil {
		t.Fatal(e)
	}
	defer server.Sync.Shutdown()
	remote := httptest.NewServer(server.Handler())
	defer remote.Close()
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	csrf := ""
	request := func(path, method string, body any, want int) []byte {
		t.Helper()
		raw, _ := json.Marshal(body)
		r, _ := http.NewRequest(method, remote.URL+path, strings.NewReader(string(raw)))
		r.Header.Set("X-CSRF-Token", csrf)
		resp, e := client.Do(r)
		if e != nil {
			t.Fatal(e)
		}
		defer resp.Body.Close()
		out, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != want {
			t.Fatalf("%d %s", resp.StatusCode, out)
		}
		return out
	}
	request("/api/setup", "POST", map[string]string{"username": "admin", "password": "Synthetic-page-test-2026", "token": server.Security.setupToken}, 200)
	var status struct{ CSRF string }
	json.Unmarshal(request("/api/status", "GET", nil, 200), &status)
	csrf = status.CSRF
	cfg := DefaultSettings()
	cfg.PageSize = 3
	request("/api/settings", "PUT", cfg, 200)
	var result Results
	json.Unmarshal(request("/api/files?service=radarr&page=2&_pageSize=1000", "GET", nil, 200), &result)
	if result.PageSize != 3 || len(result.Files) != 3 || result.Total != 7 {
		t.Fatal("saved size ignored", result)
	}
	rows, e := csv.NewReader(strings.NewReader(string(request("/api/export?service=radarr", "GET", nil, 200)))).ReadAll()
	if e != nil || len(rows) != 8 {
		t.Fatal("CSV truncated", len(rows), e)
	}
	cfg.PageSize = 1001
	request("/api/settings", "PUT", cfg, 400)
	if server.Config().PageSize != 3 {
		t.Fatal("invalid setting changed configuration")
	}
}
