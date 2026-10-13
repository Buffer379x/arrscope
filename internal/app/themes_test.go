package app

import (
	"context"
	"testing"
)

func TestInterfaceThemeCompatibilityValidationAndRestart(t *testing.T) {
	cfg := DefaultSettings()
	cfg.InterfaceTheme = ""
	if err := normalizeInterfaceTheme(&cfg); err != nil || cfg.InterfaceTheme != "default" {
		t.Fatal("existing installations must retain Default", err)
	}
	for _, value := range []string{"default", "nord", "catppuccin"} {
		cfg.InterfaceTheme = value
		if err := normalizeInterfaceTheme(&cfg); err != nil {
			t.Fatal(err)
		}
	}
	cfg.InterfaceTheme = "url(https://external.invalid/theme)"
	if normalizeInterfaceTheme(&cfg) == nil {
		t.Fatal("unsupported palette accepted")
	}
	cfg = DefaultSettings()
	cfg.InterfaceTheme = "nord"
	store := testStore(t)
	if err := store.Put("settings", cfg); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	server, err := NewServer(ctx, store, nil, "test")
	if err != nil {
		t.Fatal(err)
	}
	defer server.Sync.Shutdown()
	if server.Config().InterfaceTheme != "nord" {
		t.Fatal("saved palette lost on restart")
	}
}
