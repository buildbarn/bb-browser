package main

import (
	"flag"
	"html/template"
	"log"
	"net/http"
	_ "net/http/pprof"
	"path"
	"strings"

	"github.com/buildbarn/bb-storage/pkg/ac"
	"github.com/buildbarn/bb-storage/pkg/blobstore/configuration"
	"github.com/buildbarn/bb-storage/pkg/cas"
	"github.com/gorilla/mux"
	"github.com/kballard/go-shellquote"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	var (
		blobstoreConfig    = flag.String("blobstore-config", "/config/blobstore.conf", "Configuration for blob storage")
		maximumMessageSize = flag.Int64("maximum-message-size", 16*1024*1024, "Maximum Protobuf message size to unmarshal")
		webListenAddress   = flag.String("web.listen-address", ":80", "Port on which to expose metrics")
	)
	flag.Parse()

	// Storage access.
	contentAddressableStorageBlobAccess, actionCacheBlobAccess, err := configuration.CreateBlobAccessObjectsFromConfig(*blobstoreConfig)
	if err != nil {
		log.Fatal("Failed to create blob access: ", err)
	}

	templates, err := template.New("templates").Funcs(template.FuncMap{
		"basename": path.Base,
		"shellquote": func(in string) string {
			// Use non-breaking hyphens to improve readability of output.
			return strings.Replace(shellquote.Join(in), "-", "‑", -1)
		},
	}).ParseGlob("templates/*")
	if err != nil {
		log.Fatal("Failed to parse templates: ", err)
	}

	router := mux.NewRouter()
	router.Handle("/metrics", promhttp.Handler())
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))
	NewBrowserService(
		cas.NewBlobAccessContentAddressableStorage(
			contentAddressableStorageBlobAccess,
			*maximumMessageSize),
		contentAddressableStorageBlobAccess,
		ac.NewBlobAccessActionCache(actionCacheBlobAccess),
		templates,
		router)
	log.Fatal(http.ListenAndServe(*webListenAddress, router))
}
