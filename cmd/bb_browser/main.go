package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/buildbarn/bb-browser/pkg/proto/configuration/bb_browser"
	"github.com/buildbarn/bb-remote-execution/pkg/proto/resourceusage"
	blobstore_configuration "github.com/buildbarn/bb-storage/pkg/blobstore/configuration"
	"github.com/buildbarn/bb-storage/pkg/cas"
	"github.com/buildbarn/bb-storage/pkg/global"
	"github.com/buildbarn/bb-storage/pkg/util"
	"github.com/dustin/go-humanize"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/gorilla/mux"
	"github.com/kballard/go-shellquote"
)

const (
	// rfc3339Milli is identical similar to the time.RFC3339 and
	// time.RFC3339Nano formats, except that it shows the time in
	// milliseconds.
	rfc3339Milli = "2006-01-02T15:04:05.999Z07:00"
)

// timestampDelta is returned by the timestamp_proto_delta, returning a
// timestamp and a duration relative to a previous timestamp value. It
// can be used to display split times.
type timestampDelta struct {
	Time                 time.Time
	DurationFromPrevious time.Duration
}

func main() {
	if len(os.Args) != 2 {
		log.Fatal("Usage: bb_browser bb_browser.jsonnet")
	}
	var configuration bb_browser.ApplicationConfiguration
	if err := util.UnmarshalConfigurationFromFile(os.Args[1], &configuration); err != nil {
		log.Fatalf("Failed to read configuration from %s: %s", os.Args[1], err)
	}
	if err := global.ApplyConfiguration(configuration.Global); err != nil {
		log.Fatal("Failed to apply global configuration options: ", err)
	}

	// Storage access.
	contentAddressableStorageBlobAccess, actionCacheBlobAccess, err := blobstore_configuration.CreateBlobAccessObjectsFromConfig(
		configuration.Blobstore,
		int(configuration.MaximumMessageSizeBytes))
	if err != nil {
		log.Fatal("Failed to create blob access: ", err)
	}

	templates := template.New("templates").Funcs(template.FuncMap{
		"asset_path": GetAssetPath,
		"basename":   path.Base,
		"duration_proto": func(pb *duration.Duration) *time.Duration {
			// Converts a Protobuf duration to Go's native type.
			d, err := ptypes.Duration(pb)
			if err != nil {
				return nil
			}
			return &d
		},
		"humanize_bytes": func(v interface{}) string {
			switch i := v.(type) {
			case uint64:
				return humanize.Bytes(i)
			case int64:
				return humanize.Bytes(uint64(i))
			default:
				panic("Unknown type")
			}
		},
		"inc": func(n int) int {
			return n + 1
		},
		"to_file_pool_resource_usage": func(any *any.Any) *resourceusage.FilePoolResourceUsage {
			var pb resourceusage.FilePoolResourceUsage
			if ptypes.UnmarshalAny(any, &pb) != nil {
				return nil
			}
			return &pb
		},
		"to_posix_resource_usage": func(any *any.Any) *resourceusage.POSIXResourceUsage {
			var pb resourceusage.POSIXResourceUsage
			if ptypes.UnmarshalAny(any, &pb) != nil {
				return nil
			}
			return &pb
		},
		"shellquote": shellquote.Join,
		"timestamp_rfc3339": func(t time.Time) string {
			// Converts a timestamp to RFC3339 format.
			return t.Format(rfc3339Milli)
		},
		"timestamp_proto_delta": func(pbPrevious *timestamp.Timestamp, pbNow *timestamp.Timestamp) *timestampDelta {
			tNow, err := ptypes.Timestamp(pbNow)
			if err != nil {
				return nil
			}
			tPrevious, err := ptypes.Timestamp(pbPrevious)
			if err != nil {
				// Time may be parsed, but no split time
				// is available.
				return &timestampDelta{
					Time: tNow,
				}
			}
			if tNow.Equal(tPrevious) {
				// Don't display the split time, as
				// there is no difference.
				return nil
			}
			return &timestampDelta{
				Time:                 tNow,
				DurationFromPrevious: tNow.Sub(tPrevious),
			}
		},
		"timestamp_proto_rfc3339": func(pb *timestamp.Timestamp) string {
			// Converts a Protobuf timestamp to RFC 3339 format.
			t, err := ptypes.Timestamp(pb)
			if err != nil {
				return ""
			}
			return t.Format(rfc3339Milli)
		},
	})

	for name, template := range TemplatesData {
		templates, err = templates.New(name).Parse(template)
		if err != nil {
			log.Fatalf("Failed to parse template %#v: %s", name, err)
		}
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
