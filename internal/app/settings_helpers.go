package app

import "database/sql"

func connectionsChanged(before, after Settings) bool {
	return before.Radarr != after.Radarr || before.Sonarr != after.Sonarr || before.Emby != after.Emby || before.Jellyfin != after.Jellyfin
}

func tagPolicyChanged(before, after ServiceConfig) bool {
	return before.TagSync != after.TagSync || before.TagSyncMode != after.TagSyncMode || before.URL != after.URL && after.TagSync.Active()
}

// Callers serialize presentation writes with settingsMu. Config readers never
// wait for recognition work or database writes while holding Server.mu.
func (s *Server) refreshTagSettings() error {
	var stored Settings
	if err := s.Store.Get("settings", &stored); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}
	s.mu.Lock()
	s.settings.TagColors = stored.TagColors
	s.settings.TagTextColors = stored.TagTextColors
	s.settings.HiddenTags = stored.HiddenTags
	s.mu.Unlock()
	return nil
}
