package agentassets

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed all:web/dist all:photos
var assets embed.FS

type emptyFS struct{}

func (emptyFS) Open(string) (http.File, error) {
	return nil, fs.ErrNotExist
}

func staticFS(root string) http.FileSystem {
	sub, err := fs.Sub(assets, root)
	if err != nil {
		return emptyFS{}
	}
	return http.FS(sub)
}

func FrontendAssetsFS() http.FileSystem {
	return staticFS("web/dist/app-assets")
}

func PhotosFS() http.FileSystem {
	return staticFS("photos")
}

func FrontendIndex() ([]byte, error) {
	return fs.ReadFile(assets, "web/dist/index.html")
}
