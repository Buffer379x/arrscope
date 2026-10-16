package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestEmbyReadOnlyTracksAndVersions(t *testing.T) {
	wanted := sample(1, "Movie.1080p.DTS.AC3.German.English.mkv")
	wanted.Size = 1234
	missing := sample(2, "Missing.mkv")
	missing.Size = 99
	wrong := sample(3, "WrongVersion.mkv")
	wrong.Size = 44
	calls := 0
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "GET" || r.Header.Get("X-Emby-Token") != "fake-secret" || r.URL.Query().Has("api_key") {
			t.Error("unsafe authentication or mutation")
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/sub/emby/System/Info":
			w.Write([]byte(`{"Version":"4.10.1.0"}`))
		case "/sub/emby/Items":
			if r.URL.Query().Get("IncludeItemTypes") != "Movie" || r.URL.Query().Get("Fields") != "Path,MediaSources,MediaStreams" {
				t.Error("wrong fields")
			}
			w.Write([]byte(`{"TotalRecordCount":1,"Items":[{"Id":"movie1","Type":"Movie","Path":"/library/primary.mkv","MediaStreams":[{"Type":"Audio","Index":1,"Codec":"aac","Language":"fra"}],"MediaSources":[{"Id":"primary","Path":"/library/primary.mkv","Size":500},{"Id":"secondary","Path":"/different/Movie.1080p.DTS.AC3.German.English.mkv","Size":1234,"MediaStreams":[{"Type":"Audio","Index":2,"Codec":"ac3","Language":"eng","Channels":6,"ChannelLayout":"5.1"},{"Type":"Audio","Index":1,"Codec":"dts","Language":"ger","Channels":6,"IsDefault":true},{"Type":"Subtitle","Index":3,"Codec":"subrip","Language":"fra"},{"Type":"Audio","Index":4,"Codec":"aac","Language":"fra","IsExternal":true},{"Type":"Video","Index":0,"Width":1920,"Height":800,"Codec":"h264","BitDepth":8,"ExtendedVideoType":"DolbyVision","ExtendedVideoSubType":"Hdr10"}]},{"Id":"wrong","Path":"/library/WrongVersion.mkv","Size":999,"MediaStreams":[{"Type":"Audio","Index":1,"Codec":"aac","Language":"fra"}]}]}]}`))
		default:
			t.Errorf("unexpected endpoint %s", r.URL.Path)
		}
	}))
	defer mock.Close()
	client := NewEmby(ServiceConfig{URL: mock.URL + "/sub", APIKey: "fake-secret"})
	if version, err := client.Test(context.Background()); err != nil || version != "4.10.1.0" {
		t.Fatal(version, err)
	}
	files := []File{wanted, missing, wrong}
	if err := client.Enrich(context.Background(), "radarr", files); err != nil {
		t.Fatal(err)
	}
	f := files[0]
	if f.EmbyMatch != "matched" || len(f.AudioTracks) != 2 || f.AudioTracks[0].Language != "Deutsch" || f.AudioTracks[0].Codec != "DTS" || f.AudioTracks[1].Language != "Englisch" || f.AudioTracks[1].Codec != "AC-3" {
		t.Fatal(f)
	}
	if f.Audio != "AC-3 + DTS" || f.Language != "Deutsch, Englisch" || f.HDR != "Dolby Vision + HDR10" || f.Evidence["audio"].Origin != "mediaServer" || f.Evidence["codec"].Emby != "AVC" || f.BitDepth != "8" {
		t.Fatal(f)
	}
	if files[1].EmbyMatch != "missing" || files[2].EmbyMatch != "missing" {
		t.Fatal("wrong size/version matched", files)
	}
	if !strings.Contains(audioTrackSummary(f.AudioTracks), "Deutsch → DTS") {
		t.Fatal("CSV mapping missing")
	}
	store := testStore(t)
	if err := store.Replace("radarr", files, SyncStatus{}); err != nil {
		t.Fatal(err)
	}
	result, err := store.Query(url.Values{"service": {"radarr"}, "audio": {"DTS", "AC-3"}, "audioOp": {"all"}, "language": {"Deutsch", "Englisch"}, "languageOp": {"all"}}, "matches", false)
	if err != nil || result.Total != 1 || len(result.Files[0].AudioTracks) != 2 {
		t.Fatal("cached tracks / filters", result, err)
	}
	if err := client.get(context.Background(), "Items/1/Refresh", nil, &f); err == nil {
		t.Fatal("scan endpoint accepted")
	}
	if calls != 2 {
		t.Fatal("unexpected requests", calls)
	}
}

