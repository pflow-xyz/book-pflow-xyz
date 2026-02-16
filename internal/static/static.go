package static

import (
	"embed"
	"io/fs"
)

//go:embed public
var content embed.FS

func Public() (fs.FS, error) {
	return fs.Sub(content, "public")
}
