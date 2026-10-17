package app

import (
	"errors"
	"net/http"
)

func (s *Server) tagImportHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		failure(w, r, http.StatusMethodNotAllowed, errors.New("Endpoint or method not supported"))
		return
	}
	var in struct {
		Service    string `json:"service"`
		MediaID    int    `json:"mediaId"`
		RemoteID   int    `json:"remoteId"`
		Resolution string `json:"resolution"`
	}
	if e := decode(w, r, &in); e != nil || !serviceOK(in.Service) || s.Demo {
		failure(w, r, http.StatusBadRequest, errors.New("Invalid service or demo mode"))
		return
	}
	cfg := s.Config()
	c := cfg.Radarr
	if in.Service == "sonarr" {
		c = cfg.Sonarr
	}
	if !c.Enabled || !c.TagImport {
		failure(w, r, http.StatusBadRequest, errors.New("Enable external tag import first"))
		return
	}
	if in.Resolution != "" {
		if in.Resolution != "local" && in.Resolution != "remote" {
			failure(w, r, http.StatusBadRequest, errors.New("Invalid conflict resolution"))
			return
		}
		if e := s.Store.ResolveExternal(r.Context(), in.Service, c, in.MediaID, in.RemoteID, in.Resolution == "local"); e != nil {
			failure(w, r, http.StatusConflict, e)
			return
		}
	}
	s.Tags.Queue(in.Service)
	jsonOut(w, http.StatusAccepted, map[string]bool{"queued": true})
}

func (s *Server) syncHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		failure(w, r, http.StatusMethodNotAllowed, errors.New("Endpoint or method not supported"))
		return
	}
	var in struct {
		Service string `json:"service"`
	}
	if e := decode(w, r, &in); e != nil {
		failure(w, r, http.StatusBadRequest, e)
		return
	}
	if s.Demo {
		failure(w, r, http.StatusBadRequest, errors.New("Demo data does not need syncing"))
		return
	}
	if mediaProviderOK(in.Service) {
		s.settingsMu.Lock()
		cfg := s.Config()
		statuses := s.Sync.Statuses()
		mediaConfig := cfg.Emby
		if in.Service == "jellyfin" {
			mediaConfig = cfg.Jellyfin
		}
		if !mediaConfig.Enabled || (!cfg.Radarr.Enabled && !cfg.Sonarr.Enabled) {
			s.settingsMu.Unlock()
			failure(w, r, http.StatusConflict, errors.New("Enable the selected media server and at least one library before matching"))
			return
		}
		for _, service := range []string{"radarr", "sonarr"} {
			if statuses[service].Running {
				s.settingsMu.Unlock()
				failure(w, r, http.StatusConflict, errors.New("Sync already running"))
				return
			}
		}
		for service, connection := range map[string]ServiceConfig{"radarr": cfg.Radarr, "sonarr": cfg.Sonarr} {
			if connection.Enabled {
				if e := s.Sync.Start(s.ctx, service, connection, cfg.Emby, cfg.Jellyfin); e != nil {
					s.settingsMu.Unlock()
					failure(w, r, http.StatusConflict, e)
					return
				}
			}
		}
		s.settingsMu.Unlock()
		jsonOut(w, http.StatusAccepted, map[string]bool{"ok": true})
		return
	}
	s.settingsMu.Lock()
	settings := s.Config()
	cfg := settings.Radarr
	if in.Service == "sonarr" {
		cfg = settings.Sonarr
	}
	e := s.Sync.Start(s.ctx, in.Service, cfg, settings.Emby, settings.Jellyfin)
	s.settingsMu.Unlock()
	if e != nil {
		failure(w, r, http.StatusConflict, e)
		return
	}
	jsonOut(w, http.StatusAccepted, map[string]bool{"ok": true})
}
