package app

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Server struct {
	Store    *Store
	Security *Security
	Sync     *Syncer
	mu       sync.Mutex
	settings Settings
	ctx      context.Context
	Static   fs.FS
	Demo     bool
	Version  string
}

func NewServer(ctx context.Context, store *Store, static fs.FS, version string) (*Server, error) {
	sec, e := NewSecurity(store)
	if e != nil {
		return nil, e
	}
	s := &Server{Store: store, Security: sec, Sync: NewSyncer(store), settings: DefaultSettings(), ctx: ctx, Static: static, Version: version}
	var cfg Settings
	e = store.Get("settings", &cfg)
	if e == nil {
		cfg.Radarr.APIKey, e = sec.Open(cfg.Radarr.APIKey)
		if e != nil {
			return nil, e
		}
		cfg.Sonarr.APIKey, e = sec.Open(cfg.Sonarr.APIKey)
		if e != nil {
			return nil, e
		}
		cfg.Emby.APIKey, e = sec.Open(cfg.Emby.APIKey)
		if e != nil {
			return nil, e
		}
		if cfg.TagColors == nil {
			cfg.TagColors = map[string]string{}
		}
		if e := normalizeVisibleColumns(&cfg); e != nil {
			return nil, e
		}
		s.settings = cfg
	}
	go s.Sync.Schedule(ctx, s.scheduleDue)
	return s, nil
}
func (s *Server) Config() Settings { s.mu.Lock(); defer s.mu.Unlock(); return s.settings }

