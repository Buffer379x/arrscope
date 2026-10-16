package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	_ "modernc.org/sqlite"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Store struct {
	presetMu        sync.Mutex
	recognitionMu   sync.Mutex
	recognition     *RecognitionEngine
	DB              *sql.DB
	Dir             string
	cacheMu         sync.Mutex
	facetCache      map[string]map[string][]string
	statisticsCache *Statistics
}

func OpenStore(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	// Never silently initialize an empty account over an existing differently
	// named database. Current deployments use scopearr.db.
	if _, e := os.Stat(filepath.Join(dir, "scopearr.db")); os.IsNotExist(e) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".db") {
				return nil, fmt.Errorf("Existing database found. Restore the active database as scopearr.db before starting")
			}
		}
	}
	db, err := sql.Open("sqlite", filepath.Join(dir, "scopearr.db")+"?_txlock=immediate&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)&_pragma=soft_heap_limit(16777216)&_pragma=cache_size(-1024)&_pragma=temp_store(FILE)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(4)
	s := &Store{DB: db, Dir: dir, facetCache: map[string]map[string][]string{}, recognition: defaultRecognitionEngine}
	fail := func(e error) (*Store, error) { db.Close(); return nil, e }
	if _, err = db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return fail(err)
	}
	var version int
	if err = db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return fail(err)
	}
	originalVersion := version
	if version > 14 {
		return fail(fmt.Errorf("Datenbankversion %d ist neuer als diese App; passende Sicherung wiederherstellen", version))
	}
	if version < 1 {
		var tables int
		db.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table'").Scan(&tables)
		if tables > 0 {
			if err = s.Backup(); err != nil {
				return fail(err)
			}
		}
		_, err = db.Exec(`BEGIN;
CREATE TABLE IF NOT EXISTS kv(key TEXT PRIMARY KEY,value TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS files(rowid INTEGER PRIMARY KEY,service TEXT NOT NULL,id INTEGER NOT NULL,media_id INTEGER NOT NULL,season INTEGER NOT NULL,title TEXT NOT NULL,filename TEXT NOT NULL,resolution TEXT NOT NULL,codec TEXT NOT NULL,family TEXT NOT NULL,source TEXT NOT NULL,processing TEXT NOT NULL,hdr TEXT NOT NULL,bitDepth TEXT NOT NULL,audio TEXT NOT NULL,channels TEXT NOT NULL,language TEXT NOT NULL,size INTEGER NOT NULL,conflict INTEGER NOT NULL,data TEXT NOT NULL,UNIQUE(service,id));
CREATE INDEX IF NOT EXISTS files_service ON files(service,title,id);
CREATE INDEX IF NOT EXISTS files_group ON files(service,media_id,season);
CREATE INDEX IF NOT EXISTS files_filter ON files(service,resolution,codec,source);
CREATE INDEX IF NOT EXISTS files_family ON files(service,family);
CREATE INDEX IF NOT EXISTS files_size ON files(service,size);
CREATE TABLE IF NOT EXISTS years(file_rowid INTEGER REFERENCES files(rowid) ON DELETE CASCADE,year INTEGER NOT NULL,PRIMARY KEY(file_rowid,year));
CREATE INDEX IF NOT EXISTS years_value ON years(year,file_rowid);
CREATE VIRTUAL TABLE IF NOT EXISTS search USING fts5(title,filename, tokenize='unicode61 remove_diacritics 2');
CREATE TRIGGER IF NOT EXISTS files_delete AFTER DELETE ON files BEGIN DELETE FROM search WHERE rowid=old.rowid; END;
CREATE TABLE IF NOT EXISTS presets(id INTEGER PRIMARY KEY,name TEXT NOT NULL,service TEXT NOT NULL,query TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS sessions(token TEXT PRIMARY KEY,csrf TEXT NOT NULL,expires INTEGER NOT NULL);
PRAGMA user_version=1;COMMIT;`)
		if err != nil {
			return fail(err)
		}
		version = 1
	}
	if originalVersion < 13 {
		if _, err = db.Exec(strings.TrimSuffix(seasonSchema, "PRAGMA user_version=14;")); err != nil {
			return fail(err)
		}
	}
	if version < 2 {
		if originalVersion == 1 {
			if err = s.Backup(); err != nil {
				return fail(err)
			}
		}
		_, err = db.Exec(`BEGIN; ALTER TABLE files ADD COLUMN series TEXT NOT NULL DEFAULT ''; ALTER TABLE files ADD COLUMN episode_sort INTEGER NOT NULL DEFAULT 0; UPDATE files SET series=coalesce(json_extract(data,'$.series'),''),episode_sort=coalesce(json_extract(data,'$.episodes[0].number'),0); CREATE INDEX files_series_sort ON files(service,series COLLATE NOCASE,media_id,season,episode_sort,id); PRAGMA user_version=2; COMMIT;`)
		if err != nil {
			return fail(err)
		}
	}
	if version < 4 {
		if originalVersion >= 2 {
			if err = s.Backup(); err != nil {
				return fail(err)
			}
		}
		if err = s.upgradeMediaTraits(); err != nil {
			return fail(err)
		}
	}
	if version < 5 {
		if originalVersion >= 4 {
			if err = s.Backup(); err != nil {
				return fail(err)
			}
		}
		tx, e := db.Begin()
		if e != nil {
			return fail(e)
		}
		for _, column := range []struct{ name, definition string }{{"rule_conflict", "INTEGER NOT NULL DEFAULT 0"}, {"emby_match", "TEXT NOT NULL DEFAULT ''"}} {
			var exists int
			e = tx.QueryRow("SELECT count(*) FROM pragma_table_info('files') WHERE name=?", column.name).Scan(&exists)
			if e != nil {
				break
			}
			if exists == 0 {
				_, e = tx.Exec("ALTER TABLE files ADD COLUMN " + column.name + " " + column.definition)
				if e != nil {
					break
				}
			}
		}
		if e == nil {
			_, e = tx.Exec(`UPDATE files SET emby_match=coalesce(json_extract(data,'$.embyMatch'),''); CREATE INDEX IF NOT EXISTS files_conflicts ON files(service,conflict); CREATE INDEX IF NOT EXISTS files_rule_conflicts ON files(service,rule_conflict); CREATE INDEX IF NOT EXISTS files_emby_match ON files(service,emby_match); CREATE TABLE IF NOT EXISTS file_languages(file_rowid INTEGER REFERENCES files(rowid) ON DELETE CASCADE,language TEXT NOT NULL,PRIMARY KEY(file_rowid,language)); CREATE INDEX IF NOT EXISTS file_languages_value ON file_languages(language,file_rowid);`)
		}
		if e == nil {
			e = reparseCached(context.Background(), tx, defaultRecognitionEngine)
		}
		if e == nil {
			_, e = tx.Exec("PRAGMA user_version=5")
		}
		if e != nil {
			tx.Rollback()
			return fail(e)
		}
		if e = tx.Commit(); e != nil {
			return fail(e)
		}
	}
	if version < 6 {
		if originalVersion >= 5 {
			if err = s.Backup(); err != nil {
				return fail(err)
			}
		}
		var cfg Settings
		saved := s.Get("settings", &cfg) == nil
		if saved {
			for service, columns := range cfg.VisibleColumns {
				if !containsColumn(columns, "custom") {
					cfg.VisibleColumns[service] = append(columns, "custom")
				}
			}
		}
		tx, e := db.Begin()
		if e != nil {
			return fail(e)
		}
		if _, e = tx.Exec(customSchema); e == nil && saved {
			raw, _ := json.Marshal(cfg)
			_, e = tx.Exec("UPDATE kv SET value=? WHERE key='settings'", string(raw))
		}
		if e != nil {
			tx.Rollback()
			return fail(e)
		}
		if e = tx.Commit(); e != nil {
			return fail(e)
		}
	}
	if originalVersion < 7 {
		if originalVersion == 6 {
			if err = s.Backup(); err != nil {
				return fail(err)
			}
		}
		var cfg RecognitionConfig
		loaded := s.Get("recognition", &cfg) == nil
		upgraded, changed := upgradeDefaultRules(cfg.Rules)
		tx, e := db.Begin()
		if e != nil {
			return fail(e)
		}
		if _, e = tx.Exec(tagSyncSchema); e == nil && loaded && changed {
			engine, compileErr := CompileRecognition(upgraded)
			e = compileErr
			if e == nil {
				var raw []byte
				raw, e = json.Marshal(engine.Config())
				if e == nil {
					_, e = tx.Exec("UPDATE kv SET value=? WHERE key='recognition'", string(raw))
				}
			}
			if e == nil {
				e = reparseCached(context.Background(), tx, engine)
			}
		}
		if e != nil {
			tx.Rollback()
			return fail(e)
		}
		if e = tx.Commit(); e != nil {
			return fail(e)
		}
	}

	if originalVersion < 8 {
		if originalVersion == 7 {
			if err = s.Backup(); err != nil {
				return fail(err)
			}
		}
		var cfg RecognitionConfig
		loaded := s.Get("recognition", &cfg) == nil
		upgraded, changed := addUHDBDDefault(cfg.Rules)
		tx, e := db.Begin()
		if e != nil {
			return fail(e)
		}
		if _, e = tx.Exec(tagPolicySchema); e == nil && loaded {
			var engine *RecognitionEngine
			engine, e = CompileRecognition(upgraded)
			if e == nil {
				s.recognition = engine
			}
			if e == nil && changed {
				raw, _ := json.Marshal(engine.Config())
				_, e = tx.Exec("UPDATE kv SET value=? WHERE key='recognition'", string(raw))
				if e == nil {
					e = reparseCached(context.Background(), tx, engine)
				}
			}
		}
		var settings Settings
		if e == nil && s.Get("settings", &settings) == nil {
			for service, c := range map[string]ServiceConfig{"radarr": settings.Radarr, "sonarr": settings.Sonarr} {
				if !c.TagSync.Active() {
					continue
				}
				endpoint := tagEndpoint(c)
				if e = s.seedTagValues(context.Background(), tx, service, endpoint, c.TagSync, true); e != nil {
					break
				}
				for field, on := range c.TagSync.fields() {
					if on {
						_, e = tx.Exec("INSERT OR IGNORE INTO tag_sync_groups VALUES(?,?,?,'active')", service, endpoint, field)
						if e != nil {
							break
						}
					}
				}
				if e != nil {
					break
				}
				_, e = tx.Exec("INSERT INTO kv VALUES(?, 'true') ON CONFLICT(key) DO UPDATE SET value=excluded.value", "tagSyncDirty:"+service)
				if e != nil {
					break
				}
			}
		}
		if e != nil {
			tx.Rollback()
			return fail(e)
		}
		if e = tx.Commit(); e != nil {
			return fail(e)
		}
	}

	if originalVersion < 9 {
		if originalVersion == 8 {
			if err = s.Backup(); err != nil {
				return fail(err)
			}
		}
		var cfg RecognitionConfig
		loaded := s.Get("recognition", &cfg) == nil
		upgraded, changed := upgradeRules053(cfg.Rules)
		tx, e := db.Begin()
		if e != nil {
			return fail(e)
		}
		if _, e = tx.Exec(tagApprovalSchema); e == nil && loaded && changed {
			var engine *RecognitionEngine
			engine, e = CompileRecognition(upgraded)
			if e == nil {
				raw, _ := json.Marshal(engine.Config())
				_, e = tx.Exec("UPDATE kv SET value=? WHERE key='recognition'", string(raw))
			}
			if e == nil {
				e = reparseCached(context.Background(), tx, engine)
			}
		}
		var settings Settings
		if e == nil && s.Get("settings", &settings) == nil {
			for service, c := range map[string]ServiceConfig{"radarr": settings.Radarr, "sonarr": settings.Sonarr} {
				if c.TagSync.Active() {
					_, e = tx.Exec("INSERT INTO kv VALUES(?, 'true') ON CONFLICT(key) DO UPDATE SET value=excluded.value", "tagSyncDirty:"+service)
					if e != nil {
						break
					}
				}
			}
		}
		if e != nil {
			tx.Rollback()
			return fail(e)
		}
		if e = tx.Commit(); e != nil {
			return fail(e)
		}
	}
	if originalVersion < 10 {
		if originalVersion == 9 {
			if err = s.Backup(); err != nil {
				return fail(err)
			}
		}
		var cfg RecognitionConfig
		loaded := s.Get("recognition", &cfg) == nil
		rules, changed := upgradeRules054(cfg.Rules)
		tx, e := db.Begin()
		if e != nil {
			return fail(e)
		}
		if loaded && changed {
			var engine *RecognitionEngine
			engine, e = CompileRecognition(rules)
			if e == nil {
				raw, _ := json.Marshal(engine.Config())
				_, e = tx.Exec("UPDATE kv SET value=? WHERE key='recognition'", string(raw))
			}
			if e == nil {
				e = reparseCached(context.Background(), tx, engine)
			}
			if e == nil {
				var settings Settings
				if s.Get("settings", &settings) == nil {
					for service, config := range map[string]ServiceConfig{"radarr": settings.Radarr, "sonarr": settings.Sonarr} {
						if config.TagSync.Active() {
							_, e = tx.Exec("INSERT INTO kv(key,value) VALUES(?, 'true') ON CONFLICT(key) DO UPDATE SET value=excluded.value", "tagSyncDirty:"+service)
							if e != nil {
								break
							}
						}
					}
				}
			}
		}
		if e == nil {
			_, e = tx.Exec("PRAGMA user_version=10")
		}
		if e != nil {
			tx.Rollback()
			return fail(e)
		}
		if e = tx.Commit(); e != nil {
			return fail(e)
		}
	}

	if originalVersion < 11 {
		if originalVersion == 10 {
			if err = s.Backup(); err != nil {
				return fail(err)
			}
		}
		engine := defaultRecognitionEngine
		var config RecognitionConfig
		if s.Get("recognition", &config) == nil {
			engine, err = CompileRecognition(config.Rules)
			if err != nil {
				return fail(err)
			}
		}
		tx, e := db.Begin()
		if e != nil {
			return fail(e)
		}
		e = reparseCached(context.Background(), tx, engine)
		if e == nil {
			raw, _ := json.Marshal(engine.Config())
			_, e = tx.Exec("INSERT INTO kv(key,value) VALUES('recognition',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", string(raw))
		}
		if e == nil {
			var settings Settings
			if s.Get("settings", &settings) == nil {
				for service, cfg := range map[string]ServiceConfig{"radarr": settings.Radarr, "sonarr": settings.Sonarr} {
					if cfg.TagSync.Active() {
						_, e = tx.Exec("INSERT INTO kv(key,value) VALUES(?, 'true') ON CONFLICT(key) DO UPDATE SET value=excluded.value", "tagSyncDirty:"+service)
						if e != nil {
							break
						}
					}
				}
			}
		}

		if e == nil {
			_, e = tx.Exec("PRAGMA user_version=11")
		}
		if e != nil {
			tx.Rollback()
			return fail(e)
		}
		if e = tx.Commit(); e != nil {
			return fail(e)
		}
	}

	if version < 12 {
		if originalVersion >= 11 {
			if err = s.Backup(); err != nil {
				return fail(err)
			}
		}
		if _, err = db.Exec("BEGIN;" + tagImportSchema + "COMMIT;"); err != nil {
			return fail(err)
		}
	}

	var recognition RecognitionConfig
	if originalVersion < 13 {
		if originalVersion == 12 {
			if err = s.Backup(); err != nil {
				return fail(err)
			}
		}
		if _, err = db.Exec("BEGIN;" + tagCatalogSchema + "COMMIT;"); err != nil {
			return fail(err)
		}
	}
	if originalVersion < 14 {
		if originalVersion == 13 {
			if err = s.Backup(); err != nil {
				return fail(err)
			}
		}
		tx, e := db.Begin()
		if e != nil {
			return fail(e)
		}
		if _, e = tx.Exec(seasonSchema); e == nil {
			e = rebuildSeasonAnalysis(context.Background(), tx, nil)
		}
		if e != nil {
			tx.Rollback()
			return fail(e)
		}
		if e = tx.Commit(); e != nil {
			return fail(e)
		}
	}
	// A compact covering index avoids reading full evidence/audio JSON while
	// counting and selecting IDs for compound filters and season membership.
	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS files_traits ON files(service,resolution,codec,source,hdr,audio,language,media_id,season,processing,family,bitDepth,channels,size,conflict,rule_conflict,emby_match,title COLLATE NOCASE,id)`); err != nil {
		return fail(err)
	}
	// Dashboard reads need only episode identities and completeness, not the large
	// evidence/track JSON. SQLite maintains these compact expressions on every write.
	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS files_statistics ON files(service,media_id,season,id,CASE WHEN json_valid(data) THEN json_extract(data,'$.episodes') END,CASE WHEN json_valid(data) THEN coalesce(json_extract(data,'$.metadataIncomplete'),0) END,json_valid(data))`); err != nil {
		return fail(err)
	}
	// Audio distributions retain unknown-track semantics without scanning full JSON.
	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS files_audio_statistics ON files(service,audio,CASE WHEN json_valid(data) THEN json_extract(data,'$.unknownTraits') END)`); err != nil {
		return fail(err)
	}

	if err = s.Get("recognition", &recognition); err == nil {
		s.recognition, err = CompileRecognition(recognition.Rules)
		if err != nil {
			return fail(err)
		}
	} else if err == sql.ErrNoRows {
		if err = s.Put("recognition", s.recognition.Config()); err != nil {
			return fail(err)
		}
	} else {
		return fail(err)
	}
	if originalVersion < 13 {
		if err = s.seedTagCatalog(context.Background()); err != nil {
			return fail(err)
		}
	}
	os.Chmod(filepath.Join(dir, "scopearr.db"), 0600)
	return s, nil
}
func (s *Store) Backup() error {
	if err := os.MkdirAll(filepath.Join(s.Dir, "backups"), 0700); err != nil {
		return err
	}
	p := filepath.Join(s.Dir, "backups", time.Now().UTC().Format("20060102T150405.000000000")+".db")
	if _, err := s.DB.Exec("VACUUM INTO ?", p); err != nil {
		return err
	}
	os.Chmod(p, 0600)
	fs, _ := filepath.Glob(filepath.Join(s.Dir, "backups", "*.db"))
	sort.Strings(fs)
	if len(fs) > 3 {
		for _, f := range fs[:len(fs)-3] {
			if err := os.Remove(f); err != nil {
				return err
			}
		}
	}
	return nil
}
func (s *Store) Get(key string, v any) error {
	var raw string
	if err := s.DB.QueryRow("SELECT value FROM kv WHERE key=?", key).Scan(&raw); err != nil {
		return err
	}
	return json.Unmarshal([]byte(raw), v)
}
func (s *Store) Put(key string, v any) error {
	b, e := json.Marshal(v)
	if e != nil {
		return e
	}
	_, e = s.DB.Exec("INSERT INTO kv(key,value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", key, string(b))
	return e
}
func (s *Store) Replace(service string, files []File, status SyncStatus) error {
	return s.replace(service, files, status, false, nil)
}

// replaceSyncSnapshot takes ownership of a completed upstream snapshot. Drop
// records as they are written so the growing SQLite transaction does not retain
// a second full library in Go memory. Rollback still preserves the old cache.
func (s *Store) replaceSyncSnapshot(service string, files []File, status SyncStatus) error {
	return s.replace(service, files, status, true, nil)
}

func (s *Store) replace(service string, files []File, status SyncStatus, consume bool, scope *DeltaScope) error {
	if consume {
		defer clear(files)
	}
	s.recognitionMu.Lock()
	defer s.recognitionMu.Unlock()
	tx, e := s.DB.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	customActive := false
	for _, rule := range s.recognition.config.Rules {
		if rule.Enabled && rule.Field == "custom" {
			customActive = true
			break
		}
	}
	if !customActive {
		if e = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM custom_overrides WHERE service=?) OR EXISTS(SELECT 1 FROM external_tag_bindings WHERE service=? AND present=1 AND imported=1)", service, service).Scan(&customActive); e != nil {
			return e
		}
	}
	if _, e = deleteScopedFiles(tx, service, scope); e != nil {
		return e
	}
	st, e := tx.Prepare("INSERT INTO files(service,id,media_id,season,title,filename,resolution,codec,family,source,processing,hdr,bitDepth,audio,channels,language,size,conflict,data,series,episode_sort,rule_conflict,emby_match) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)")
	if e != nil {
		return e
	}
	defer st.Close()
	for index, f := range files {
		if consume {
			files[index] = File{}
		}
		if f.APIInfo != nil || len(f.Evidence) > 0 || s.recognition.config.Revision != defaultRecognitionEngine.config.Revision {
			reparseFile(&f, s.recognition)
		}
		updateCompleteness(&f)
		f.Service = service
		f.Matched = true
		b, e := json.Marshal(f)
		if e != nil {
			return e
		}
		episodeSort := 0
		if len(f.Episodes) > 0 {
			episodeSort = f.Episodes[0].Number
		}
		r, e := st.Exec(service, f.ID, f.MediaID, f.Season, f.Title, f.Filename, f.Resolution, f.Codec, f.Family, f.Source, f.Processing, f.HDR, f.BitDepth, f.Audio, f.Channels, f.Language, f.Size, f.Conflict, string(b), f.Series, episodeSort, f.RuleConflict, f.EmbyMatch)
		if e != nil {
			return e
		}
		id, _ := r.LastInsertId()
		if customActive {
			if e = replaceFileCustom(context.Background(), tx, id, f, true); e != nil {
				return e
			}
		}
		if e = replaceFileLanguages(context.Background(), tx, id, f); e != nil {
			return e
		}
		if _, e = tx.Exec("INSERT INTO search(rowid,title,filename) VALUES(?,?,?)", id, f.Series+" "+f.Title, f.Filename); e != nil {
			return e
		}
		for _, year := range f.Years {
			if _, e = tx.Exec("INSERT OR IGNORE INTO years(file_rowid,year) VALUES(?,?)", id, year); e != nil {
				return e
			}
		}
	}
	if _, e = tx.Exec("DELETE FROM custom_overrides WHERE service=? AND NOT EXISTS(SELECT 1 FROM files f WHERE f.service=custom_overrides.service AND f.id=custom_overrides.file_id)", service); e != nil {
		return e
	}
	if service == "sonarr" {
		var affected *int
		if scope != nil {
			affected = &scope.MediaID
		}
		if e = rebuildSeasonAnalysis(context.Background(), tx, affected); e != nil {
			return e
		}
	}
	if scope == nil {
		b, _ := json.Marshal(status)
		if _, e = tx.Exec("INSERT INTO kv(key,value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", "sync:"+service, string(b)); e != nil {
			return e
		}
	}
	if _, e = tx.Exec("INSERT INTO kv(key,value) VALUES('cacheRevision',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", strconv.Quote(fmt.Sprintf("%d", time.Now().UnixNano()))); e != nil {
		return e
	}
	if e = tx.Commit(); e != nil {
		return e
	}
	s.invalidateAggregates()
	return nil
}

func (s *Store) invalidateAggregates() {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	s.facetCache = map[string]map[string][]string{}
	s.statisticsCache = nil
}

var fields = []string{"resolution", "codec", "family", "source", "processing", "hdr", "bitDepth", "audio", "channels", "language", "custom"}

func serviceOK(s string) bool { return s == "radarr" || s == "sonarr" }
func where(q url.Values) (string, []any, error) {
	if _, e := movieOrder(q); e != nil {
		return "", nil, e
	}
	service := q.Get("service")
	if !serviceOK(service) {
		return "", nil, fmt.Errorf("Invalid service")
	}
	w := []string{"f.service=?"}
	a := []any{service}
	if values, ok := q["fileId"]; ok {
		id, e := strconv.Atoi(q.Get("fileId"))
		if len(values) != 1 || e != nil || id <= 0 {
			return "", nil, fmt.Errorf("Invalid file ID")
		}
		w = append(w, "f.id=?")
		a = append(a, id)
	}
	if search := strings.TrimSpace(q.Get("q")); search != "" {
		var terms []string
		for _, v := range strings.Fields(search) {
			terms = append(terms, `"`+strings.ReplaceAll(v, `"`, `""`)+`"*`)
		}
		w = append(w, "f.rowid IN (SELECT rowid FROM search WHERE search MATCH ?)")
		a = append(a, strings.Join(terms, " AND "))
	}
	for _, field := range fields {
		opValues := q[field+"Op"]
		op := q.Get(field + "Op")
		compound := field == "language" || field == "hdr" || field == "audio" || field == "custom"
		if len(opValues) > 1 || (op != "" && op != "eq" && op != "not" && !(compound && (op == "all" || op == "notAll"))) {
			return "", nil, fmt.Errorf("Invalid filter operator for %s", field)
		}
		negative := op == "not" || op == "notAll"
		values := q[field]
		if field == "custom" && len(values) > 128 {
			return "", nil, fmt.Errorf("At most 128 custom tags are allowed")
		}
		if len(values) == 0 {
			continue
		}
		if compound {
			var parts []string
			for _, v := range values {
				if field == "custom" {
					if v == "__unknown__" {
						parts = append(parts, "NOT EXISTS(SELECT 1 FROM file_custom c WHERE c.file_rowid=f.rowid)")
					} else {
						id, err := strconv.ParseInt(v, 10, 64)
						if err != nil || id <= 0 {
							return "", nil, fmt.Errorf("Invalid custom tag")
						}
						parts = append(parts, "EXISTS(SELECT 1 FROM file_custom c WHERE c.file_rowid=f.rowid AND c.tag_id=?)")
						a = append(a, id)
					}
					continue
				}
				if field == "language" && v != "__unknown__" {
					v = normLanguage(v)
				}
				if v == "__unknown__" || v == "" {
					if field == "language" {
						parts = append(parts, "f.rowid IN (SELECT l.file_rowid FROM file_languages l WHERE l.language='')")
					} else if field == "audio" {
						parts = append(parts, "(f."+field+"='' OR EXISTS(SELECT 1 FROM json_each(f.data,'$.unknownTraits') u WHERE u.value=?))")
						a = append(a, field)
					} else {
						parts = append(parts, "f."+field+"=?")
						a = append(a, "")
					}
					continue
				}
				if field == "language" {
					parts = append(parts, "f.rowid IN (SELECT l.file_rowid FROM file_languages l WHERE l.language=?)")
					a = append(a, v)
				} else {
					parts = append(parts, "(f."+field+"=? OR instr(' + '||f."+field+"||' + ', ' + '||?||' + ')>0)")
					a = append(a, v, v)
				}
			}
			join := " OR "
			if op == "all" || op == "notAll" {
				join = " AND "
			}
			condition := "(" + strings.Join(parts, join) + ")"
			if negative {
				condition = "NOT " + condition
			}
			w = append(w, condition)
			continue
		}
		var marks []string
		for _, v := range values {
			if v == "__unknown__" {
				v = ""
			}
			marks = append(marks, "?")
			a = append(a, v)
		}
		operator := " IN "
		if negative {
			operator = " NOT IN "
		}
		w = append(w, "f."+field+operator+"("+strings.Join(marks, ",")+")")
	}
	if state := q.Get("metadata"); state != "" {
		if state != "complete" && state != "incomplete" {
			return "", nil, fmt.Errorf("Invalid metadata status")
		}
		w = append(w, "coalesce(json_extract(f.data,'$.metadataIncomplete'),0)=?")
		a = append(a, state == "incomplete")
	}
	if c := q.Get("conflict"); c != "" {
		if c != "true" && c != "false" {
			return "", nil, fmt.Errorf("Invalid conflict filter")
		}
		w = append(w, "f.conflict=?")
		a = append(a, c == "true")
	}
	if c := q.Get("ruleConflict"); c != "" {
		if c != "true" && c != "false" {
			return "", nil, fmt.Errorf("Invalid recognition conflict filter")
		}
		w = append(w, "f.rule_conflict=?")
		a = append(a, c == "true")
	}
	if state := q.Get("embyMatch"); state != "" && q.Get("mediaMatch") == "" {
		switch state {
		case "unmatched":
			w = append(w, "f.emby_match IN ('missing','ambiguous','missing_size','no_tracks','pending','conflicting')")
		case "matched", "missing", "ambiguous", "missing_size", "no_tracks", "pending", "conflicting":
			w = append(w, "f.emby_match=?")
			a = append(a, state)
		case "unconfigured":
			w = append(w, "f.emby_match IN ('','unconfigured')")
		default:
			return "", nil, fmt.Errorf("Invalid Emby match filter")
		}
	}
	if e := appendMediaFilters(q, &w, &a); e != nil {
		return "", nil, e
	}
	if e := appendSeasonFilters(q, &w); e != nil {
		return "", nil, e
	}
	if q.Get("yearMin") != "" || q.Get("yearMax") != "" {
		ys := []string{"y.file_rowid=f.rowid"}
		for _, part := range []struct{ key, op string }{{"yearMin", ">="}, {"yearMax", "<="}} {
			if v := q.Get(part.key); v != "" {
				n, e := strconv.Atoi(v)
				if e != nil || n < 1800 || n > 3000 {
					return "", nil, fmt.Errorf("Invalid year")
				}
				ys = append(ys, "y.year"+part.op+"?")
				a = append(a, n)
			}
		}
		w = append(w, "EXISTS(SELECT 1 FROM years y WHERE "+strings.Join(ys, " AND ")+")")
	}
	if q.Get("unknownYear") == "true" {
		w = append(w, "NOT EXISTS(SELECT 1 FROM years y WHERE y.file_rowid=f.rowid)")
	}
	if q.Get("unknownSize") == "true" {
		w = append(w, "f.size=0")
	}
	for _, part := range []struct{ key, op string }{{"sizeMin", ">="}, {"sizeMax", "<="}} {
		if v := q.Get(part.key); v != "" {
			n, e := strconv.ParseInt(v, 10, 64)
			if e != nil || n < 0 {
				return "", nil, fmt.Errorf("Invalid size")
			}
			w = append(w, "f.size"+part.op+"?")
			a = append(a, n)
		}
	}
	return strings.Join(w, " AND "), a, nil
}

