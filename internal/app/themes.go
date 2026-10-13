package app

import "errors"

func normalizeInterfaceTheme(cfg *Settings) error {
	if cfg.InterfaceTheme == "" {
		cfg.InterfaceTheme = "default"
	}
	switch cfg.InterfaceTheme {
	case "default", "nord", "catppuccin", "graphite", "slate", "ocean", "atlantic", "forest", "sage", "sandstone", "copper", "coffee", "midnight", "ink", "steel", "mist", "linen", "charcoal":
		return nil
	default:
		return errors.New("Invalid interface theme")
	}
}
