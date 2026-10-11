package app

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

type ServiceConfig struct {
	Enabled      bool   `json:"enabled"`
	URL          string `json:"url"`
	APIKey       string `json:"apiKey,omitempty"`
	HasKey       bool   `json:"hasKey"`
	ClearKey     bool   `json:"clearKey,omitempty"`
	PathMappings string `json:"pathMappings,omitempty"`
	Interval     int    `json:"interval"`
}
type Settings struct {
	ColumnOrder    map[string][]string `json:"columnOrder"`
	VisibleColumns map[string][]string `json:"visibleColumns"`
	TagColors      map[string]string   `json:"tagColors"`
	Radarr         ServiceConfig       `json:"radarr"`
	Sonarr         ServiceConfig       `json:"sonarr"`
	Emby           ServiceConfig       `json:"emby"`
	SeasonMode     string              `json:"seasonMode"`
}
type Evidence struct {
	Filename string `json:"filename"`
	API      string `json:"api"`
	Emby     string `json:"emby,omitempty"`
	Origin   string `json:"origin"`
	Conflict bool   `json:"conflict"`
}
type Episode struct {
	ID     int    `json:"id"`
	Number int    `json:"number"`
	Title  string `json:"title"`
	Year   int    `json:"year"`
}
type AudioTrack struct {
	Index         int    `json:"index"`
	Language      string `json:"language"`
	Codec         string `json:"codec"`
	Channels      int    `json:"channels"`
	ChannelLayout string `json:"channelLayout"`
	Title         string `json:"title"`
	Default       bool   `json:"default"`
}
type File struct {
	ID                 int                 `json:"id"`
	Service            string              `json:"service"`
	MediaID            int                 `json:"mediaId"`
	Title              string              `json:"title"`
	Series             string              `json:"series"`
	Season             int                 `json:"season"`
	Episodes           []Episode           `json:"episodes"`
	Years              []int               `json:"years"`
	Filename           string              `json:"filename"`
	SourcePath         string              `json:"-"`
	Resolution         string              `json:"resolution"`
	Codec              string              `json:"codec"`
	Family             string              `json:"family"`
	Source             string              `json:"source"`
	Processing         string              `json:"processing"`
	HDR                string              `json:"hdr"`
	BitDepth           string              `json:"bitDepth"`
	Audio              string              `json:"audio"`
	AudioStreamCount   int                 `json:"audioStreamCount"`
	AudioTracks        []AudioTrack        `json:"audioTracks,omitempty"`
	EmbyMatch          string              `json:"embyMatch,omitempty"`
	Channels           string              `json:"channels"`
	Language           string              `json:"language"`
	Size               int64               `json:"size"`
	Conflict           bool                `json:"conflict"`
	Evidence           map[string]Evidence `json:"evidence"`
	Matched            bool                `json:"matched"`
	UnknownTraits      []string            `json:"unknownTraits,omitempty"`
	MetadataIncomplete bool                `json:"metadataIncomplete"`
}
type SyncStatus struct {
	Running     bool       `json:"running"`
	Queued      bool       `json:"queued"`
	LastAttempt *time.Time `json:"lastAttempt"`
	LastSuccess *time.Time `json:"lastSuccess"`
	Error       string     `json:"error"`
	Count       int        `json:"count"`
}
type Preset struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Service string `json:"service"`
	Query   string `json:"query"`
}

func DefaultSettings() Settings {
	return Settings{ColumnOrder: defaultColumnOrder(), VisibleColumns: defaultVisibleColumns(), TagColors: map[string]string{}, Radarr: ServiceConfig{Interval: 60}, Sonarr: ServiceConfig{Interval: 60}, Emby: ServiceConfig{}, SeasonMode: "matches"}
}

func (s *Settings) Connections() []*ServiceConfig {
	return []*ServiceConfig{&s.Radarr, &s.Sonarr, &s.Emby}
}
func mergeConnection(in *ServiceConfig, old ServiceConfig) error {
	if in.ClearKey {
		if in.Enabled || in.APIKey != "" {
			return fmt.Errorf("Disable the service and clear the input to delete the API key")
		}
		in.APIKey = ""
	} else if in.APIKey == "" {
		in.APIKey = old.APIKey
	}
	in.ClearKey, in.HasKey = false, false
	return ValidateConfig(*in)
}

func defaultColumnOrder() map[string][]string {
	return map[string][]string{
		"radarr": {"title", "year", "resolution", "codec", "source", "size", "hdr", "bitDepth", "audio", "channels", "language"},
		"sonarr": {"title", "year", "resolution", "codec", "source", "size", "hdr", "bitDepth", "audio", "channels", "language"},
	}
}

func defaultVisibleColumns() map[string][]string {
	return map[string][]string{
		"radarr": {"title", "year", "resolution", "codec", "source"},
		"sonarr": {"title", "year", "resolution", "codec", "source"},
	}
}

// Missing selections preserve the original layout for existing installations.
func normalizeVisibleColumns(cfg *Settings) error {
	if cfg.VisibleColumns == nil {
		cfg.VisibleColumns = defaultVisibleColumns()
	}
	allowed := map[string]bool{"title": true, "year": true, "resolution": true, "codec": true, "source": true, "size": true, "hdr": true, "bitDepth": true, "audio": true, "channels": true, "language": true}
	for service, columns := range cfg.VisibleColumns {
		if service != "radarr" && service != "sonarr" {
			return fmt.Errorf("Invalid service for column selection")
		}
		seen := map[string]bool{}
		for _, column := range columns {
			if !allowed[column] || seen[column] {
				return fmt.Errorf("Invalid column selection")
			}
			seen[column] = true
		}
		if !seen["title"] {
			return fmt.Errorf("Title or episode must remain visible")
		}
	}
	for service, columns := range defaultVisibleColumns() {
		if _, exists := cfg.VisibleColumns[service]; !exists {
			cfg.VisibleColumns[service] = columns
		}
	}
	if cfg.ColumnOrder == nil {
		cfg.ColumnOrder = defaultColumnOrder()
	}
	for service, columns := range cfg.ColumnOrder {
		if service != "radarr" && service != "sonarr" {
			return fmt.Errorf("Invalid service for column order")
		}
		if len(columns) != len(allowed) {
			return fmt.Errorf("Column order must include all columns")
		}
		seen := map[string]bool{}
		for _, column := range columns {
			if !allowed[column] || seen[column] {
				return fmt.Errorf("Invalid column order")
			}
			seen[column] = true
		}
	}
	for service, columns := range defaultColumnOrder() {
		if _, exists := cfg.ColumnOrder[service]; !exists {
			cfg.ColumnOrder[service] = columns
		}
	}
	return nil
}

var tagColorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

func validateTagColors(colors map[string]string) error {
	if len(colors) > 512 {
		return fmt.Errorf("Too many tag colors")
	}
	for key, color := range colors {
		field, value, ok := strings.Cut(key, ":")
		valid := field == "resolution" || field == "codec" || field == "source" || field == "processing" || field == "hdr" || field == "audio"
		if !ok || !valid || value == "" || len(key) > 200 || !tagColorPattern.MatchString(color) {
			return fmt.Errorf("Invalid tag color")
		}
	}
	return nil
}
