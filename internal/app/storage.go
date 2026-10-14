package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type StorageSelection struct {
	Path        string `json:"path"`
	SharedGroup string `json:"sharedGroup"`
}
type WidgetSettings struct {
	Storage map[string][]StorageSelection `json:"storage"`
	Colors  map[string]string             `json:"colors"`
}
type DiskSpace struct {
	Path  string `json:"path"`
	Label string `json:"label"`
	Free  int64  `json:"freeSpace"`
	Total int64  `json:"totalSpace"`
}
type StorageSnapshot struct {
	Disks     []DiskSpace `json:"disks"`
	CheckedAt *time.Time  `json:"checkedAt"`
	SuccessAt *time.Time  `json:"successAt"`
	Error     string      `json:"error"`
	Source    string      `json:"-"`
}
type StorageTotal struct {
	Total int64  `json:"total"`
	Free  int64  `json:"free"`
	Used  int64  `json:"used"`
	Error string `json:"error"`
	Count int    `json:"count"`
}

func defaultWidgets() WidgetSettings {
	return WidgetSettings{Storage: map[string][]StorageSelection{"radarr": {}, "sonarr": {}}, Colors: map[string]string{"used": "#729acb", "free": "#62b69d"}}
}
func normalize070(cfg *Settings) error {
	if cfg.Emby.SyncStart != "" || cfg.Emby.WebhookEnabled {
		return errors.New("Emby follows the Radarr/Sonarr schedule and has no webhook")
	}
	if cfg.SyncTimezone != "" {
		if _, e := time.LoadLocation(cfg.SyncTimezone); e != nil {
			return errors.New("Invalid sync timezone")
		}
	}
	if cfg.Widgets.Storage == nil {
		cfg.Widgets.Storage = defaultWidgets().Storage
	}
	if cfg.Widgets.Colors == nil {
		cfg.Widgets.Colors = defaultWidgets().Colors
	}
	for service, list := range cfg.Widgets.Storage {
		if !serviceOK(service) || len(list) > 128 {
			return errors.New("Invalid storage selection")
		}
		seen := map[string]bool{}
		for i, item := range list {
			list[i].SharedGroup = strings.TrimSpace(item.SharedGroup)
			if item.Path == "" || len(item.Path) > 1024 || seen[item.Path] || len(list[i].SharedGroup) > 80 {
				return errors.New("Invalid storage selection")
			}
			seen[item.Path] = true
		}
	}
	for key, color := range cfg.Widgets.Colors {
		if (key != "used" && key != "free") || !tagColorPattern.MatchString(color) {
			return errors.New("Invalid storage color")
		}
	}
	for k, v := range defaultWidgets().Colors {
		if cfg.Widgets.Colors[k] == "" {
			cfg.Widgets.Colors[k] = v
		}
	}
	return nil
}
func storageTotals(cfg Settings, snap map[string]StorageSnapshot, scope string) StorageTotal {
	result := StorageTotal{}
	type sample struct {
		disk DiskSpace
		at   time.Time
	}
	groups := map[string]sample{}
	for _, service := range []string{"radarr", "sonarr"} {
		if scope != "all" && scope != service {
			continue
		}
		c := cfg.Radarr
		if service == "sonarr" {
			c = cfg.Sonarr
		}
		if !c.Enabled {
			continue
		}
		s := snap[service]
		if len(cfg.Widgets.Storage[service]) == 0 {
			continue
		}
		if s.Error != "" {
			result.Error = "Storage readings are stale or unavailable"
		}
		if s.Source != tagEndpoint(c) || s.SuccessAt == nil {
			result.Error = "Storage readings are stale or unavailable"
			continue
		}
		index := map[string]DiskSpace{}
		for _, d := range s.Disks {
			index[d.Path] = d
		}
		for _, selection := range cfg.Widgets.Storage[service] {
			d, ok := index[selection.Path]
			if !ok {
				result.Error = "A selected storage path is unavailable"
				continue
			}
			key := service + ":" + d.Path
			if selection.SharedGroup != "" {
				key = "shared:" + selection.SharedGroup
			}
			v, exists := groups[key]
			if exists && v.disk.Total != d.Total {
				result.Error = "Shared storage readings have different total capacities"
				return result
			}
			if !exists || s.SuccessAt.After(v.at) {
				groups[key] = sample{d, *s.SuccessAt}
			}
		}
	}
	for _, s := range groups {
		d := s.disk
		if d.Total < 0 || d.Free < 0 || d.Free > d.Total {
			return StorageTotal{Error: "Invalid storage capacity"}
		}
		if d.Total > 9223372036854775807-result.Total {
			return StorageTotal{Error: "Storage capacity exceeds limits"}
		}
		result.Total += d.Total
		result.Free += d.Free
		result.Count++
	}
	result.Used = result.Total - result.Free
	return result
}
func (s *Server) storageHTTP(w http.ResponseWriter, r *http.Request) {
	snap := s.Background.Storage()
	totals := map[string]StorageTotal{}
	for _, scope := range []string{"all", "radarr", "sonarr"} {
		totals[scope] = storageTotals(s.Config(), snap, scope)
	}
	jsonOut(w, 200, map[string]any{"services": snap, "totals": totals})
}
func (s *Server) storageRefreshHTTP(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Service string `json:"service"`
	}
	if e := decode(w, r, &in); e != nil || !serviceOK(in.Service) {
		failure(w, r, 400, errors.New("Invalid service"))
		return
	}
	if s.Demo {
		failure(w, r, 400, errors.New("Demo connections are locked"))
		return
	}
	if e := s.Background.RefreshStorage(r.Context(), in.Service); e != nil {
		failure(w, r, 502, e)
		return
	}
	s.storageHTTP(w, r)
}
func fetchStorage(ctx context.Context, c ServiceConfig) ([]DiskSpace, error) {
	var disks []DiskSpace
	if e := NewArr(c).get(ctx, "diskspace", nil, &disks); e != nil {
		return nil, e
	}
	seen := map[string]bool{}
	for _, d := range disks {
		if d.Path == "" || seen[d.Path] || d.Total < 0 || d.Free < 0 || d.Free > d.Total {
			return nil, fmt.Errorf("Invalid disk-space response")
		}
		seen[d.Path] = true
	}
	return disks, nil
}
