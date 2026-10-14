package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestJellyfinReadsSubpathStreamsWithoutMediaScans(t *testing.T) {
	calls := []string{}
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		if r.Method != "GET" || r.Header.Get("X-Emby-Token") != "synthetic-key" {
			t.Errorf("unexpected request %s", r.URL)
			http.Error(w, "forbidden", 405)
			return
		}
		switch r.URL.Path {
		case "/jellyfin/System/Info":
			w.Write([]byte(`{"Version":"10.11.0"}`))
		case "/jellyfin/Items":
			if r.URL.Query().Get("Fields") != "Path,MediaSources,MediaStreams" || r.URL.Query().Get("EnableTotalRecordCount") != "true" {
				t.Error(r.URL)
			}
			w.Write([]byte(`{"TotalRecordCount":1,"Items":[{"Id":"1","Type":"Episode","MediaSources":[{"Path":"/tv/Episode.mkv","Size":123,"MediaStreams":[{"Type":"Video","VideoRangeType":"DOVIWithHDR10"},{"Type":"Audio","Index":1,"Language":"deu","Codec":"eac3","Channels":6},{"Type":"Audio","Index":2,"Language":"eng","Codec":"aac","Channels":2}]}]}]}`))
		default:
			t.Errorf("endpoint %s", r.URL.Path)
			http.Error(w, "forbidden", 405)
		}
	}))
	defer remote.Close()
	client := NewJellyfin(ServiceConfig{URL: remote.URL + "/jellyfin", APIKey: "synthetic-key"})
	if version, e := client.Test(context.Background()); e != nil || version != "10.11.0" {
		t.Fatal(version, e)
	}
	f := File{ID: 1, Filename: "Episode.mkv", Size: 123}
	Enrich(&f, APIInfo{})
	files := []File{f}
	if e := client.Enrich(context.Background(), "sonarr", files); e != nil {
		t.Fatal(e)
	}
	result := files[0]
	if result.EmbyMatch != "matched" || result.MediaOrigin != "jellyfin" || result.MediaServers.Jellyfin.Status != "matched" || result.MediaServers.count() != 1 || result.HDR != "Dolby Vision + HDR10" || result.Language != "Deutsch, Englisch" || len(result.AudioTracks) != 2 {
		t.Fatal(result)
	}
	if e := client.get(context.Background(), "Items/1/Refresh", nil, nil); e == nil {
		t.Fatal("refresh allowed")
	}
	if len(calls) != 2 {
		t.Fatal(calls)
	}
}
func TestMediaServersAgreeDisagreeAndMissing(t *testing.T) {
	f := File{Filename: "Episode.mkv"}
	Enrich(&f, APIInfo{AudioCodec: "aac", AudioLanguages: "eng"})
	en := AudioTrack{Index: 1, Language: "Englisch", Codec: "AAC", Channels: 2}
	de := AudioTrack{Index: 2, Language: "Deutsch", Codec: "E-AC-3", Channels: 6}
	f.MediaServers = &MediaMatches{Emby: &MediaMatch{Status: "matched", Tracks: []AudioTrack{en, de}}, Jellyfin: &MediaMatch{Status: "matched", Tracks: []AudioTrack{en, de}}}
	resolveMediaServers(&f)
	if f.EmbyMatch != "matched" || len(f.AudioTracks) != 2 || f.MediaOrigin != "emby + jellyfin" {
		t.Fatal(f)
	}
	f.MediaServers.Jellyfin = &MediaMatch{Status: "matched", Tracks: []AudioTrack{en}}
	resolveMediaServers(&f)
	if f.EmbyMatch != "conflicting" || len(f.AudioTracks) != 0 || f.Language != "Englisch" || f.Audio != "AAC" {
		t.Fatal("conflicting metadata was mixed", f)
	}
	f.MediaServers.Jellyfin = &MediaMatch{Status: "missing"}
	resolveMediaServers(&f)
	if f.EmbyMatch != "matched" || f.MediaOrigin != "emby" || len(f.AudioTracks) != 2 {
		t.Fatal(f)
	}
	store := testStore(t)
	if e := store.Replace("sonarr", []File{f}, SyncStatus{}); e != nil {
		t.Fatal(e)
	}
	rules := store.Recognition()
	rules.Rules = append(rules.Rules, RecognitionRule{ID: "resolution-extra", Name: "extra", Field: "resolution", Match: "token", Pattern: "Episode", Value: "1080p", Mode: "direct", Enabled: true})
	if _, e := store.SaveRecognition(context.Background(), rules.Rules, rules.Revision); e != nil {
		t.Fatal(e)
	}
	result, e := store.Query(url.Values{"service": {"sonarr"}}, "matches", true)
	if e != nil || result.Files[0].MediaServers.count() != 2 || len(result.Files[0].AudioTracks) != 2 {
		t.Fatal(result, e)
	}
}
func seasonFiles() []File {
	files := []File{}
	for i := 1; i <= 10; i++ {
		f := File{ID: i, MediaID: 1, Season: 1, Title: "Episode", Filename: "Episode.mkv", Resolution: "1080p", Family: "H.265", Codec: "x265", Audio: "E-AC-3", Channels: "5.1", Language: "Deutsch, Englisch", Episodes: []Episode{{ID: i, Number: i}}}
		files = append(files, f)
	}
	files[3].Resolution = "720p"
	files[3].Family = "H.264"
	files[3].Codec = "x264"
	files[3].Audio = "AAC"
	files[3].Channels = "2.0"
	files[3].Language = "Englisch"
	return files
}
func TestSeasonOutliersUseWholeSeasonAndContextCSVQuery(t *testing.T) {
	s := testStore(t)
	files := seasonFiles()
	if e := s.Replace("sonarr", files, SyncStatus{}); e != nil {
		t.Fatal(e)
	}
	q := url.Values{"service": {"sonarr"}, "seasonOutlier": {"true"}, "languageOutlier": {"true"}}
	result, e := s.Query(q, "matches", true)
	if e != nil || result.Total != 1 || result.Files[0].ID != 4 || !result.Files[0].SeasonOutlier || !result.Files[0].LanguageOutlier || len(result.Files[0].SeasonReasons) != 5 {
		t.Fatal(result, e)
	}
	contextRows, e := s.Query(q, "season", false)
	if e != nil || contextRows.Total != 1 || contextRows.DisplayTotal != 10 {
		t.Fatal(contextRows, e)
	}
	matched := 0
	for _, f := range contextRows.Files {
		if f.Matched {
			matched++
		}
	}
	if matched != 1 {
		t.Fatal(matched)
	}
	q.Set("resolution", "720p")
	r, e := s.Query(q, "matches", false)
	if e != nil || r.Total != 1 {
		t.Fatal("filtered majority changed", r, e)
	}
	// A webhook replacement recomputes the affected series and clears old flags.
	fixed := files[0]
	fixed.ID = 4
	fixed.Episodes = []Episode{{ID: 4, Number: 4}}
	if e = s.replace("sonarr", []File{fixed}, SyncStatus{}, false, &DeltaScope{MediaID: 1, FileIDs: []int{4}}); e != nil {
		t.Fatal(e)
	}
	r, e = s.Query(q, "matches", true)
	if e != nil || r.Total != 0 {
		t.Fatal(r, e)
	}
}
func TestSeasonTiesUnknownsAndMultiEpisodeWeight(t *testing.T) {
	s := testStore(t)
	files := seasonFiles()[:2]
	files[1].Resolution = "720p"
	files[1].Language = "Englisch"
	if e := s.Replace("sonarr", files, SyncStatus{}); e != nil {
		t.Fatal(e)
	}
	q := url.Values{"service": {"sonarr"}, "seasonOutlier": {"true"}}
	r, e := s.Query(q, "matches", true)
	if e != nil || r.Total != 0 {
		t.Fatal(r, e)
	}
	// A file carrying nine episodes forms the known majority over one episode.
	files[0].Episodes = nil
	for i := 1; i <= 9; i++ {
		files[0].Episodes = append(files[0].Episodes, Episode{ID: i, Number: i})
	}
	files[1].Episodes = []Episode{{ID: 10, Number: 10}}
	if e = s.Replace("sonarr", files, SyncStatus{}); e != nil {
		t.Fatal(e)
	}
	r, e = s.Query(q, "matches", true)
	if e != nil || r.Total != 1 || r.Files[0].ID != 2 {
		t.Fatal(r, e)
	}
	files[1].Resolution = ""
	files[1].Language = ""
	if e = s.Replace("sonarr", files, SyncStatus{}); e != nil {
		t.Fatal(e)
	}
	r, e = s.Query(q, "matches", true)
	if e != nil || r.Total != 0 {
		t.Fatal(r, e)
	}
	q.Set("seasonOutlier", "false")
	r, e = s.Query(q, "matches", true)
	if e != nil || r.Total != 2 {
		t.Fatal(r, e)
	}
	q.Set("service", "radarr")
	if _, e = s.Query(q, "matches", true); e == nil {
		t.Fatal("series filter on movies accepted")
	}
}
func TestMediaCheckboxUnionAndExclusion(t *testing.T) {
	s := testStore(t)
	files := seasonFiles()[:5]
	states := []string{"matched", "missing", "missing_size", "ambiguous", "conflicting"}
	for i := range files {
		files[i].EmbyMatch = states[i]
	}
	if e := s.Replace("sonarr", files, SyncStatus{}); e != nil {
		t.Fatal(e)
	}
	q := url.Values{"service": {"sonarr"}, "mediaMatched": {"true"}, "mediaMissing": {"true"}, "mediaAmbiguous": {"false"}}
	r, e := s.Query(q, "matches", true)
	if e != nil || r.Total != 3 {
		t.Fatal(r, e)
	}
	q.Del("mediaMatched")
	q.Del("mediaMissing")
	q.Set("mediaMatched", "false")
	r, e = s.Query(q, "matches", true)
	if e != nil || r.Total != 3 {
		t.Fatal(r, e)
	}
	key, e := canonicalPresetQuery(q.Encode(), "sonarr")
	if e != nil || !strings.Contains(key, "mediaMatched=false") {
		t.Fatal(key, e)
	}
	q.Set("mediaMatched", "bogus")
	if _, e = s.Query(q, "matches", true); e == nil {
		t.Fatal("invalid checkbox")
	}
}
func TestJellyfinKeysStayEncryptedAndPrivate(t *testing.T) {
	s := testStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	server, e := NewServer(ctx, s, nil, "test")
	if e != nil {
		t.Fatal(e)
	}
	defer server.Sync.Shutdown()
	cfg := DefaultSettings()
	cfg.Jellyfin = ServiceConfig{URL: "https://jellyfin.example/subpath", APIKey: server.Security.Seal("private-key")}
	if e = s.Put("settings", cfg); e != nil {
		t.Fatal(e)
	}
	second, e := NewServer(ctx, s, nil, "test")
	if e != nil {
		t.Fatal(e)
	}
	defer second.Sync.Shutdown()
	if second.Config().Jellyfin.APIKey != "private-key" {
		t.Fatal("key did not decrypt")
	}
	public := second.Config()
	for _, connection := range public.Connections() {
		connection.HasKey = connection.APIKey != ""
		connection.APIKey = ""
	}
	raw, _ := json.Marshal(public)
	if strings.Contains(string(raw), "private-key") || !public.Jellyfin.HasKey {
		t.Fatal(string(raw))
	}
}

