package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestCustomBulkPersistenceRecognitionAndFilterSnapshot(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	a, err := s.SaveCustomTag(ctx, CustomTag{Name: "Zulu"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.SaveCustomTag(ctx, CustomTag{Name: "Alpha"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.SaveCustomTag(ctx, CustomTag{Name: "ALPHA"}); err == nil {
		t.Fatal("duplicate accepted")
	}
	rules := DefaultRecognitionRules()
	rules = append(rules, customRule("tag-a", "custom", "Special", strconv.FormatInt(a.ID, 10), "direct", 100), customRule("tag-b", "custom", "Special", strconv.FormatInt(b.ID, 10), "fallback", 100))
	if _, err = s.SaveRecognition(ctx, rules, s.Recognition().Revision); err != nil {
		t.Fatal(err)
	}
	files := []File{}
	for id := 1; id <= 250; id++ {
		files = append(files, sample(id, "Movie.Special.1080p.x265.mkv"))
	}
	if err = s.Replace("radarr", files, SyncStatus{}); err != nil {
		t.Fatal(err)
	}
	q := url.Values{"service": {"radarr"}, "custom": {strconv.FormatInt(a.ID, 10)}}
	out, err := s.Query(q, "matches", false)
	if err != nil || out.Total != 250 || len(out.Files) != 100 || len(out.Files[0].Custom) != 2 || out.Files[0].Custom[0].Name != "Alpha" {
		t.Fatal(out.Total, err, out.Files[0].Custom)
	}
	// Both criteria resolve from one frozen match snapshot despite removing the filter tag first.
	if n, err := s.ApplyCustom(ctx, q, map[string]bool{strconv.FormatInt(a.ID, 10): false, strconv.FormatInt(b.ID, 10): false}); err != nil || n != 250 {
		t.Fatal(n, err)
	}
	out, err = s.Query(q, "matches", true)
	if err != nil || out.Total != 0 {
		t.Fatal(out.Total, err)
	}
	all := url.Values{"service": {"radarr"}}
	if _, err = s.ApplyCustom(ctx, all, map[string]bool{strconv.FormatInt(b.ID, 10): true}); err != nil {
		t.Fatal(err)
	}
	q["custom"] = []string{strconv.FormatInt(a.ID, 10), strconv.FormatInt(b.ID, 10)}
	q.Set("customOp", "all")
	out, err = s.Query(q, "matches", true)
	if err != nil || out.Total != 0 {
		t.Fatal(out.Total, err)
	}
	q.Set("customOp", "notAll")
	out, err = s.Query(q, "matches", true)
	if err != nil || out.Total != 250 {
		t.Fatal(out.Total, err)
	}
	b.Name = "Aardvark"
	if _, err = s.SaveCustomTag(ctx, b); err != nil {
		t.Fatal(err)
	}
	// Successful sync clears temporary removals, retains positive manual assignments.
	if err = s.Replace("radarr", files, SyncStatus{}); err != nil {
		t.Fatal(err)
	}
	q.Set("customOp", "all")
	out, err = s.Query(q, "matches", true)
	if err != nil || out.Total != 250 || out.Files[0].Custom[0].Name != "Aardvark" {
		t.Fatal(out.Total, err)
	}
	selection, err := s.CustomSelection(ctx, all)
	if err != nil || selection.Counts[strconv.FormatInt(a.ID, 10)] != 250 {
		t.Fatal(selection, err)
	}
	if _, err = s.ApplyCustom(ctx, all, map[string]bool{strconv.FormatInt(a.ID, 10): false, "99999": true}); err == nil {
		t.Fatal("invalid batch accepted")
	}
	out, err = s.Query(q, "matches", true)
	if err != nil || out.Total != 250 {
		t.Fatal("failed bulk changed cache", out.Total, err)
	}
	p, _, err := s.SavePreset(ctx, Preset{Name: "Tags", Service: "radarr", Query: q.Encode()})
	if err != nil {
		t.Fatal(err)
	}
	if err = s.DeleteCustomTag(ctx, a.ID); err == nil {
		t.Fatal("linked tag deletion accepted")
	}
	if _, err = s.ApplyCustom(ctx, all, map[string]bool{strconv.FormatInt(a.ID, 10): false}); err != nil {
		t.Fatal(err)
	}
	if err = s.DeleteCustomTag(ctx, a.ID); err != nil {
		t.Fatal(err)
	}
	presets, err := s.Presets()
	if err != nil || presets[0].ID != p.ID || strings.Contains(presets[0].Query, "custom="+strconv.FormatInt(a.ID, 10)+"&") {
		t.Fatal(presets, err)
	}
	for _, r := range s.Recognition().Rules {
		if r.Field == "custom" && r.Value == strconv.FormatInt(a.ID, 10) {
			t.Fatal("deleted tag rule remains")
		}
	}
}
func TestCustomBulkExcludesSeasonContextAndProtectsAPI(t *testing.T) {
	s := testStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tag, err := s.SaveCustomTag(ctx, CustomTag{Name: "Review"})
	if err != nil {
		t.Fatal(err)
	}
	f, g := sample(1, "One.1080p.mkv"), sample(2, "Two.720p.mkv")
	f.Season, g.Season = 1, 1
	if err = s.Replace("sonarr", []File{f, g}, SyncStatus{}); err != nil {
		t.Fatal(err)
	}
	q := url.Values{"service": {"sonarr"}, "resolution": {"1080p"}}
	if n, err := s.ApplyCustom(ctx, q, map[string]bool{strconv.FormatInt(tag.ID, 10): true}); err != nil || n != 1 {
		t.Fatal(n, err)
	}
	out, err := s.Query(q, "season", false)
	if err != nil || out.DisplayTotal != 2 || len(out.Files[0].Custom) != 1 || len(out.Files[1].Custom) != 0 {
		t.Fatal(out, err)
	}
	server, err := NewServer(ctx, s, nil, "test")
	if err != nil {
		t.Fatal(err)
	}
	defer server.Sync.Shutdown()
	s.DB.Exec("INSERT INTO sessions(token,csrf,expires) VALUES(?,?,?)", tokenHash("test-session"), "test-csrf", time.Now().Add(time.Hour).Unix())
	request := func(csrf bool) int {
		body, _ := json.Marshal(map[string]any{"query": q.Encode(), "changes": map[string]bool{strconv.FormatInt(tag.ID, 10): false}})
		r := httptest.NewRequest(http.MethodPut, "/api/custom-selection", strings.NewReader(string(body)))
		r.AddCookie(&http.Cookie{Name: "scopearr_session", Value: "test-session"})
		if csrf {
			r.Header.Set("X-CSRF-Token", "test-csrf")
		}
		w := httptest.NewRecorder()
		server.Handler().ServeHTTP(w, r)
		return w.Code
	}
	if request(false) != 403 || request(true) != 200 {
		t.Fatal("bulk CSRF protection")
	}
}

func TestCustomDeletionClearsOnlyItsColors(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	tag, err := s.SaveCustomTag(ctx, CustomTag{Name: "Colored"})
	if err != nil {
		t.Fatal(err)
	}
	key := "custom:" + strconv.FormatInt(tag.ID, 10)
	cfg := DefaultSettings()
	cfg.TagColors[key] = "#112233"
	cfg.TagTextColors[key] = "#ffffff"
	cfg.TagColors["source:BluRay"] = "#445566"
	if err = s.Put("settings", cfg); err != nil {
		t.Fatal(err)
	}
	tag.Name = "Renamed"
	if _, err = s.SaveCustomTag(ctx, tag); err != nil {
		t.Fatal(err)
	}
	var after Settings
	s.Get("settings", &after)
	if after.TagColors[key] != "#112233" {
		t.Fatal("rename lost color")
	}
	if err = s.DeleteCustomTag(ctx, tag.ID); err != nil {
		t.Fatal(err)
	}
	after = Settings{}
	s.Get("settings", &after)
	if after.TagColors[key] != "" || after.TagTextColors[key] != "" || after.TagColors["source:BluRay"] != "#445566" {
		t.Fatal("color cleanup", after.TagColors)
	}
}
func TestCustomRecognitionRejectsMissingTagAndIgnoresZeroWidth(t *testing.T) {
	s := testStore(t)
	r := customRule("custom-zero", "custom", `\b`, "123", "direct", 100)
	r.Match = "regex"
	engine, err := CompileRecognition([]RecognitionRule{r})
	if err != nil {
		t.Fatal(err)
	}
	if engine.Recognize("Movie.mkv").Values["custom"] != "" {
		t.Fatal("zero-width tag")
	}
	if _, err = s.SaveRecognition(context.Background(), []RecognitionRule{r}, s.Recognition().Revision); err == nil {
		t.Fatal("unknown tag accepted")
	}
}

func TestCustomCSVUsesActualMatchesAndSortedNames(t *testing.T) {
	s := testStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	a, _ := s.SaveCustomTag(ctx, CustomTag{Name: "Zulu"})
	b, _ := s.SaveCustomTag(ctx, CustomTag{Name: "Alpha"})
	if err := s.Replace("radarr", []File{sample(1, "OnlyA.1080p.mkv"), sample(2, "OnlyB.1080p.mkv")}, SyncStatus{}); err != nil {
		t.Fatal(err)
	}
	q := url.Values{"service": {"radarr"}, "q": {"OnlyA"}}
	if _, err := s.ApplyCustom(ctx, q, map[string]bool{strconv.FormatInt(a.ID, 10): true, strconv.FormatInt(b.ID, 10): true}); err != nil {
		t.Fatal(err)
	}
	server, err := NewServer(ctx, s, nil, "test")
	if err != nil {
		t.Fatal(err)
	}
	defer server.Sync.Shutdown()
	s.DB.Exec("INSERT INTO sessions(token,csrf,expires) VALUES(?,?,?)", tokenHash("csv-session"), "csv-csrf", time.Now().Add(time.Hour).Unix())
	r := httptest.NewRequest("GET", "/api/export?"+q.Encode(), nil)
	r.AddCookie(&http.Cookie{Name: "scopearr_session", Value: "csv-session"})
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "Alpha · Zulu") || strings.Contains(w.Body.String(), "OnlyB") {
		t.Fatal(w.Code, w.Body.String())
	}
}
