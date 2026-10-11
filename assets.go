package arrscope

import (
	"embed"
	"io/fs"
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