func TestFailedSecondMediaServerPreservesCompleteSnapshot(t *testing.T) {
	s := testStore(t)
	old := sample(99, "Kept.1080p.x264.mkv")
	if e := s.Replace("radarr", []File{old}, SyncStatus{Count: 1}); e != nil {
		t.Fatal(e)
	}
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Error("upstream write")
			http.Error(w, "forbidden", 405)
			return
		}
		switch r.URL.Path {
		case "/api/v3/movie":
			w.Write([]byte(`[{"id":1,"title":"New","year":2020}]`))
		case "/api/v3/moviefile":
			w.Write([]byte(`[{"id":1,"movieId":1,"relativePath":"New.mkv","size":123}]`))
		case "/emby/Items":
			w.Write([]byte(`{"TotalRecordCount":1,"Items":[{"Id":"1","Type":"Movie","MediaSources":[{"Path":"/New.mkv","Size":123,"MediaStreams":[{"Type":"Audio","Index":1,"Language":"eng","Codec":"aac"}]}]}]}`))
		case "/Items":
			http.Error(w, "unauthorized", 401)
		default:
			t.Errorf("unexpected endpoint %s", r.URL.Path)
			http.Error(w, "forbidden", 405)
		}
	}))
	defer remote.Close()
	syncer := NewSyncer(s)
	defer syncer.Shutdown()
	cfg := ServiceConfig{Enabled: true, URL: remote.URL}
	if e := syncer.Start(context.Background(), "radarr", cfg, cfg, cfg); e != nil {
		t.Fatal(e)
	}
	waitSync(t, syncer, "radarr")
	if !strings.Contains(syncer.Statuses()["radarr"].Error, "Jellyfin") {
		t.Fatal(syncer.Statuses())
	}
	result, e := s.Query(url.Values{"service": {"radarr"}}, "matches", true)
	if e != nil || result.Total != 1 || result.Files[0].ID != 99 {
		t.Fatal("failed supplement replaced cache", result, e)
	}
}

