package app

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"time"
)

func (s *Server) settingsHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		cfg := s.Config()
		for _, connection := range cfg.Connections() {
			connection.HasKey = connection.APIKey != ""
			connection.APIKey = ""
		}
		jsonOut(w, http.StatusOK, cfg)
		return
	}
	if r.Method == "PUT" {
		var in struct {
			Settings
			TagApprovals map[string]string `json:"tagApprovals"`
		}
		if e := decode(w, r, &in); e != nil {
			failure(w, r, http.StatusBadRequest, e)
			return
		}
		cfg := in.Settings
		if e := normalizeSettings(&cfg); e != nil {
			failure(w, r, http.StatusBadRequest, e)
			return
		}
		if e := normalizeInterfaceTheme(&cfg); e != nil {
			failure(w, r, http.StatusBadRequest, e)
			return
		}
		if e := normalizeWarningColors(&cfg); e != nil {
			failure(w, r, http.StatusBadRequest, e)
			return
		}
		if e := normalizeTagTextColors(&cfg); e != nil {
			failure(w, r, http.StatusBadRequest, e)
			return
		}
		if e := validateTagColors(cfg.TagColors); e != nil {
			failure(w, r, http.StatusBadRequest, e)
			return
		}
		if e := normalizeVisibleColumns(&cfg); e != nil {
			failure(w, r, http.StatusBadRequest, e)
			return
		}
		if cfg.TagColors == nil {
			cfg.TagColors = map[string]string{}
		}
		if cfg.SeasonMode != "matches" && cfg.SeasonMode != "season" {
			failure(w, r, http.StatusBadRequest, errors.New("Invalid TV view"))
			return
		}

		base := s.Config()
		for i, connection := range cfg.Connections() {
			if e := mergeConnection(connection, *base.Connections()[i]); e != nil {
				failure(w, r, http.StatusBadRequest, e)
				return
			}
		}
		var plans []*tagPlan
		changed := tagPolicyChanged(base.Radarr, cfg.Radarr) || tagPolicyChanged(base.Sonarr, cfg.Sonarr)
		_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(130 * time.Second))
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
		defer cancel()
		if connectionsChanged(base, cfg) {
			select {
			case s.Sync.gate <- struct{}{}:
				defer func() { <-s.Sync.gate }()
			case <-ctx.Done():
				failure(w, r, http.StatusServiceUnavailable, errors.New("Sync is busy. Try again after it completes"))
				return
			}
		}
		s.settingsMu.Lock()
		defer s.settingsMu.Unlock()
		if !reflect.DeepEqual(s.Config(), base) {
			failure(w, r, http.StatusConflict, ErrTagPreviewChanged)
			return
		}
		if changed {
			for service, pair := range map[string][2]ServiceConfig{"radarr": {base.Radarr, cfg.Radarr}, "sonarr": {base.Sonarr, cfg.Sonarr}} {
				before, next := pair[0], pair[1]
				if before.TagSync == next.TagSync && before.TagSyncMode == next.TagSyncMode && before.URL == next.URL {
					continue
				}
				if e := mergeConnection(&next, before); e != nil {
					failure(w, r, http.StatusBadRequest, e)
					return
				}
				if e := validateTagPolicy(before, next); e != nil {
					failure(w, r, http.StatusBadRequest, e)
					return
				}
				plan, e := s.Store.BuildTagPlan(ctx, service, next, &before)
				if e != nil {
					failure(w, r, http.StatusBadRequest, e)
					return
				}
				if plan.Waiting {
					failure(w, r, http.StatusConflict, errors.New("Synchronize file data before reviewing outbound tags"))
					return
				}
				if len(plan.Preview.Rows) > 0 {
					if in.TagApprovals[service] != plan.Preview.Token {
						failure(w, r, http.StatusConflict, ErrTagPreviewChanged)
						return
					}
					plans = append(plans, plan)
				}
			}
		}
		old := base
		if s.Demo && (cfg.Radarr != old.Radarr || cfg.Sonarr != old.Sonarr || cfg.Emby != old.Emby || cfg.Jellyfin != old.Jellyfin) {
			failure(w, r, http.StatusForbidden, errors.New("Demo connections cannot be changed"))
			return
		}
		if invalidMediaConfig(cfg.Emby) || invalidMediaConfig(cfg.Jellyfin) {
			failure(w, r, http.StatusBadRequest, errors.New("Media servers sync at the Radarr/Sonarr interval"))
			return
		}

		for _, pair := range [][2]ServiceConfig{{old.Radarr, cfg.Radarr}, {old.Sonarr, cfg.Sonarr}} {
			if e := validateTagPolicy(pair[0], pair[1]); e != nil {
				failure(w, r, http.StatusBadRequest, e)
				return
			}
		}
		if cfg.Emby.TagSyncMode != "" || cfg.Emby.TagSyncDisable != "" || cfg.Jellyfin.TagSyncMode != "" || cfg.Jellyfin.TagSyncDisable != "" {
			failure(w, r, http.StatusBadRequest, errors.New("Media servers do not support outbound tags"))
			return
		}
		stored := cfg
		for _, connection := range stored.Connections() {
			connection.APIKey = s.Security.Seal(connection.APIKey)
		}
		if e := s.Store.saveSettingsWithTagPolicies(ctx, stored, old, cfg, plans...); e != nil {
			failure(w, r, http.StatusInternalServerError, errors.New("Could not save settings"))
			return
		}
		for service, pair := range map[string][2]ServiceConfig{"radarr": {old.Radarr, cfg.Radarr}, "sonarr": {old.Sonarr, cfg.Sonarr}} {
			if dataConfig(pair[0]) != dataConfig(pair[1]) || (old.Emby != cfg.Emby || old.Jellyfin != cfg.Jellyfin) {
				s.Sync.Invalidate(service)
			}
		}
		s.mu.Lock()
		s.settings = cfg
		s.mu.Unlock()
		for service, pair := range map[string][2]ServiceConfig{"radarr": {old.Radarr, cfg.Radarr}, "sonarr": {old.Sonarr, cfg.Sonarr}} {
			if pair[1].Enabled && (!pair[0].Enabled || pair[0].URL != pair[1].URL || pair[0].APIKey != pair[1].APIKey || (old.Emby != cfg.Emby || old.Jellyfin != cfg.Jellyfin) || pair[0].TagSync != pair[1].TagSync && pair[1].TagSync.Active()) {
				s.Sync.Start(s.ctx, service, pair[1], cfg.Emby, cfg.Jellyfin)
			}
		}
		for service, pair := range map[string][2]ServiceConfig{"radarr": {old.Radarr, cfg.Radarr}, "sonarr": {old.Sonarr, cfg.Sonarr}} {
			if pair[0] != pair[1] {
				s.Tags.Queue(service)
			}
		}
		jsonOut(w, http.StatusOK, map[string]bool{"ok": true})
		return
	}
	failure(w, r, http.StatusMethodNotAllowed, errors.New("Endpoint or method not supported"))

}

