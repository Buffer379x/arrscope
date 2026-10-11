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
	db, err := sql.Open("sqlite", filepath.Join(dir, "arrscope.db")+"?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(4)
	s := &Store{DB: db, Dir: dir, facetCache: map[string]map[string][]string{}}
	fail := func(e error) (*Store, error) { db.Close(); return nil, e }
	if _, err = db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return fail(err)
	}
	var version int
	if err = db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return fail(err)
	}
	originalVersion := version
	if version > 4 {
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
	os.Chmod(filepath.Join(dir, "arrscope.db"), 0600)
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
	tx, e := s.DB.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if _, e = tx.Exec("DELETE FROM files WHERE service=?", service); e != nil {
		return e
	}
	st, e := tx.Prepare("INSERT INTO files(service,id,media_id,season,title,filename,resolution,codec,family,source,processing,hdr,bitDepth,audio,channels,language,size,conflict,data,series,episode_sort) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)")
	if e != nil {
		return e
	}
	defer st.Close()
	for _, f := range files {
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
		r, e := st.Exec(service, f.ID, f.MediaID, f.Season, f.Title, f.Filename, f.Resolution, f.Codec, f.Family, f.Source, f.Processing, f.HDR, f.BitDepth, f.Audio, f.Channels, f.Language, f.Size, f.Conflict, string(b), f.Series, episodeSort)
		if e != nil {
			return e
		}
		id, _ := r.LastInsertId()
		if _, e = tx.Exec("INSERT INTO search(rowid,title,filename) VALUES(?,?,?)", id, f.Series+" "+f.Title, f.Filename); e != nil {
			return e
		}
		for _, year := range f.Years {
			if _, e = tx.Exec("INSERT OR IGNORE INTO years(file_rowid,year) VALUES(?,?)", id, year); e != nil {
				return e
			}
		}
	}
	b, _ := json.Marshal(status)
	if _, e = tx.Exec("INSERT INTO kv(key,value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", "sync:"+service, string(b)); e != nil {
		return e
	}
	if e = tx.Commit(); e != nil {
		return e
	}
	s.cacheMu.Lock()
	s.facetCache = map[string]map[string][]string{}
	s.statisticsCache = nil
	s.cacheMu.Unlock()
	return nil
}

var fields = []string{"resolution", "codec", "family", "source", "processing", "hdr", "bitDepth", "audio", "channels", "language"}