// Keep configuration selection and sync dispatch under the same lock as updates.
func (s *Server) scheduleDue() {
	s.mu.Lock()
	defer s.mu.Unlock()
	statuses := s.Sync.Statuses()
	for service, cfg := range map[string]ServiceConfig{"radarr": s.settings.Radarr, "sonarr": s.settings.Sonarr} {
		st := statuses[service]
		if !cfg.Enabled || cfg.Interval == 0 || st.Running {
			continue
		}
		if st.LastAttempt == nil || time.Since(*st.LastAttempt) >= time.Duration(cfg.Interval)*time.Minute {
			s.Sync.Start(s.ctx, service, cfg, s.settings.Emby)
		}
	}
}
func jsonOut(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
func failure(w http.ResponseWriter, status int, e error) {
	jsonOut(w, status, map[string]string{"error": e.Error()})
}
func decode(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		return errors.New("Ungültige Eingabe")
	}
	var x any
	if d.Decode(&x) != io.EOF {
		return errors.New("Ungültige Eingabe")
	}
	return nil
}
func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; script-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
		if r.URL.Path == "/healthz" {
			jsonOut(w, 200, map[string]string{"status": "ok", "version": s.Version})
			return
		}
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			if r.Method != http.MethodGet && r.Method != http.MethodHead {
				http.Error(w, "Method not allowed", 405)
				return
			}
			if s.Static == nil {
				http.NotFound(w, r)
				return
			}
			path := strings.TrimPrefix(r.URL.Path, "/")
			if path == "" {
				path = "index.html"
			}
			if _, e := fs.Stat(s.Static, path); e != nil {
				http.NotFound(w, r)
				return
			}
			http.FileServer(http.FS(s.Static)).ServeHTTP(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		mutating := r.Method != "GET" && r.Method != "HEAD"
		if mutating {
			if origin := r.Header.Get("Origin"); origin != "" {
				u, e := url.Parse(origin)
				if e != nil || u.Host != r.Host {
					failure(w, 403, errors.New("Ungültiger Origin"))
					return
				}
			}
		}
		hash, csrf, authenticated := s.Security.Session(r)
		if r.URL.Path == "/api/status" && r.Method == "GET" {
			out := map[string]any{"initialized": s.Security.Initialized(), "authenticated": authenticated, "version": s.Version, "demo": s.Demo}
			if authenticated {
				out["csrf"] = csrf
				out["sync"] = s.Sync.Statuses()
			}
			jsonOut(w, 200, out)
			return
		}
		if (r.URL.Path == "/api/setup" || r.URL.Path == "/api/login") && r.Method == "POST" {
			var in struct {
				Username string `json:"username"`
				Password string `json:"password"`
				Token    string `json:"token"`
			}
			if e := decode(w, r, &in); e != nil {
				failure(w, 400, e)
				return
			}
			if r.URL.Path == "/api/setup" {
				if e := s.Security.Setup(in.Username, in.Password, in.Token); e != nil {
					failure(w, 400, e)
					return
				}
				if e := s.Security.CreateSession(w, r); e != nil {
					failure(w, 500, errors.New("Sitzung konnte nicht erstellt werden"))
					return
				}
			} else {
				valid, err := s.Security.LoginSession(w, r, in.Username, in.Password, remoteIP(r))
				if err != nil {
					failure(w, 500, errors.New("Sitzung konnte nicht erstellt werden"))
					return
				}
				if !valid {
					failure(w, 401, errors.New("Anmeldung fehlgeschlagen oder vorübergehend gesperrt"))
					return
				}
			}
			jsonOut(w, 200, map[string]bool{"ok": true})
			return
		}
		if !authenticated {
			failure(w, 401, errors.New("Bitte anmelden"))
			return
		}
		if mutating && subtle.ConstantTimeCompare([]byte(r.Header.Get("X-CSRF-Token")), []byte(csrf)) != 1 {
			failure(w, 403, errors.New("Ungültiger CSRF-Token"))
			return
		}
		switch r.URL.Path {
		case "/api/logout":
			if r.Method != "POST" {
				break
			}
			s.Store.DB.Exec("DELETE FROM sessions WHERE token=?", hash)
			http.SetCookie(w, &http.Cookie{Name: "arrscope_session", Value: "", Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: -1})
			jsonOut(w, 200, map[string]bool{"ok": true})
			return
		case "/api/settings":
			if r.Method == "GET" {
				cfg := s.Config()
				for _, connection := range cfg.Connections() {
					connection.HasKey = connection.APIKey != ""
					connection.APIKey = ""
				}
				jsonOut(w, 200, cfg)
				return
			}
			if r.Method == "PUT" {
				var cfg Settings
				if e := decode(w, r, &cfg); e != nil {
					failure(w, 400, e)
					return
				}
				if e := validateTagColors(cfg.TagColors); e != nil {
					failure(w, 400, e)
					return
				}
				if e := normalizeVisibleColumns(&cfg); e != nil {
					failure(w, 400, e)
					return
				}
				if cfg.TagColors == nil {
					cfg.TagColors = map[string]string{}
				}
				if cfg.SeasonMode != "matches" && cfg.SeasonMode != "season" {
					failure(w, 400, errors.New("Ungültige Serienansicht"))
					return
				}
				s.mu.Lock()
				old := s.settings
				for i, connection := range cfg.Connections() {
					if err := mergeConnection(connection, *old.Connections()[i]); err != nil {
						s.mu.Unlock()
						failure(w, 400, err)
						return
					}
				}
				if s.Demo && (cfg.Radarr != old.Radarr || cfg.Sonarr != old.Sonarr || cfg.Emby != old.Emby) {
					s.mu.Unlock()
					failure(w, 403, errors.New("Demo-Verbindungen können nicht geändert werden"))
					return
				}
				if cfg.Emby.Interval != 0 {
					s.mu.Unlock()
					failure(w, 400, errors.New("Emby wird im Intervall von Radarr/Sonarr synchronisiert"))
					return
				}
				stored := cfg
				for _, connection := range stored.Connections() {
					connection.APIKey = s.Security.Seal(connection.APIKey)
				}
				if e := s.Store.Put("settings", stored); e != nil {
					s.mu.Unlock()
					failure(w, 500, errors.New("Einstellungen konnten nicht gespeichert werden"))
					return
				}
				for service, pair := range map[string][2]ServiceConfig{"radarr": {old.Radarr, cfg.Radarr}, "sonarr": {old.Sonarr, cfg.Sonarr}} {
					if pair[0] != pair[1] || old.Emby != cfg.Emby {
						s.Sync.Invalidate(service)
					}
				}
				s.settings = cfg
				for service, pair := range map[string][2]ServiceConfig{"radarr": {old.Radarr, cfg.Radarr}, "sonarr": {old.Sonarr, cfg.Sonarr}} {
					if pair[1].Enabled && (!pair[0].Enabled || pair[0].URL != pair[1].URL || pair[0].APIKey != pair[1].APIKey || old.Emby != cfg.Emby) {
						s.Sync.Start(s.ctx, service, pair[1], cfg.Emby)
					}
				}
				s.mu.Unlock()
				jsonOut(w, 200, map[string]bool{"ok": true})
				return
			}
		case "/api/test":
			if r.Method != "POST" {
				break
			}
			var in struct {
				Service string `json:"service"`
				URL     string `json:"url"`
				APIKey  string `json:"apiKey"`
			}
			if e := decode(w, r, &in); e != nil {
				failure(w, 400, e)
				return
			}
			if (!serviceOK(in.Service) && in.Service != "emby") || s.Demo {
				failure(w, 400, errors.New("Ungültiger Dienst oder Demo-Modus"))
				return
			}
			cfg := s.Config().Radarr
			if in.Service == "sonarr" {
				cfg = s.Config().Sonarr
			}
			if in.Service == "emby" {
				cfg = s.Config().Emby
			}
			if in.APIKey != "" {
				cfg.APIKey = in.APIKey
			}
			cfg.URL = in.URL
			cfg.Enabled = true
			if e := ValidateConfig(cfg); e != nil {
				failure(w, 400, e)
				return
			}
			var version string
			var e error
			if in.Service == "emby" {
				version, e = NewEmby(cfg).Test(r.Context())
			} else {
				version, e = NewArr(cfg).Test(r.Context())
			}
			if e != nil {
				failure(w, 502, e)
				return
			}
			jsonOut(w, 200, map[string]string{"version": version})
			return
		case "/api/sync":
			if r.Method != "POST" {
				break
			}
			var in struct {
				Service string `json:"service"`
			}
			if e := decode(w, r, &in); e != nil {
				failure(w, 400, e)
				return
			}
			if s.Demo {
				failure(w, 400, errors.New("Demo-Daten benötigen keinen Sync"))
				return
			}
			s.mu.Lock()
			cfg := s.settings.Radarr
			if in.Service == "sonarr" {
				cfg = s.settings.Sonarr
			}
			e := s.Sync.Start(s.ctx, in.Service, cfg, s.settings.Emby)
			s.mu.Unlock()
			if e != nil {
				failure(w, 409, e)
				return
			}
			jsonOut(w, 202, map[string]bool{"ok": true})
			return
		case "/api/files":
			if r.Method != "GET" {
				break
			}
			mode := r.URL.Query().Get("mode")
			if mode == "" {
				mode = s.Config().SeasonMode
			}
			if mode != "matches" && mode != "season" {
				failure(w, 400, errors.New("Ungültiger Ansichtsmodus"))
				return
			}
			out, e := s.Store.QueryContext(r.Context(), r.URL.Query(), mode, false)
			if e != nil {
				failure(w, 400, e)
				return
			}
			jsonOut(w, 200, out)
			return
		case "/api/statistics":
			if r.Method != "GET" {
				break
			}
			out, e := s.Store.StatisticsContext(r.Context())
			if e != nil {
				failure(w, 500, errors.New("Statistiken konnten nicht geladen werden"))
				return
			}
			jsonOut(w, 200, out)
			return
		case "/api/facets":
			if r.Method != "GET" {
				break
			}
			out, e := s.Store.FacetsContext(r.Context(), r.URL.Query().Get("service"))
			if e != nil {
				failure(w, 400, e)
				return
			}
			jsonOut(w, 200, out)
			return
		case "/api/export":
			if r.Method != "GET" {
				break
			}
			s.export(w, r)
			return
		case "/api/presets":
			if r.Method == "GET" {
				out, e := s.Store.Presets()
				if e != nil {
					failure(w, 500, errors.New("Presets konnten nicht geladen werden"))
					return
				}
				jsonOut(w, 200, out)
				return
			}
			if r.Method == "POST" {
				var p Preset
				if e := decode(w, r, &p); e != nil {
					failure(w, 400, e)
					return
				}
				q, e := url.ParseQuery(p.Query)
				if e != nil {
					failure(w, 400, errors.New("Ungültiger Filter"))
					return
				}
				q.Set("service", p.Service)
				if _, _, e = where(q); e != nil || strings.TrimSpace(p.Name) == "" || len(p.Name) > 80 {
					failure(w, 400, errors.New("Ungültiges Preset"))
					return
				}
				q.Del("page")
				p.Query = q.Encode()
				res, e := s.Store.DB.Exec("INSERT INTO presets(name,service,query) VALUES(?,?,?)", strings.TrimSpace(p.Name), p.Service, p.Query)
				if e != nil {
					failure(w, 500, errors.New("Preset konnte nicht gespeichert werden"))
					return
				}
				p.ID, _ = res.LastInsertId()
				jsonOut(w, 201, p)
				return
			}
			if r.Method == "DELETE" {
				id, e := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
				if e != nil {
					failure(w, 400, errors.New("Ungültige ID"))
					return
				}
				if _, e = s.Store.DB.Exec("DELETE FROM presets WHERE id=?", id); e != nil {
					failure(w, 500, errors.New("Preset konnte nicht gelöscht werden"))
					return
				}
				jsonOut(w, 200, map[string]bool{"ok": true})
				return
			}
		case "/api/password":
			if r.Method != "PUT" {
				break
			}
			var in struct {
				Current  string `json:"current"`
				Password string `json:"password"`
			}
			if e := decode(w, r, &in); e != nil {
				failure(w, 400, e)
				return
			}
			var c credentials
			s.Store.Get("admin", &c)
			verifiedHash := s.Security.loginHash(c.Username, in.Current, remoteIP(r))
			if verifiedHash == "" {
				failure(w, 401, errors.New("Aktuelles Passwort ist ungültig"))
				return
			}
			if s.Demo {
				failure(w, 403, errors.New("Demo-Passwort kann nicht geändert werden"))
				return
			}
			if e := s.Security.reset(in.Password, verifiedHash); e != nil {
				failure(w, 400, e)
				return
			}
			jsonOut(w, 200, map[string]bool{"ok": true})
			return
		case "/api/backup":
			if r.Method != "POST" {
				break
			}
			if e := s.Store.Backup(); e != nil {
				failure(w, 500, errors.New("Sicherung konnte nicht erstellt werden"))
				return
			}
			jsonOut(w, 200, map[string]bool{"ok": true})
			return
		}
		failure(w, 405, errors.New("Endpunkt oder Methode nicht unterstützt"))
	})
}
func safeCell(s string) string {
	v := strings.TrimLeft(s, " \t\r\n")
	if strings.HasPrefix(v, "=") || strings.HasPrefix(v, "+") || strings.HasPrefix(v, "-") || strings.HasPrefix(v, "@") {
		return "'" + s
	}
	return s
}
func (s *Server) export(w http.ResponseWriter, r *http.Request) {
	filter, args, e := where(r.URL.Query())
	if e != nil {
		failure(w, 400, e)
		return
	}
	order, e := displayOrder(r.URL.Query())
	if e != nil {
		failure(w, 400, e)
		return
	}
	rows, e := s.Store.DB.QueryContext(r.Context(), "SELECT f.data FROM files f WHERE "+filter+" ORDER BY "+order, args...)
	if e != nil {
		failure(w, 400, e)
		return
	}
	defer rows.Close()
	temporary, e := os.CreateTemp(s.Store.Dir, ".export-*.csv")
	if e != nil {
		failure(w, 500, errors.New("Export konnte nicht vorbereitet werden"))
		return
	}
	defer func() { temporary.Close(); os.Remove(temporary.Name()) }()
	if e = s.writeCSV(r.Context(), temporary, rows); e != nil {
		failure(w, 500, errors.New("Export konnte nicht vollständig erstellt werden"))
		return
	}
	if _, e = temporary.Seek(0, 0); e != nil {
		failure(w, 500, errors.New("Export konnte nicht gelesen werden"))
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="arrscope-`+r.URL.Query().Get("service")+`-`+time.Now().Format("2006-01-02")+`.csv"`)
	if _, e = io.Copy(w, temporary); e != nil {
		return
	}
}

func (s *Server) writeCSV(ctx context.Context, destination io.Writer, rows *sql.Rows) error {
	if _, e := destination.Write([]byte("\xef\xbb\xbf")); e != nil {
		return e
	}
	cw := csv.NewWriter(destination)
	cw.Comma = ';'
	cw.Write([]string{"Titel", "Serie", "Staffel", "Jahre", "Dateiname", "Auflösung", "Codec", "Codec-Familie", "Source", "Verarbeitung", "HDR", "Bit-Tiefe", "Audio", "Kanäle", "Sprache", "Bytes", "Konflikt", "Tonspuren", "Emby-Zuordnung"})
	for rows.Next() {
		var raw string
		if e := ctx.Err(); e != nil {
			return e
		}
		if e := rows.Scan(&raw); e != nil {
			return e
		}
		var f File
		if e := json.Unmarshal([]byte(raw), &f); e != nil {
			return e
		}
		var years []string
		for _, y := range f.Years {
			years = append(years, strconv.Itoa(y))
		}
		record := []string{f.Title, f.Series, strconv.Itoa(f.Season), strings.Join(years, ", "), f.Filename, f.Resolution, f.Codec, f.Family, f.Source, f.Processing, f.HDR, f.BitDepth, f.Audio, f.Channels, f.Language, strconv.FormatInt(f.Size, 10), strconv.FormatBool(f.Conflict), audioTrackSummary(f.AudioTracks), f.EmbyMatch}
		for i, v := range record {
			record[i] = safeCell(v)
		}
		if e := cw.Write(record); e != nil {
			return e
		}
	}
	if e := rows.Err(); e != nil {
		return e
	}
	cw.Flush()
	return cw.Error()
}