func (s *Server) testConnectionHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		failure(w, r, http.StatusMethodNotAllowed, errors.New("Endpoint or method not supported"))
		return
	}
	var in struct {
		Service string `json:"service"`
		URL     string `json:"url"`
		APIKey  string `json:"apiKey"`
	}
	if e := decode(w, r, &in); e != nil {
		failure(w, r, http.StatusBadRequest, e)
		return
	}
	if (!serviceOK(in.Service) && !mediaProviderOK(in.Service)) || s.Demo {
		failure(w, r, http.StatusBadRequest, errors.New("Invalid service or demo mode"))
		return
	}
	cfg := s.Config().Radarr
	if in.Service == "sonarr" {
		cfg = s.Config().Sonarr
	}
	if in.Service == "emby" {
		cfg = s.Config().Emby
	}
	if in.Service == "jellyfin" {
		cfg = s.Config().Jellyfin
	}
	if in.APIKey != "" {
		cfg.APIKey = in.APIKey
	}
	cfg.URL = in.URL
	cfg.Enabled = true
	if e := ValidateConfig(cfg); e != nil {
		failure(w, r, http.StatusBadRequest, e)
		return
	}
	var version string
	var e error
	if in.Service == "emby" {
		version, e = NewEmby(cfg).Test(r.Context())
	} else if in.Service == "jellyfin" {
		version, e = NewJellyfin(cfg).Test(r.Context())
	} else {
		version, e = NewArr(cfg).Test(r.Context())
	}
	if e != nil {
		failure(w, r, http.StatusBadGateway, e)
		return
	}
	jsonOut(w, http.StatusOK, map[string]string{"version": version})
}
