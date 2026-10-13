package app

import "errors"

func normalizeInterfaceTheme(cfg *Settings) error {
	if cfg.InterfaceTheme == "" {
		cfg.InterfaceTheme = "default"
	}
	switch cfg.InterfaceTheme {
	case "default", "nord", "catppuccin":
		return nil
	default:
		return errors.New("Invalid interface theme")
	}
}
