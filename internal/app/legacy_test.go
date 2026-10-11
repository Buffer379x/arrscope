package app

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLegacyRenameRetainsWALKeysAccountSessionsAndColumns(t *testing.T) {
	dir := t.TempDir()
	old, err := OpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	sec, err := NewSecurity(old)
	if err != nil {
		t.Fatal(err)
	}
	if err = sec.Setup("admin", "Synthetic-password-2026", sec.setupToken); err != nil {
		t.Fatal(err)
	}
	nonce := make([]byte, sec.aead.NonceSize())
	rand.Read(nonce)
	encrypted := "enc:" + base64.StdEncoding.EncodeToString(sec.aead.Seal(nonce, nonce, []byte("synthetic-key"), []byte("arrscope-api-key-v1")))
	cfg := DefaultSettings()
	cfg.Radarr.APIKey = encrypted
	cfg.VisibleColumns["radarr"] = []string{"title", "audio"}
	old.Put("settings", cfg)
	if err = old.Replace("radarr", []File{sample(1, "Keep.1080p.mkv")}, SyncStatus{}); err != nil {
		t.Fatal(err)
	}
	old.DB.Exec("INSERT INTO sessions(token,csrf,expires) VALUES(?,?,?)", tokenHash("legacy-session"), "legacy-csrf", time.Now().Add(time.Hour).Unix())
	// Reproduce a schema-5 installation before the name change.
	if _, err = old.DB.Exec("DROP TABLE file_custom;DROP TABLE file_custom_auto;DROP TABLE custom_overrides;DROP TABLE custom_tags;PRAGMA user_version=5"); err != nil {
		t.Fatal(err)
	}
	old.DB.Close()
	legacy := filepath.Join(dir, "arrscope.db")
	if err = os.Rename(filepath.Join(dir, "scopearr.db"), legacy); err != nil {
		t.Fatal(err)
	}
	wal, err := sql.Open("sqlite", legacy)
	if err != nil {
		t.Fatal(err)
	}
	defer wal.Close()
	wal.SetMaxOpenConns(1)
	if _, err = wal.Exec("PRAGMA journal_mode=WAL;PRAGMA wal_autocheckpoint=0;INSERT INTO kv(key,value) VALUES('wal-marker','true')"); err != nil {
		t.Fatal(err)
	}
	next, err := OpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer next.DB.Close()
	var marker bool
	if err = next.Get("wal-marker", &marker); err != nil || !marker {
		t.Fatal("committed WAL data lost", err)
	}
	nextSec, err := NewSecurity(next)
	if err != nil {
		t.Fatal(err)
	}
	if !nextSec.Initialized() {
		t.Fatal("account lost")
	}
	value, err := nextSec.Open(encrypted)
	if err != nil || value != "synthetic-key" {
		t.Fatal("legacy key lost", err)
	}
	request := httptest.NewRequest("GET", "/api/settings", nil)
	request.AddCookie(&http.Cookie{Name: "arrscope_session", Value: "legacy-session"})
	if _, csrf, ok := nextSec.Session(request); !ok || csrf != "legacy-csrf" {
		t.Fatal("legacy session lost")
	}
	if err = next.Get("settings", &cfg); err != nil || !containsColumn(cfg.VisibleColumns["radarr"], "custom") {
		t.Fatal("column migration", err)
	}
	result, err := next.QueryContext(context.Background(), url.Values{"service": {"radarr"}}, "matches", false)
	if err != nil || result.Total != 1 {
		t.Fatal(result.Total, err)
	}
	snapshots, _ := filepath.Glob(filepath.Join(dir, "backups", "*.db"))
	if len(snapshots) != 1 {
		t.Fatal("schema-5 backup missing", snapshots)
	}
	if _, err = os.Stat(legacy); err != nil {
		t.Fatal("legacy rollback database removed")
	}
}