type Results struct {
	Files        []File `json:"files"`
	Total        int    `json:"total"`
	DisplayTotal int    `json:"displayTotal"`
	Page         int    `json:"page"`
	PageSize     int    `json:"pageSize"`
}

func (s *Store) Query(q url.Values, mode string, all bool) (Results, error) {
	return s.QueryContext(context.Background(), q, mode, all)
}
func (s *Store) QueryContext(ctx context.Context, q url.Values, mode string, all bool) (Results, error) {
	out := Results{Files: []File{}, Page: 1, PageSize: 100}
	if value := q.Get("_pageSize"); value != "" {
		size, err := strconv.Atoi(value)
		if err != nil || size < 1 || size > 1000 {
			return out, fmt.Errorf("Entries per page must be between 1 and 1000")
		}
		out.PageSize = size
	}
	filter, args, e := where(q)
	if e != nil {
		return out, e
	}
	if v := q.Get("page"); v != "" {
		out.Page, e = strconv.Atoi(v)
		if e != nil || out.Page < 1 || out.Page > 100000 {
			return out, fmt.Errorf("Invalid page")
		}
	}
	tx, e := s.DB.BeginTx(ctx, nil)
	if e != nil {
		return out, e
	}
	defer tx.Rollback()
	if mode == "season" && q.Get("service") == "sonarr" && !all {
		return querySeasonPage(ctx, tx, q, filter, args, out)
	}
	if e = tx.QueryRowContext(ctx, "SELECT count(*) FROM files f WHERE "+filter, args...).Scan(&out.Total); e != nil {
		return out, e
	}
	out.DisplayTotal = out.Total
	order, e := displayOrder(q)
	if e != nil {
		return out, e
	}
	query := "SELECT " + fileDataSQL + ",1," + customJSON + " FROM files f WHERE " + filter + " ORDER BY " + order
	qa := append([]any{}, args...)
	if !all {
		// Sort compact row IDs first, then load and evaluate only the current page.
		// Carrying large evidence/track JSON through a 50k-row temp sort is avoidable.
		query = "WITH page AS MATERIALIZED (SELECT f.rowid FROM files f WHERE " + filter + " ORDER BY " + order + " LIMIT ? OFFSET ?) SELECT " + fileDataSQL + ",1," + customJSON + " FROM page p JOIN files f ON f.rowid=p.rowid ORDER BY " + order
		qa = append(qa, out.PageSize, (out.Page-1)*out.PageSize)
	}

	rows, e := tx.QueryContext(ctx, query, qa...)
	if e != nil {
		return out, e
	}
	defer rows.Close()
	for rows.Next() {
		var raw string
		var matched bool
		var custom string
		if e = rows.Scan(&raw, &matched, &custom); e != nil {
			return out, e
		}
		var f File
		if e = json.Unmarshal([]byte(raw), &f); e != nil {
			return out, e
		}
		if e = json.Unmarshal([]byte(custom), &f.Custom); e != nil {
			return out, e
		}
		f.Matched = matched
		out.Files = append(out.Files, f)
	}
	return out, rows.Err()
}