func TestEmbyPaginationAmbiguityAndMissingMetadata(t *testing.T) {
	for _, mode := range []string{"pages", "duplicate", "total-change", "truncated", "missing-size", "no-tracks", "multi-episode"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			total := 251
			mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				var items []embyItem
				if calls == 1 {
					for i := 0; i < 250; i++ {
						items = append(items, embyItem{Id: strings.Repeat("x", i+1), Type: "Episode"})
					}
					items[0] = embyItem{Id: "first", Type: "Episode", MediaSources: []embySource{{Path: "/first/A.mkv", Size: 100, MediaStreams: []embyStream{{Type: "Audio", Index: 1, Codec: "dts", Language: "ger"}, {Type: "Audio", Index: 2, Codec: "ac3", Language: "eng"}}}}}
					if mode == "missing-size" {
						items[0].MediaSources[0].Size = 0
					}
					if mode == "no-tracks" {
						items[0].MediaSources[0].MediaStreams = nil
					}
				} else {
					if r.URL.Query().Get("StartIndex") != "250" {
						t.Error("wrong pagination")
					}
					if mode == "pages" {
						items = []embyItem{{Id: "last", Type: "Episode"}}
					}
					if mode == "duplicate" {
						items = []embyItem{{Id: "first", Type: "Episode"}}
					}
					if mode == "total-change" {
						total = 252
						items = []embyItem{{Id: "last", Type: "Episode"}}
					}
					if mode == "missing-size" || mode == "no-tracks" {
						items = []embyItem{{Id: "last", Type: "Episode"}}
					}
					if mode == "multi-episode" {
						items = []embyItem{{Id: "second", Type: "Episode", MediaSources: []embySource{{Path: "/first/A.mkv", Size: 100, MediaStreams: []embyStream{{Type: "Audio", Index: 1, Codec: "dts", Language: "ger"}, {Type: "Audio", Index: 2, Codec: "ac3", Language: "eng"}}}}}}
					}
				}
				json.NewEncoder(w).Encode(map[string]any{"Items": items, "TotalRecordCount": total})
			}))
			defer mock.Close()
			files := []File{sample(1, "A.mkv")}
			files[0].Size = 100
			err := NewEmby(ServiceConfig{URL: mock.URL}).Enrich(context.Background(), "sonarr", files)
			failure := mode == "duplicate" || mode == "total-change" || mode == "truncated"
			if (err != nil) != failure {
				t.Fatalf("%s: %v", mode, err)
			}
			if mode == "pages" || mode == "multi-episode" {
				if files[0].EmbyMatch != "matched" {
					t.Fatal(files)
				}
			}
			if mode == "missing-size" && files[0].EmbyMatch != "missing" {
				t.Fatal(files)
			}
			if mode == "no-tracks" && files[0].EmbyMatch != "no_tracks" {
				t.Fatal(files)
			}
		})
	}
	for _, duplicateArr := range []bool{false, true} {
		mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"TotalRecordCount":2,"Items":[{"Id":"1","Type":"Movie","Path":"/one/A.mkv","Size":100,"MediaStreams":[{"Index":1,"Type":"Audio","Codec":"dts","Language":"ger"}]},{"Id":"2","Type":"Movie","Path":"/two/A.mkv","Size":100,"MediaStreams":[{"Index":1,"Type":"Audio","Codec":"ac3","Language":"eng"}]}]}`))
		}))
		f := sample(1, "A.mkv")
		f.Size = 100
		files := []File{f}
		if duplicateArr {
			files = append(files, f)
		}
		err := NewEmby(ServiceConfig{URL: mock.URL}).Enrich(context.Background(), "radarr", files)
		mock.Close()
		if err != nil || files[0].EmbyMatch != "ambiguous" || len(files[0].AudioTracks) != 0 {
			t.Fatal("ambiguous mapping used", files, err)
		}
	}
	f := sample(1, "A.mkv")
	applyEmby(&f, []embyStream{{Type: "Audio", Index: 1, Codec: "dts"}, {Type: "Audio", Index: 1, Codec: "ac3"}})
	if f.EmbyMatch != "ambiguous" || len(f.AudioTracks) != 0 {
		t.Fatal("duplicate streams accepted")
	}
	f = sample(1, "A.AAC.German.mkv")
	applyEmby(&f, []embyStream{{Type: "Audio", Index: 1, Language: "und"}})
	if len(f.AudioTracks) != 1 || f.AudioTracks[0].Language != "" || f.AudioTracks[0].Codec != "" || f.Audio != "AAC" || f.Language != "Deutsch" || f.Evidence["audio"].Origin != "filename" {
		t.Fatal("unknown track fields guessed or filename evidence lost", f)
	}
}

func TestEmbyFailuresPreserveSyncAndDoNotLeakKey(t *testing.T) {
	for _, code := range []int{401, 500, 302} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			targetCalls := 0
			target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { targetCalls++ }))
			defer target.Close()
			emby := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if code == 302 {
					http.Redirect(w, r, target.URL, 302)
				} else {
					http.Error(w, "secret-body", code)
				}
			}))
			defer emby.Close()
			arr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.HasSuffix(r.URL.Path, "moviefile") {
					w.Write([]byte(`[{"id":1,"movieId":1,"relativePath":"New.mkv","size":100}]`))
				} else {
					w.Write([]byte(`[{"id":1,"title":"New","year":2020}]`))
				}
			}))
			defer arr.Close()
			store := testStore(t)
			store.Replace("radarr", []File{sample(9, "Kept.mkv")}, SyncStatus{Count: 1})
			syncer := NewSyncer(store)
			err := syncer.Start(context.Background(), "radarr", ServiceConfig{Enabled: true, URL: arr.URL}, ServiceConfig{Enabled: true, URL: emby.URL, APIKey: "secret-token"})
			if err != nil {
				t.Fatal(err)
			}
			syncer.wg.Wait()
			syncer.Shutdown()
			status := syncer.Statuses()["radarr"]
			if status.Error == "" || strings.Contains(status.Error, "secret") || targetCalls != 0 {
				t.Fatal(status, targetCalls)
			}
			result, _ := store.Query(url.Values{"service": {"radarr"}}, "matches", false)
			if result.Total != 1 || result.Files[0].ID != 9 {
				t.Fatal("cache destroyed", result)
			}
		})
	}
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer mock.Close()
	client := NewEmby(ServiceConfig{URL: mock.URL})
	client.transport.HTTP.Timeout = 15 * time.Millisecond
	if _, err := client.Test(context.Background()); err == nil {
		t.Fatal("timeout ignored")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := client.Enrich(ctx, "radarr", []File{{Filename: "A.mkv", Size: 1}}); err == nil {
		t.Fatal("cancel ignored")
	}
}

func TestCompactEmbyCandidatesPreserveRawDuplicateChecks(t *testing.T) {
	for _, mode := range []string{"identical", "different raw codec", "different title", "different stream order"} {
		t.Run(mode, func(t *testing.T) {
			streams := []embyStream{{Type: "Audio", Index: 1, Codec: "dts", Language: "ger"}, {Type: "Audio", Index: 2, Codec: "ac3", Language: "eng"}, {Type: "Video", ExtendedVideoType: "DolbyVision", ExtendedVideoSubType: "Hdr10"}}
			other := append([]embyStream(nil), streams...)
			switch mode {
			case "different raw codec":
				other[0].Codec = "DTS"
			case "different title":
				other[0].Title = "Commentary"
			case "different stream order":
				other[0], other[1] = other[1], other[0]
			}
			mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode(map[string]any{"TotalRecordCount": 2, "Items": []embyItem{{Id: "one", Type: "Movie", Path: "/library/A.mkv", Size: 100, MediaStreams: streams}, {Id: "two", Type: "Movie", Path: "/library/A.mkv", Size: 100, MediaStreams: other}}})
			}))
			defer mock.Close()
			f := sample(1, "A.mkv")
			f.Size = 100
			f.HDR = "HDR10"
			files := []File{f}
			if err := NewEmby(ServiceConfig{URL: mock.URL}).Enrich(context.Background(), "radarr", files); err != nil {
				t.Fatal(err)
			}
			if mode == "identical" {
				if files[0].EmbyMatch != "matched" || len(files[0].AudioTracks) != 2 || files[0].HDR != "Dolby Vision + HDR10" {
					t.Fatal("normalized tracks/HDR changed", files)
				}
			} else if files[0].EmbyMatch != "ambiguous" || len(files[0].AudioTracks) != 0 {
				t.Fatal("raw duplicate differences collapsed", files)
			}
		})
	}
}
