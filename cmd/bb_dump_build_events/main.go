package main

import (
	"io"
	"log"
	"os"

	buildeventstream "github.com/bazelbuild/bazel/src/main/java/com/google/devtools/build/lib/buildeventstream/proto"
	"github.com/buildbarn/bb-browser/pkg/buildevents"
	"github.com/davecgh/go-spew/spew"
	"github.com/matttproud/golang_protobuf_extensions/pbutil"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatal("usage: bb_dump_build_events filename")
	}
	f, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatal("Failed to open build events file: ", err)
	}

	// Parse all build events into a typed tree.
	parser := buildevents.NewStreamParser()
	for {
		var event buildeventstream.BuildEvent
		if _, err := pbutil.ReadDelimited(f, &event); err == io.EOF {
			break
		} else if err != nil {
			log.Fatal("Failed to parse message: ", err)
		}

		if err := parser.AddBuildEvent(&event); err != nil {
			log.Fatal("Failed to add build event: ", err)
		}
	}

	// Dump the full tree to stdout.
	started, finished := parser.Finalize()
	if !finished {
		log.Print("Warning: build not finished")
	}
	spew.Dump(started)
}
