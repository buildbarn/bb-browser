package main

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/url"

	"github.com/buildbarn/bb-browser/cmd/bb_browser/assets"
	"github.com/gorilla/mux"
)

type asset struct {
	cacheBustingKey string
	handler         func(w http.ResponseWriter, req *http.Request)
}

var (
	registeredAssets = map[string]asset{}
)

func registerAsset(path string, body []byte, contentType string) {
	hash := sha256.Sum256(body)
	registeredAssets[path] = asset{
		cacheBustingKey: hex.EncodeToString(hash[:]),
		handler: func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Cache-Control", "max-age=31536000")
			w.Header().Set("Content-Type", contentType)
			w.Write(body)
		},
	}
}

func init() {
	registerAsset("/bootstrap.css", assets.BootstrapCSS, "text/css")
	registerAsset("/bootstrap.js", assets.BootstrapJS, "application/javascript")
	registerAsset("/favicon.png", assets.FaviconPNG, "image/png")
	registerAsset("/jquery.js", assets.JQueryJS, "application/javascript")
	registerAsset("/terminal.css", assets.TerminalCSS, "text/css")
}

// RegisterAssetEndpoints registers URL endpoints for static resources
// used by bb_browser, such as Bootstrap and jQuery.
func RegisterAssetEndpoints(router *mux.Router) {
	for path, asset := range registeredAssets {
		router.HandleFunc(path, asset.handler)
	}
}

// GetAssetPath appends a cache busting key to the end of a pathname
// string. The cache busting key is based on a checksum of the file's
// contents.
func GetAssetPath(path string) string {
	u := url.URL{
		Path:     path,
		RawQuery: registeredAssets[path].cacheBustingKey,
	}
	return u.String()
}
