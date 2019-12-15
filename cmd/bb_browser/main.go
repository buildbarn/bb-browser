package main

import (
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"time"

	buildeventstream "github.com/bazelbuild/bazel/src/main/java/com/google/devtools/build/lib/buildeventstream/proto"
	"github.com/buildbarn/bb-browser/pkg/proto/configuration/bb_browser"
	blobstore_configuration "github.com/buildbarn/bb-storage/pkg/blobstore/configuration"
	"github.com/buildbarn/bb-storage/pkg/cas"
	"github.com/buildbarn/bb-storage/pkg/util"
	"github.com/gorilla/mux"
	"github.com/kballard/go-shellquote"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatal("Usage: bb_browser bb_browser.jsonnet")
	}
	var configuration bb_browser.ApplicationConfiguration
	if err := util.UnmarshalConfigurationFromFile(os.Args[1], &configuration); err != nil {
		log.Fatalf("Failed to read configuration from %s: %s", os.Args[1], err)
	}

	// Storage access.
	contentAddressableStorageBlobAccess, actionCacheBlobAccess, err := blobstore_configuration.CreateBlobAccessObjectsFromConfig(
		configuration.Blobstore,
		int(configuration.MaximumMessageSizeBytes))
	if err != nil {
		log.Fatal("Failed to create blob access: ", err)
	}

	templates, err := template.New("templates").Funcs(template.FuncMap{
		"asset_path": GetAssetPath,
		"basename":   path.Base,
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
		"shellquote": shellquote.Join,
		"timestamp_millis_rfc3339": func(in int64) string {
			return time.Unix(in/1000, in%1000*1000000).Format(time.RFC3339)
		},
	}).ParseGlob("templates/*")
	if err != nil {
		log.Fatal("Failed to parse templates: ", err)
	}

	router := mux.NewRouter()
	NewBrowserService(
		cas.NewBlobAccessContentAddressableStorage(
			contentAddressableStorageBlobAccess,
			int(configuration.MaximumMessageSizeBytes)),
		contentAddressableStorageBlobAccess,
		actionCacheBlobAccess,
		int(configuration.MaximumMessageSizeBytes),
		templates,
		router)
	RegisterAssetEndpoints(router)
	util.RegisterAdministrativeHTTPEndpoints(router)
	log.Fatal(http.ListenAndServe(configuration.ListenAddress, router))
}