func TestPerformance50KSeasonOutliers(t *testing.T) {
	if os.Getenv("SCOPEARR_PERF") != "1" {
		t.Skip("set SCOPEARR_PERF=1")
	}
	s := testStore(t)
	files := make([]File, 50000)
	for i := range files {
		f := File{ID: i + 1, MediaID: i/50 + 1, Season: 1, Series: fmt.Sprintf("Series %04d", i/50), Title: fmt.Sprintf("Episode %02d", i%50), Filename: "Episode.mkv", Resolution: "1080p", Codec: "x265", Family: "H.265/HEVC", Audio: "E-AC-3", Channels: "5.1", Language: "Deutsch, Englisch", Episodes: []Episode{{ID: i + 1, Number: i%50 + 1}}}
		if i%50 == 49 {
			f.Resolution = "720p"
			f.Codec = "x264"
			f.Family = "H.264/AVC"
			f.Language = "Englisch"
		}
		files[i] = f
	}
	if e := s.Replace("sonarr", files, SyncStatus{}); e != nil {
		t.Fatal(e)
	}
	files = nil
	q := url.Values{"service": {"sonarr"}, "seasonOutlier": {"true"}, "languageOutlier": {"true"}}
	for _, mode := range []string{"matches", "season"} {
		times := []time.Duration{}
		for i := 0; i < 20; i++ {
			start := time.Now()
			r, e := s.Query(q, mode, false)
			times = append(times, time.Since(start))
			if e != nil || r.Total != 1000 || len(r.Files) != 100 {
				t.Fatal(r.Total, len(r.Files), e)
			}
		}
		sort.Slice(times, func(i, j int) bool { return times[i] < times[j] })
		p95 := times[18]
		t.Logf("50,000 episode files, %s season-outlier query p95 %s", mode, p95)
		if p95 > 200*time.Millisecond {
			t.Fatal("season filter exceeds 200ms target", p95)
		}
	}
}
