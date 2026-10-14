package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestCustomDraftAtomicValidationSwapAndStaleSnapshot(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	a, _ := s.SaveCustomTag(ctx, CustomTag{Name: "Alpha"})
	b, _ := s.SaveCustomTag(ctx, CustomTag{Name: "Beta"})
	original, _ := s.CustomTags(ctx)
	_, err := s.SaveCustomTags(ctx, original, []CustomTag{{ID: a.ID, Name: "Renamed"}, {ID: b.ID, Name: "RENAMED"}})
	if err == nil {
		t.Fatal("duplicate draft accepted")
	}
	current, _ := s.CustomTags(ctx)
	if current[0].Name != "Alpha" || current[1].Name != "Beta" {
		t.Fatal("invalid draft changed stored names", current)
	}
	current, err = s.SaveCustomTags(ctx, original, []CustomTag{{ID: a.ID, Name: "Beta"}, {ID: b.ID, Name: "Alpha"}, {ID: -1, Name: "New"}})
	if err != nil || len(current) != 3 {
		t.Fatal(current, err)
	}
	if _, err = s.SaveCustomTags(ctx, original, original); err == nil {
		t.Fatal("stale draft accepted")
	}
	if _, err = s.SaveCustomTags(ctx, current, []CustomTag{{ID: 99999, Name: "Alien"}}); err == nil {
		t.Fatal("unknown tag accepted")
	}
	after, _ := s.CustomTags(ctx)
	if len(after) != 3 {
		t.Fatal("failed draft removed tags")
	}
}

func TestCustomDraftDeletionCleansRulesColorsPresetsAndAssignments(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	a, _ := s.SaveCustomTag(ctx, CustomTag{Name: "Remove"})
	b, _ := s.SaveCustomTag(ctx, CustomTag{Name: "Keep"})
	key := strconv.FormatInt(a.ID, 10)
	rules := append(DefaultRecognitionRules(), customRule("auto-delete", "custom", "Special", key, "direct", 100))
	if _, err := s.SaveRecognition(ctx, rules, s.Recognition().Revision); err != nil {
		t.Fatal(err)
	}
	cfg := DefaultSettings()
	cfg.TagColors["custom:"+key] = "#ff0000"
	cfg.TagTextColors["custom:"+key] = "#ffffff"
	if err := s.Put("settings", cfg); err != nil {
		t.Fatal(err)
	}
	if err := s.Replace("radarr", []File{sample(1, "Special.1080p.mkv")}, SyncStatus{}); err != nil {
		t.Fatal(err)
	}
	// Preset cleanup also removes the orphaned operator when its last value is deleted.
	q := url.Values{"service": {"radarr"}, "custom": {key}, "customOp": {"not"}}
	if _, err := s.DB.Exec("INSERT INTO presets(name,service,query) VALUES(?,?,?)", "Deleted tag", "radarr", q.Encode()); err != nil {
		t.Fatal(err)
	}
	original, _ := s.CustomTags(ctx)
	if _, err := s.SaveCustomTags(ctx, original, []CustomTag{b}); err == nil {
		t.Fatal("linked tag deletion accepted")
	}
	if _, err := s.ApplyCustom(ctx, url.Values{"service": {"radarr"}}, map[string]bool{key: false}); err != nil {
		t.Fatal(err)
	}
	original, _ = s.CustomTags(ctx)
	if _, err := s.SaveCustomTags(ctx, original, []CustomTag{b}); err != nil {
		t.Fatal(err)
	}
	for _, rule := range s.Recognition().Rules {
		if rule.ID == "auto-delete" {
			t.Fatal("deleted tag rule retained")
		}
	}
	cfg = Settings{}
	if err := s.Get("settings", &cfg); err != nil {
		t.Fatal(err)
	}
	if len(cfg.TagColors) != 0 || len(cfg.TagTextColors) != 0 {
		t.Fatal("orphaned colors", cfg)
	}
	var query string
	if err := s.DB.QueryRow("SELECT query FROM presets WHERE name=?", "Deleted tag").Scan(&query); err != nil {
		t.Fatal(err)
	}
	cleaned, _ := url.ParseQuery(query)
	if cleaned.Has("custom") || cleaned.Has("customOp") {
		t.Fatal(cleaned)
	}
	out, err := s.Query(url.Values{"service": {"radarr"}}, "matches", true)
	if err != nil || len(out.Files) != 1 || len(out.Files[0].Custom) != 0 {
		t.Fatal(out, err)
	}
}

func TestCustomDraftEndpointRequiresAuthenticationAndCSRF(t *testing.T) {
	s := testStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	server, err := NewServer(ctx, s, nil, "0.4.1")
	if err != nil {
		t.Fatal(err)
	}
	defer server.Sync.Shutdown()
	s.DB.Exec("INSERT INTO sessions(token,csrf,expires) VALUES(?,?,?)", tokenHash("batch-session"), "batch-csrf", time.Now().Add(time.Hour).Unix())
	for _, tc := range []struct {
		session, csrf string
		code          int
	}{{"", "", 401}, {"batch-session", "", 403}, {"batch-session", "batch-csrf", 200}} {
		req := httptest.NewRequest("PUT", "/api/custom-tags/batch", strings.NewReader(`{"original":[],"tags":[{"id":-1,"name":"Saved draft"}]}`))
		if tc.session != "" {
			req.AddCookie(&http.Cookie{Name: "scopearr_session", Value: tc.session})
		}
		req.Header.Set("X-CSRF-Token", tc.csrf)
		res := httptest.NewRecorder()
		server.Handler().ServeHTTP(res, req)
		if res.Code != tc.code {
			t.Fatal(res.Code, res.Body.String())
		}
	}
	tags, _ := s.CustomTags(ctx)
	if len(tags) != 1 || tags[0].Name != "Saved draft" {
		t.Fatal(tags)
	}
}
