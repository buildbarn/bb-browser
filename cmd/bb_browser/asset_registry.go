package main

import (
	"crypto/sha256"
	"encoding/hex"
	"mime"
	"net/http"
	"path"
	"path/filepath"

	"github.com/gorilla/mux"
)

type asset struct {
	cacheBustingKey string
	handler         func(w http.ResponseWriter, req *http.Request)
}

var registeredAssets = map[string]asset{}

func registerAsset(filename string, data []byte) {
	hash := sha256.Sum256(data)
	registeredAssets[filename] = asset{
		cacheBustingKey: hex.EncodeToString(hash[:]),
		handler: func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Cache-Control", "max-age=31536000")
			w.Header().Set("Content-Type", mime.TypeByExtension(filepath.Ext(filename)))
			w.Write(data)
		},
	}
}

func init() {
	for filename, data := range AssetsData {
		registerAsset(filename, data)
	}
}

// RegisterAssetEndpoints registers URL endpoints for static resources
// used by bb_browser, such as Bootstrap and jQuery.
//
// The assets are registered in a directory named "/uploads". This is
// done to ensure that paths of static resources don't conflict with
// paths starting with REv2 instance names. "uploads" is a reserved
// keyword, meaning it cannot occur within instance names.
func RegisterAssetEndpoints(router *mux.Router) {
	for filename, asset := range registeredAssets {
		router.HandleFunc(path.Join("/uploads", asset.cacheBustingKey, filename), asset.handler)
	}
}

// GetAssetPath appends a cache busting key to the end of a pathname
// string. The cache busting key is based on a checksum of the file's
// contents.
func GetAssetPath(routePrefix, filename string) string {
	asset := registeredAssets[filename]
	return path.Join(routePrefix, "uploads", asset.cacheBustingKey, filename)
}