func querySeasonPage(ctx context.Context, tx *sql.Tx, q url.Values, filter string, args []any, out Results) (Results, error) {
	order, err := displayOrder(q)
	if err != nil {
		return out, err
	}
	// Evaluate costly compound filters once, then reuse compact IDs for season
	// membership, counts and matched markers. The left join retains counts even
	// when no results exist or the requested page is beyond the last page.
	query := `WITH matched AS MATERIALIZED (SELECT f.rowid,f.media_id,f.season FROM files f WHERE ` + filter + `),
groups AS MATERIALIZED (SELECT DISTINCT media_id,season FROM matched),
counts AS MATERIALIZED (SELECT (SELECT count(*) FROM matched) AS total,(SELECT count(*) FROM files f INDEXED BY files_group WHERE f.service='sonarr' AND (f.media_id,f.season) IN (SELECT media_id,season FROM groups)) AS displayed),
page AS MATERIALIZED (SELECT f.rowid FROM files f INDEXED BY files_series_sort WHERE f.service='sonarr' AND (f.media_id,f.season) IN (SELECT media_id,season FROM groups) ORDER BY ` + order + ` LIMIT ? OFFSET ?)
SELECT ` + fileDataSQL + `,coalesce(f.rowid IN (SELECT rowid FROM matched),0),counts.total,counts.displayed,` + customJSON + `
FROM counts LEFT JOIN page p ON 1 LEFT JOIN files f ON f.rowid=p.rowid ORDER BY ` + order
	params := append(append([]any{}, args...), out.PageSize, (out.Page-1)*out.PageSize)
	rows, err := tx.QueryContext(ctx, query, params...)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var raw sql.NullString
		var matched bool
		var custom string
		if err = rows.Scan(&raw, &matched, &out.Total, &out.DisplayTotal, &custom); err != nil {
			return out, err
		}
		if !raw.Valid {
			continue
		}
		var file File
		if err = json.Unmarshal([]byte(raw.String), &file); err != nil {
			return out, err
		}
		if err = json.Unmarshal([]byte(custom), &file.Custom); err != nil {
			return out, err
		}
		file.Matched = matched
		out.Files = append(out.Files, file)
	}
	return out, rows.Err()
}
func (s *Store) facets(ctx context.Context, service string) (map[string][]string, error) {
	if !serviceOK(service) {
		return nil, fmt.Errorf("Invalid service")
	}
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	out := map[string][]string{}
	for _, field := range fields {
		if field == "custom" {
			continue
		}
		rows, e := tx.QueryContext(ctx, "SELECT DISTINCT "+field+" FROM files WHERE service=? ORDER BY "+field, service)
		if e != nil {
			return nil, e
		}
		out[field] = []string{}
		unique := map[string]bool{}
		for rows.Next() {
			var v string
			if e = rows.Scan(&v); e != nil {
				rows.Close()
				return nil, e
			}
			if v == "" {
				v = "__unknown__"
			}
			values := []string{v}
			if field == "language" {
				values = strings.Split(v, ", ")
			}
			if field == "hdr" || field == "audio" {
				values = strings.Split(v, " + ")
			}
			for _, value := range values {
				if !unique[value] {
					out[field] = append(out[field], value)
					unique[value] = true
				}
			}
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return nil, e
		}
		if field == "audio" || field == "language" {
			var unknown bool
			if e := tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM files,json_each(data,'$.unknownTraits') u WHERE service=? AND u.value=?)", service, field).Scan(&unknown); e != nil {
				return nil, e
			}
			if unknown && !unique["__unknown__"] {
				out[field] = append(out[field], "__unknown__")
			}
		}
		sort.Strings(out[field])
	}
	rows, err := tx.QueryContext(ctx, "SELECT DISTINCT cast(c.tag_id AS TEXT) FROM file_custom c JOIN files f ON f.rowid=c.file_rowid JOIN custom_tags t ON t.id=c.tag_id WHERE f.service=? ORDER BY t.name_key", service)
	if err != nil {
		return nil, err
	}
	out["custom"] = []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		out["custom"] = append(out["custom"], id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	var untagged bool
	if err = tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM files f WHERE service=? AND NOT EXISTS(SELECT 1 FROM file_custom c WHERE c.file_rowid=f.rowid))", service).Scan(&untagged); err != nil {
		return nil, err
	}
	if untagged {
		out["custom"] = append(out["custom"], "__unknown__")
	}
	return out, tx.Commit()
}
func (s *Store) Presets() ([]Preset, error) {
	out := []Preset{}
	rows, e := s.DB.Query("SELECT id,name,service,query FROM presets ORDER BY name COLLATE NOCASE")
	if e != nil {
		return out, e
	}
	defer rows.Close()
	for rows.Next() {
		var p Preset
		if e = rows.Scan(&p.ID, &p.Name, &p.Service, &p.Query); e != nil {
			return out, e
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// SQL expressions and direction come exclusively from this allowlist.
func displayOrder(q url.Values) (string, error) {
	order, err := movieOrder(q)
	if err == nil && q.Get("service") == "sonarr" && q.Get("sort") != "info" {
		order = "f.series COLLATE NOCASE,f.media_id,f.season,f.episode_sort,f.id"
	}
	return order, err
}
func movieOrder(q url.Values) (string, error) {
	field := q.Get("sort")
	if field == "info" {
		return infoOrderSQL(q)
	}
	if field == "" {
		field = "title"
	}
	expression, ok := map[string]string{
		"custom": "(SELECT group_concat(name_sort) FROM (SELECT lower(coalesce(nullif(l.label,''),t.name_key)) name_sort FROM file_custom c JOIN custom_tags t ON t.id=c.tag_id LEFT JOIN custom_tag_labels l ON l.tag_id=t.id WHERE c.file_rowid=f.rowid ORDER BY name_sort))",
		"title":  "f.title COLLATE NOCASE", "year": "(SELECT min(year) FROM years y WHERE y.file_rowid=f.rowid)",
		"resolution": "CASE f.resolution WHEN '4320p' THEN 4320 WHEN '2160p' THEN 2160 WHEN '1440p' THEN 1440 WHEN '1080p' THEN 1080 WHEN '720p' THEN 720 WHEN '576p' THEN 576 WHEN '480p' THEN 480 ELSE CAST(f.resolution AS INTEGER) END",
		"codec":      "f.codec COLLATE NOCASE", "source": "f.source COLLATE NOCASE", "processing": "f.processing COLLATE NOCASE", "size": "f.size", "hdr": "f.hdr COLLATE NOCASE", "bitDepth": "CAST(f.bitDepth AS INTEGER)", "audio": "(f.language||' '||f.audio) COLLATE NOCASE", "channels": "CAST(f.channels AS REAL)", "language": "f.language COLLATE NOCASE",
	}[field]
	if !ok {
		return "", fmt.Errorf("Invalid sort field")
	}
	direction := q.Get("dir")
	if direction == "" {
		direction = "asc"
	}
	if direction != "asc" && direction != "desc" {
		return "", fmt.Errorf("Invalid sort direction")
	}
	unknown := "f." + field + "=''"
	if field == "custom" {
		unknown = "NOT EXISTS(SELECT 1 FROM file_custom c WHERE c.file_rowid=f.rowid)"
	}
	if field == "audio" {
		unknown = "f.language='' AND f.audio=''"
	}
	if field == "year" {
		unknown = "NOT EXISTS(SELECT 1 FROM years y WHERE y.file_rowid=f.rowid)"
	}
	if field == "size" {
		unknown = "f.size=0"
	}
	return unknown + " ASC," + expression + " " + strings.ToUpper(direction) + ",f.title COLLATE NOCASE ASC,f.id ASC", nil
}

type StatisticBucket struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}
type Statistics struct {
	FileTotals     map[string]int                          `json:"fileTotals"`
	LanguageTotals map[string]int                          `json:"languageTotals"`
	Movies         int                                     `json:"movies"`
	Series         int                                     `json:"series"`
	Episodes       int                                     `json:"episodes"`
	Files          int                                     `json:"files"`
	Emby           map[string]int                          `json:"emby"`
	Incomplete     int                                     `json:"incomplete"`
	Distributions  map[string]map[string][]StatisticBucket `json:"distributions"`
}

// One read transaction keeps widgets and diagrams consistent during a sync.
func (s *Store) statistics(ctx context.Context) (Statistics, error) {
	out := Statistics{LanguageTotals: map[string]int{}, Emby: map[string]int{}, Distributions: map[string]map[string][]StatisticBucket{}}
	tx, e := s.DB.BeginTx(ctx, nil)
	if e != nil {
		return out, e
	}
	defer tx.Rollback()
	var damaged int
	if e = tx.QueryRowContext(ctx, "SELECT count(*) FROM files INDEXED BY files_statistics WHERE json_valid(data)=0").Scan(&damaged); e != nil {
		return out, e
	}
	if damaged > 0 {
		return out, fmt.Errorf("Invalid cached file data")
	}
	if e = tx.QueryRowContext(ctx, `SELECT (SELECT count(DISTINCT media_id) FROM files WHERE service='radarr'),(SELECT count(DISTINCT media_id) FROM files WHERE service='sonarr'),count(*) FROM files`).Scan(&out.Movies, &out.Series, &out.Files); e != nil {
		return out, e
	}
	if e = tx.QueryRowContext(ctx, `SELECT count(*) FROM (SELECT DISTINCT f.media_id,f.season,json_extract(ep.value,'$.number') FROM files f INDEXED BY files_statistics,json_each(CASE WHEN json_valid(f.data) THEN json_extract(f.data,'$.episodes') END) ep WHERE f.service='sonarr')`).Scan(&out.Episodes); e != nil {
		return out, e
	}
	if e = tx.QueryRowContext(ctx, "SELECT count(*) FROM files INDEXED BY files_statistics WHERE (CASE WHEN json_valid(data) THEN coalesce(json_extract(data,'$.metadataIncomplete'),0) END)=1").Scan(&out.Incomplete); e != nil {
		return out, e
	}
	rows, e := tx.QueryContext(ctx, "SELECT coalesce(nullif(emby_match,''),'unconfigured'),count(*) FROM files INDEXED BY files_emby_match GROUP BY 1")
	if e != nil {
		return out, e
	}
	for rows.Next() {
		var state string
		var n int
		if e = rows.Scan(&state, &n); e != nil {
			rows.Close()
			return out, e
		}
		out.Emby[state] = n
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return out, e
	}
	for _, service := range []string{"all", "radarr", "sonarr"} {
		out.Distributions[service] = map[string][]StatisticBucket{}
		for _, field := range []string{"resolution", "codec", "source"} {
			buckets := []StatisticBucket{}
			query := "SELECT " + field + ",count(*) FROM files"
			args := []any{}
			if service != "all" {
				query += " WHERE service=?"
				args = append(args, service)
			}
			query += " GROUP BY " + field + " ORDER BY count(*) DESC," + field + " COLLATE NOCASE"
			rows, err := tx.QueryContext(ctx, query, args...)
			if err != nil {
				return out, err
			}
			for rows.Next() {
				var b StatisticBucket
				if err = rows.Scan(&b.Value, &b.Count); err != nil {
					rows.Close()
					return out, err
				}
				buckets = append(buckets, b)
			}
			err = rows.Err()
			rows.Close()
			if err != nil {
				return out, err
			}
			out.Distributions[service][field] = buckets
		}
	}
	if e = addAudioStatistics(ctx, tx, &out); e != nil {
		return out, e
	}
	// Decode episode associations once, rather than once per language and scope.
	// DISTINCT entity identities retain multi-episode and duplicate-file semantics.
	entity := `f.service||':'||f.media_id||CASE WHEN f.service='sonarr' THEN ':'||f.season||':'||CASE WHEN json_extract(ep.value,'$.number') IS NULL THEN 'file:'||f.id ELSE 'episode:'||json_extract(ep.value,'$.number') END ELSE '' END`
	query := `WITH entities AS MATERIALIZED (SELECT f.rowid AS file_rowid,f.service,` + entity + ` AS identity FROM files f INDEXED BY files_statistics LEFT JOIN json_each(CASE WHEN f.service='sonarr' THEN (CASE WHEN json_valid(f.data) THEN json_extract(f.data,'$.episodes') END) ELSE '[]' END) ep),
scopes(service) AS (VALUES('all'),('radarr'),('sonarr'))
SELECT s.service,'' AS language,1 AS is_total,count(DISTINCT e.identity) AS n FROM scopes s LEFT JOIN entities e ON s.service='all' OR e.service=s.service GROUP BY s.service
UNION ALL
SELECT s.service,l.language,0,count(DISTINCT e.identity) FROM scopes s JOIN entities e ON s.service='all' OR e.service=s.service JOIN file_languages l ON l.file_rowid=e.file_rowid GROUP BY s.service,l.language
ORDER BY 1,3 DESC,4 DESC,2`
	for _, service := range []string{"all", "radarr", "sonarr"} {
		out.Distributions[service]["language"] = []StatisticBucket{}
	}
	rows, e = tx.QueryContext(ctx, query)
	if e != nil {
		return out, e
	}
	for rows.Next() {
		var service string
		var bucket StatisticBucket
		var total bool
		if e = rows.Scan(&service, &bucket.Value, &total, &bucket.Count); e != nil {
			rows.Close()
			return out, e
		}
		if total {
			out.LanguageTotals[service] = bucket.Count
		} else {
			out.Distributions[service]["language"] = append(out.Distributions[service]["language"], bucket)
		}
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return out, e
	}
	return out, tx.Commit()
}
