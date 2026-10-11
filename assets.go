package scopearr

import (
	"embed"
	"io/fs"
	"strings"
)

//go:embed web/dist
var assets embed.FS

func Assets() fs.FS {
	f, err := fs.Sub(assets, "web/dist")
	if err != nil {
		panic(err)
	}
	return f
}

//go:embed VERSION
var projectVersion string

// Version is the source version, shared by local builds and release validation.
func Version() string { return strings.TrimSpace(projectVersion) }

//go:embed web/src/locales/*.json
var localeAssets embed.FS

func Locales() fs.FS {
	f, err := fs.Sub(localeAssets, "web/src/locales")
	if err != nil {
		panic(err)
	}
	return f
}
