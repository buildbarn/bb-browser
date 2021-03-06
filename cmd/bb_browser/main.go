package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	remoteexecution "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
	"github.com/buildbarn/bb-browser/pkg/proto/configuration/bb_browser"
	"github.com/buildbarn/bb-remote-execution/pkg/proto/resourceusage"
	"github.com/buildbarn/bb-storage/pkg/blobstore"
	blobstore_configuration "github.com/buildbarn/bb-storage/pkg/blobstore/configuration"
	"github.com/buildbarn/bb-storage/pkg/digest"
	"github.com/buildbarn/bb-storage/pkg/global"
	bb_grpc "github.com/buildbarn/bb-storage/pkg/grpc"
	"github.com/buildbarn/bb-storage/pkg/proto/iscc"
	"github.com/buildbarn/bb-storage/pkg/util"
	"github.com/dustin/go-humanize"
	"github.com/gorilla/mux"
	"github.com/kballard/go-shellquote"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"
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
	lifecycleState, err := global.ApplyConfiguration(configuration.Global)
	if err != nil {
		log.Fatal("Failed to apply global configuration options: ", err)
	}

	// Storage access.
	contentAddressableStorage, actionCache, err := blobstore_configuration.NewCASAndACBlobAccessFromConfiguration(
		configuration.Blobstore,
		bb_grpc.DefaultClientFactory,
		int(configuration.MaximumMessageSizeBytes))
	if err != nil {
		log.Fatal(err)
	}
	var initialSizeClassCache blobstore.BlobAccess
	if configuration.InitialSizeClassCache == nil {
		initialSizeClassCache = blobstore.NewErrorBlobAccess(status.Error(codes.NotFound, "No Initial Size Class Cache configured"))
	} else {
		info, err := blobstore_configuration.NewBlobAccessFromConfiguration(
			configuration.InitialSizeClassCache,
			blobstore_configuration.NewISCCBlobAccessCreator(
				bb_grpc.DefaultClientFactory,
				int(configuration.MaximumMessageSizeBytes)))
		if err != nil {
			log.Fatal("Failed to create Initial Size Class Cache: ", err)
		}
		initialSizeClassCache = info.BlobAccess
	}

	routePrefix := path.Join("/", configuration.RoutePrefix)
	if !strings.HasSuffix(routePrefix, "/") {
		routePrefix += "/"
	}

	templates := template.New("templates").Funcs(template.FuncMap{
		"asset_path": func(filename string) string {
			return GetAssetPath(routePrefix, filename)
		},
		"basename": path.Base,
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
		"to_outcome_failed": func(previousExecution *iscc.PreviousExecution) bool {
			_, ok := previousExecution.Outcome.(*iscc.PreviousExecution_Failed)
			return ok
		},
		"to_outcome_timed_out": func(previousExecution *iscc.PreviousExecution) *time.Duration {
			if outcome, ok := previousExecution.Outcome.(*iscc.PreviousExecution_TimedOut); ok {
				if outcome.TimedOut.CheckValid() == nil {
					d := outcome.TimedOut.AsDuration()
					return &d
				}
			}
			return nil
		},
		"to_outcome_succeeded": func(previousExecution *iscc.PreviousExecution) *time.Duration {
			if outcome, ok := previousExecution.Outcome.(*iscc.PreviousExecution_Succeeded); ok {
				if outcome.Succeeded.CheckValid() == nil {
					d := outcome.Succeeded.AsDuration()
					return &d
				}
			}
			return nil
		},
		"to_build_executor_resource_usage": func(any *anypb.Any) *resourceusage.BuildExecutorResourceUsage {
			var pb resourceusage.BuildExecutorResourceUsage
			if err := any.UnmarshalTo(&pb); err != nil {
				return nil
			}
			return &pb
		},
		"to_monetary_resource_usage": func(any *anypb.Any) *resourceusage.MonetaryResourceUsage {
			var pb resourceusage.MonetaryResourceUsage
			if err := any.UnmarshalTo(&pb); err != nil {
				return nil
			}
			return &pb
		},
		"to_file_pool_resource_usage": func(any *anypb.Any) *resourceusage.FilePoolResourceUsage {
			var pb resourceusage.FilePoolResourceUsage
			if any.UnmarshalTo(&pb) != nil {
				return nil
			}
			return &pb
		},
		"to_posix_resource_usage": func(any *anypb.Any) *resourceusage.POSIXResourceUsage {
			var pb resourceusage.POSIXResourceUsage
			if any.UnmarshalTo(&pb) != nil {
				return nil
			}
			return &pb
		},
		"to_request_metadata": func(any *anypb.Any) *remoteexecution.RequestMetadata {
			var pb remoteexecution.RequestMetadata
			if any.UnmarshalTo(&pb) != nil {
				return nil
			}
			return &pb
		},
		"shellquote": shellquote.Join,
		"timestamp_rfc3339": func(t time.Time) string {
			// Converts a timestamp to RFC3339 format.
			return t.Format(rfc3339Milli)
		},
		"timestamp_proto_delta": func(pbPrevious, pbNow *timestamppb.Timestamp) *timestampDelta {
			if err := pbNow.CheckValid(); err != nil {
				return nil
			}
			tNow := pbNow.AsTime()
			if err := pbPrevious.CheckValid(); err != nil {
				// Time may be parsed, but no split time
				// is available.
				return &timestampDelta{
					Time: tNow,
				}
			}
			tPrevious := pbPrevious.AsTime()
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
		"timestamp_proto_rfc3339": func(pb *timestamppb.Timestamp) string {
			// Converts a Protobuf timestamp to RFC 3339 format.
			if pb.CheckValid() != nil {
				return ""
			}
			return pb.AsTime().Format(rfc3339Milli)
		},
	})

	for name, template := range TemplatesData {
		templates, err = templates.New(name).Parse(template)
		if err != nil {
			log.Fatalf("Failed to parse template %#v: %s", name, err)
		}
	}

	// Prefix to add to instance names that are placed in bb_clientd
	// pathname strings.
	bbClientdInstanceNamePrefix, err := digest.NewInstanceName(configuration.BbClientdInstanceNamePrefix)
	if err != nil {
		log.Fatalf("Invalid instance name %#v: %s", configuration.BbClientdInstanceNamePrefix, err)
	}
	bbClientdInstanceNamePatcher := digest.NewInstanceNamePatcher(digest.EmptyInstanceName, bbClientdInstanceNamePrefix)

	router := mux.NewRouter()
	subrouter := router.PathPrefix(routePrefix).Subrouter()
	NewBrowserService(
		contentAddressableStorage,
		actionCache,
		initialSizeClassCache,
		int(configuration.MaximumMessageSizeBytes),
		templates,
		bbClientdInstanceNamePatcher,
		subrouter)
	RegisterAssetEndpoints(subrouter)
	go func() {
		log.Fatal(http.ListenAndServe(configuration.ListenAddress, router))
	}()

	lifecycleState.MarkReadyAndWait()
}