func serviceOK(s string) bool { return s == "radarr" || s == "sonarr" }
func where(q url.Values) (string, []any, error) {
	if _, e := movieOrder(q); e != nil {
		return "", nil, e
	}
	service := q.Get("service")
	if !serviceOK(service) {
		return "", nil, fmt.Errorf("Ungültiger Dienst")
	}
	w := []string{"f.service=?"}
	a := []any{service}
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
		compound := field == "language" || field == "hdr" || field == "audio"
		if len(opValues) > 1 || (op != "" && op != "eq" && op != "not" && !(compound && (op == "all" || op == "notAll"))) {
			return "", nil, fmt.Errorf("Ungültiger Filteroperator für %s", field)
		}
		negative := op == "not" || op == "notAll"
		values := q[field]
		if len(values) == 0 {
			continue
		}
		if compound {
			var parts []string
			for _, v := range values {
				if v == "__unknown__" || v == "" {
					if field == "audio" || field == "language" {
						parts = append(parts, "(f."+field+"='' OR EXISTS(SELECT 1 FROM json_each(f.data,'$.unknownTraits') u WHERE u.value=?))")
						a = append(a, field)
					} else {
						parts = append(parts, "f."+field+"=?")
						a = append(a, "")
					}
					continue
				}
				if field == "language" {
					parts = append(parts, "(f.language=? OR instr(','||replace(f.language,', ', ',')||',', ','||?||',')>0)")
				} else {
					parts = append(parts, "(f."+field+"=? OR instr(' + '||f."+field+"||' + ', ' + '||?||' + ')>0)")
				}
				a = append(a, v, v)
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
			return "", nil, fmt.Errorf("Ungültiger Metadatenstatus")
		}
		w = append(w, "coalesce(json_extract(f.data,'$.metadataIncomplete'),0)=?")
		a = append(a, state == "incomplete")
	}
	if c := q.Get("conflict"); c != "" {
		if c != "true" && c != "false" {
			return "", nil, fmt.Errorf("Ungültiger Konfliktfilter")
		}
		w = append(w, "f.conflict=?")
		a = append(a, c == "true")
	}
	if q.Get("yearMin") != "" || q.Get("yearMax") != "" {
		ys := []string{"y.file_rowid=f.rowid"}
		for _, part := range []struct{ key, op string }{{"yearMin", ">="}, {"yearMax", "<="}} {
			if v := q.Get(part.key); v != "" {
				n, e := strconv.Atoi(v)
				if e != nil || n < 1800 || n > 3000 {
					return "", nil, fmt.Errorf("Ungültiges Jahr")
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
				return "", nil, fmt.Errorf("Ungültige Größe")
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
	filter, args, e := where(q)
	if e != nil {
		return out, e
	}
	if v := q.Get("page"); v != "" {
		out.Page, e = strconv.Atoi(v)
		if e != nil || out.Page < 1 || out.Page > 100000 {
			return out, fmt.Errorf("Ungültige Seite")
		}
	}
	tx, e := s.DB.BeginTx(ctx, nil)
	if e != nil {
		return out, e
	}
	defer tx.Rollback()
	if e = tx.QueryRowContext(ctx, "SELECT count(*) FROM files f WHERE "+filter, args...).Scan(&out.Total); e != nil {
		return out, e
	}
	out.DisplayTotal = out.Total
	display := filter
	da := append([]any{}, args...)
	if mode == "season" && q.Get("service") == "sonarr" && !all {
		display = "f.service=? AND (f.media_id,f.season) IN (SELECT f.media_id,f.season FROM files f WHERE " + filter + ")"
		da = append([]any{"sonarr"}, args...)
		if e = tx.QueryRowContext(ctx, "SELECT count(*) FROM files f WHERE "+display, da...).Scan(&out.DisplayTotal); e != nil {
			return out, e
		}
	}
	order, e := displayOrder(q)
	if e != nil {
		return out, e
	}
	query := "SELECT f.data,(" + filter + ") FROM files f WHERE " + display + " ORDER BY " + order
	qa := append(append([]any{}, args...), da...)
	if !all {
		query += " LIMIT ? OFFSET ?"
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
		if e = rows.Scan(&raw, &matched); e != nil {
			return out, e
		}
		var f File
		if e = json.Unmarshal([]byte(raw), &f); e != nil {
			return out, e
		}
		f.Matched = matched
		out.Files = append(out.Files, f)
	}
	return out, rows.Err()
}
func (s *Store) facets(ctx context.Context, service string) (map[string][]string, error) {
	if !serviceOK(service) {
		return nil, fmt.Errorf("Ungültiger Dienst")
	}
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	out := map[string][]string{}
	for _, field := range fields {
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
	if err == nil && q.Get("service") == "sonarr" {
		order = "f.series COLLATE NOCASE,f.media_id,f.season,f.episode_sort,f.id"
	}
	return order, err
}
func movieOrder(q url.Values) (string, error) {
	field := q.Get("sort")
	if field == "" {
		field = "title"
	}
	expression, ok := map[string]string{
		"title": "f.title COLLATE NOCASE", "year": "(SELECT min(year) FROM years y WHERE y.file_rowid=f.rowid)",
		"resolution": "CASE f.resolution WHEN '4320p' THEN 4320 WHEN '2160p' THEN 2160 WHEN '1440p' THEN 1440 WHEN '1080p' THEN 1080 WHEN '720p' THEN 720 WHEN '576p' THEN 576 WHEN '480p' THEN 480 ELSE CAST(f.resolution AS INTEGER) END",
		"codec":      "f.codec COLLATE NOCASE", "source": "(f.source||' '||f.processing) COLLATE NOCASE", "size": "f.size", "hdr": "f.hdr COLLATE NOCASE", "bitDepth": "CAST(f.bitDepth AS INTEGER)", "audio": "f.audio COLLATE NOCASE", "channels": "CAST(f.channels AS REAL)", "language": "f.language COLLATE NOCASE",
	}[field]
	if !ok {
		return "", fmt.Errorf("Ungültige Sortierung")
	}
	direction := q.Get("dir")
	if direction == "" {
		direction = "asc"
	}
	if direction != "asc" && direction != "desc" {
		return "", fmt.Errorf("Ungültige Sortierrichtung")
	}
	unknown := "f." + field + "=''"
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
	Movies        int                                     `json:"movies"`
	Series        int                                     `json:"series"`
	Episodes      int                                     `json:"episodes"`
	Files         int                                     `json:"files"`
	Emby          map[string]int                          `json:"emby"`
	Incomplete    int                                     `json:"incomplete"`
	Distributions map[string]map[string][]StatisticBucket `json:"distributions"`
}

// One read transaction keeps widgets and diagrams consistent during a sync.
func (s *Store) statistics(ctx context.Context) (Statistics, error) {
	out := Statistics{Emby: map[string]int{}, Distributions: map[string]map[string][]StatisticBucket{}}
	tx, e := s.DB.BeginTx(ctx, nil)
	if e != nil {
		return out, e
	}
	defer tx.Rollback()
	if e = tx.QueryRowContext(ctx, `SELECT (SELECT count(DISTINCT media_id) FROM files WHERE service='radarr'),(SELECT count(DISTINCT media_id) FROM files WHERE service='sonarr'),count(*) FROM files`).Scan(&out.Movies, &out.Series, &out.Files); e != nil {
		return out, e
	}
	if e = tx.QueryRowContext(ctx, `SELECT count(*) FROM (SELECT DISTINCT f.media_id,f.season,json_extract(ep.value,'$.number') FROM files f,json_each(f.data,'$.episodes') ep WHERE f.service='sonarr')`).Scan(&out.Episodes); e != nil {
		return out, e
	}
	if e = tx.QueryRowContext(ctx, "SELECT count(*) FROM files WHERE json_extract(data,'$.metadataIncomplete')=1").Scan(&out.Incomplete); e != nil {
		return out, e
	}
	rows, e := tx.QueryContext(ctx, "SELECT coalesce(nullif(json_extract(data,'$.embyMatch'),''),'unconfigured'),count(*) FROM files GROUP BY 1")
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
	return out, tx.Commit()
}
