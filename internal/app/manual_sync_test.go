package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestManualEmbySyncQueuesLibrariesWithoutWrites(t *testing.T) {
	store := testStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	server, err := NewServer(ctx, store, nil, "test")
	if err != nil {
		t.Fatal(err)
	}
	defer server.Sync.Shutdown()
	if _, err := store.DB.Exec("INSERT INTO sessions(token,csrf,expires) VALUES(?,?,?)", tokenHash("test-session"), "test-csrf", time.Now().Add(time.Hour).Unix()); err != nil {
		t.Fatal(err)
	}
	gate := make(chan struct{})
	var calls, writes atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != http.MethodGet || (r.URL.Path != "/api/v3/movie" && r.URL.Path != "/api/v3/series") {
			writes.Add(1)
			http.Error(w, "forbidden", 405)
			return
		}
		select {
		case <-gate:
			w.Write([]byte("[]"))
		case <-r.Context().Done():
		}
	}))
	defer upstream.Close()
	server.mu.Lock()
	server.settings.Radarr = ServiceConfig{Enabled: true, URL: upstream.URL, Interval: 0}
	server.settings.Sonarr = ServiceConfig{Enabled: true, URL: upstream.URL, Interval: 0}
	server.settings.Emby.Enabled = true
	server.settings.Emby.URL = upstream.URL
	server.mu.Unlock()
	request := func(csrf bool) int {
		r := httptest.NewRequest(http.MethodPost, "/api/sync", strings.NewReader(`{"service":"emby"}`))
		r.AddCookie(&http.Cookie{Name: "scopearr_session", Value: "test-session"})
		if csrf {
			r.Header.Set("X-CSRF-Token", "test-csrf")
		}
		w := httptest.NewRecorder()
		server.Handler().ServeHTTP(w, r)
		return w.Code
	}
	if got := request(false); got != 403 {
		t.Fatalf("missing CSRF: %d", got)
	}
	if got := request(true); got != 202 {
		t.Fatalf("queue: %d", got)
	}
	if got := request(true); got != 409 {
		t.Fatalf("overlap: %d", got)
	}
	close(gate)
	for _, service := range []string{"radarr", "sonarr"} {
		waitSync(t, server.Sync, service)
		if status := server.Sync.Statuses()[service]; status.Error != "" || status.LastSuccess == nil {
			t.Fatalf("%s: %+v", service, status)
		}
	}
	if calls.Load() != 2 || writes.Load() != 0 {
		t.Fatalf("requests=%d forbidden=%d", calls.Load(), writes.Load())
	}
}
