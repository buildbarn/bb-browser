package main

import (
	"html/template"
	"log"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	buildeventstream "github.com/bazelbuild/bazel/src/main/java/com/google/devtools/build/lib/buildeventstream/proto"
	"github.com/buildbarn/bb-browser/pkg/configuration"
	"github.com/buildbarn/bb-storage/pkg/ac"
	blobstore_configuration "github.com/buildbarn/bb-storage/pkg/blobstore/configuration"
	"github.com/buildbarn/bb-storage/pkg/cas"
	"github.com/buildbarn/bb-storage/pkg/util"
	"github.com/gorilla/mux"
	"github.com/kballard/go-shellquote"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatal("Usage: bb_browser bb_browser.conf")
	}

	browserConfiguration, err := configuration.GetBrowserConfiguration(os.Args[1])
	if err != nil {
		log.Fatalf("Failed to read configuration from %s: %s", os.Args[1], err)
	}

	// Storage access.
	contentAddressableStorageBlobAccess, actionCacheBlobAccess, err := blobstore_configuration.CreateBlobAccessObjectsFromConfig(browserConfiguration.Blobstore)
	if err != nil {
		log.Fatal("Failed to create blob access: ", err)
	}

	templates, err := template.New("templates").Funcs(template.FuncMap{
		"basename": path.Base,
		"build_event_file_to_digest": func(in *buildeventstream.File) *util.Digest {
			// Converts URLs of this format to digest objects:
			// bytestream://host/instance/blobs/hash/size
			fileURI, ok := in.File.(*buildeventstream.File_Uri)
			if !ok {
				return nil
			}
			u, err := url.Parse(fileURI.Uri)
			if err != nil || u.Scheme != "bytestream" {
				return nil
			}
			digest, _ := util.NewDigestFromBytestreamPath(u.Path)
			return digest
		},
		"inc": func(n int) int {
			return n + 1
		},
		"shellquote": func(in string) string {
			// Use non-breaking hyphens to improve readability of output.
			return strings.Replace(shellquote.Join(in), "-", "‑", -1)
		},
		"timestamp_millis_rfc3339": func(in int64) string {
			return time.Unix(in/1000, in%1000*1000000).Format(time.RFC3339)
		},
	}).ParseGlob("templates/*")
	if err != nil {
		log.Fatal("Failed to parse templates: ", err)
	}

	router := mux.NewRouter()
	router.Handle("/metrics", promhttp.Handler())
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))
	NewAssetService(router)
	NewBrowserService(
		cas.NewBlobAccessContentAddressableStorage(
			contentAddressableStorageBlobAccess,
			browserConfiguration.MaximumMessageSizeBytes),
		contentAddressableStorageBlobAccess,
		ac.NewBlobAccessActionCache(actionCacheBlobAccess),
		templates,
		router)
	log.Fatal(http.ListenAndServe(browserConfiguration.ListenAddress, router))
}
