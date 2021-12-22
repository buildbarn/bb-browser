package main

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"html/template"
	"image/color"
	"io"
	"log"
	"math/rand"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	remoteexecution "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
	"github.com/buildbarn/bb-remote-execution/pkg/builder"
	"github.com/buildbarn/bb-storage/pkg/blobstore"
	"github.com/buildbarn/bb-storage/pkg/digest"
	"github.com/buildbarn/bb-storage/pkg/filesystem/path"
	bb_http "github.com/buildbarn/bb-storage/pkg/http"
	cas_proto "github.com/buildbarn/bb-storage/pkg/proto/cas"
	"github.com/buildbarn/bb-storage/pkg/proto/iscc"
	"github.com/buildbarn/bb-storage/pkg/util"
	"github.com/buildkite/terminal-to-html"
	"github.com/gorilla/mux"
	"github.com/kballard/go-shellquote"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

func getDigestFromRequest(req *http.Request) (digest.Digest, error) {
	vars := mux.Vars(req)
	instanceNameStr := strings.TrimSuffix(vars["instanceName"], "/")
	instanceName, err := digest.NewInstanceName(instanceNameStr)
	if err != nil {
		return digest.BadDigest, util.StatusWrapf(err, "Invalid instance name %#v", instanceNameStr)
	}
	sizeBytes, err := strconv.ParseInt(vars["sizeBytes"], 10, 64)
	if err != nil {
		return digest.BadDigest, util.StatusWrapf(err, "Invalid blob size %#v", vars["sizeBytes"])
	}
	return instanceName.NewDigest(vars["hash"], sizeBytes)
}

// Generates a Context from an incoming HTTP request, forwarding any
// request headers as gRPC metadata.
func extractContextFromRequest(req *http.Request) context.Context {
	var pairs []string
	for key, values := range req.Header {
		for _, value := range values {
			pairs = append(pairs, key, value)
		}
	}
	return metadata.NewIncomingContext(req.Context(), metadata.Pairs(pairs...))
}

// BrowserService implements a web service that can be used to explore
// data stored in the Content Addressable Storage and Action Cache. It
// can show the details of actions and download their input and output
// files.
type BrowserService struct {
	contentAddressableStorage    blobstore.BlobAccess
	actionCache                  blobstore.BlobAccess
	initialSizeClassCache        blobstore.BlobAccess
	maximumMessageSizeBytes      int
	templates                    *template.Template
	bbClientdInstanceNamePatcher digest.InstanceNamePatcher
}

// NewBrowserService constructs a BrowserService that accesses storage
// through a set of handles.
func NewBrowserService(contentAddressableStorage, actionCache, initialSizeClassCache blobstore.BlobAccess, maximumMessageSizeBytes int, templates *template.Template, bbClientdInstanceNamePatcher digest.InstanceNamePatcher, router *mux.Router) *BrowserService {
	s := &BrowserService{
		contentAddressableStorage:    contentAddressableStorage,
		actionCache:                  actionCache,
		initialSizeClassCache:        initialSizeClassCache,
		maximumMessageSizeBytes:      maximumMessageSizeBytes,
		templates:                    templates,
		bbClientdInstanceNamePatcher: bbClientdInstanceNamePatcher,
	}
	router.HandleFunc("/", s.handleWelcome)
	router.HandleFunc("/{instanceName:(?:.*?/)?}blobs/action/{hash}-{sizeBytes}/", s.handleAction)
	router.HandleFunc("/{instanceName:(?:.*?/)?}blobs/command/{hash}-{sizeBytes}/", s.handleCommand)
	router.HandleFunc("/{instanceName:(?:.*?/)?}blobs/directory/{hash}-{sizeBytes}/", s.handleDirectory)
	router.HandleFunc("/{instanceName:(?:.*?/)?}blobs/file/{hash}-{sizeBytes}/{name}", s.handleFile)
	router.HandleFunc("/{instanceName:(?:.*?/)?}blobs/previous_execution_stats/{hash}-{sizeBytes}/", s.handlePreviousExecutionStats)
	router.HandleFunc("/{instanceName:(?:.*?/)?}blobs/tree/{hash}-{sizeBytes}/{subdirectory:(?:.*/)?}", s.handleTree)
	router.HandleFunc("/{instanceName:(?:.*?/)?}blobs/historical_execute_response/{hash}-{sizeBytes}/", s.handleHistoricalExecuteResponse)
	return s
}

var (
	invalidReplacementComponent = path.MustNewComponent("???")
	commandDirectoryComponent   = path.MustNewComponent("command")
	blobsDirectoryComponent     = path.MustNewComponent("blobs")
	directoryDirectoryComponent = path.MustNewComponent("directory")
	treeDirectoryComponent      = path.MustNewComponent("tree")
)

func (s *BrowserService) renderError(w http.ResponseWriter, err error) {
	st := status.Convert(err)
	w.WriteHeader(bb_http.StatusCodeFromGRPCCode(st.Code()))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if err := s.templates.ExecuteTemplate(w, "error.html", st); err != nil {
		log.Print(err)
	}
}

// getBBClientdBlobPath returns a relative path of the shape
// "${instanceName}/blobs/${blobType}/${hash}-${sizeBytes}". This
// corresponds to the pathname scheme that can be used to access
// individual objects in bb_clientd.
func (s *BrowserService) getBBClientdBlobPath(blobDigest digest.Digest, blobType path.Component) *path.Trace {
	// Convert the instance name to a pathname.
	externalDigest := s.bbClientdInstanceNamePatcher.PatchDigest(blobDigest)
	var fullPath *path.Trace
	for _, component := range externalDigest.GetInstanceName().GetComponents() {
		pathComponent, ok := path.NewComponent(component)
		if !ok {
			// Instance name cannot be represented as a UNIX
			// pathname. Replace the invalid component with
			// "???", so that we can still generate pathname
			// strings for the user to view and edit.
			pathComponent = invalidReplacementComponent
		}
		fullPath = fullPath.Append(pathComponent)
	}
	return fullPath.
		Append(blobsDirectoryComponent).
		Append(blobType).
		Append(path.MustNewComponent(fmt.Sprintf("%s-%d", externalDigest.GetHashString(), externalDigest.GetSizeBytes())))
}

// formatBBClientdPath converts the value generated by
// getBBClientdBlobPath to a full path into bb_clientd that can be
// pasted into a shell. It assumes that bb_clientd's FUSE file system is
// mounted at ~/bb_clientd, the default.
func formatBBClientdPath(p *path.Trace) string {
	return "~/bb_clientd/cas/" + shellquote.Join(p.String())
}

func (s *BrowserService) handleWelcome(w http.ResponseWriter, req *http.Request) {
	if err := s.templates.ExecuteTemplate(w, "page_welcome.html", nil); err != nil {
		log.Print(err)
	}
}

type commandInfo struct {
	Digest        digest.Digest
	Command       *remoteexecution.Command
	BBClientdPath string
}

type directoryInfo struct {
	Digest        digest.Digest
	Directory     *remoteexecution.Directory
	BBClientdPath string
}

type logInfo struct {
	Name     string
	Digest   digest.Digest
	TooLarge bool
	NotFound bool
	HTML     template.HTML
}

func (s *BrowserService) handleAction(w http.ResponseWriter, req *http.Request) {
	digest, err := getDigestFromRequest(req)
	if err != nil {
		s.renderError(w, err)
		return
	}

	ctx := extractContextFromRequest(req)
	var actionResult *remoteexecution.ActionResult
	if m, err := s.actionCache.Get(ctx, digest).ToProto(
		&remoteexecution.ActionResult{},
		s.maximumMessageSizeBytes); err == nil {
		actionResult = m.(*remoteexecution.ActionResult)
	} else if status.Code(err) != codes.NotFound {
		s.renderError(w, err)
		return
	}

	s.handleActionCommon(w, req, digest, &remoteexecution.ExecuteResponse{
		Result: actionResult,
	}, false)
}

func (s *BrowserService) handleHistoricalExecuteResponse(w http.ResponseWriter, req *http.Request) {
	digest, err := getDigestFromRequest(req)
	if err != nil {
		s.renderError(w, err)
		return
	}
	ctx := extractContextFromRequest(req)
	m, err := s.contentAddressableStorage.Get(ctx, digest).ToProto(&cas_proto.HistoricalExecuteResponse{}, s.maximumMessageSizeBytes)
	if err != nil {
		s.renderError(w, err)
		return
	}
	historicalExecuteResponse := m.(*cas_proto.HistoricalExecuteResponse)
	actionDigest, err := digest.GetInstanceName().NewDigestFromProto(historicalExecuteResponse.ActionDigest)
	if err != nil {
		s.renderError(w, err)
		return
	}
	s.handleActionCommon(w, req, actionDigest, historicalExecuteResponse.ExecuteResponse, true)
}

func (s *BrowserService) getLogInfoFromActionResult(ctx context.Context, name string, instanceName digest.InstanceName, logDigest *remoteexecution.Digest, rawLogBody []byte) (*logInfo, error) {
	var blobDigest digest.Digest
	if logDigest != nil {
		var err error
		blobDigest, err = instanceName.NewDigestFromProto(logDigest)
		if err != nil {
			return nil, err
		}
	}

	if len(rawLogBody) > 0 {
		// Log body is small enough to be provided inline.
		return &logInfo{
			Name:   name,
			Digest: blobDigest,
			HTML:   template.HTML(terminal.Render(rawLogBody)),
		}, nil
	} else if logDigest != nil {
		// Load the log from the Content Addressable Storage.
		return s.getLogInfoForDigest(ctx, name, blobDigest)
	}
	return nil, nil
}

func (s *BrowserService) getLogInfoForDigest(ctx context.Context, name string, digest digest.Digest) (*logInfo, error) {
	maximumLogSizeBytes := 100000
	if size := digest.GetSizeBytes(); size == 0 {
		// No log file present.
		return nil, nil
	} else if size > int64(maximumLogSizeBytes) {
		// Log file too large to show inline.
		return &logInfo{
			Name:     name,
			Digest:   digest,
			TooLarge: true,
		}, nil
	}

	data, err := s.contentAddressableStorage.Get(ctx, digest).ToByteSlice(maximumLogSizeBytes)
	if err == nil {
		// Log found. Convert ANSI escape sequences to HTML.
		return &logInfo{
			Name:   name,
			Digest: digest,
			HTML:   template.HTML(terminal.Render(data)),
		}, nil
	} else if status.Code(err) == codes.NotFound {
		// Not found.
		return &logInfo{
			Name:     name,
			Digest:   digest,
			NotFound: true,
		}, nil
	}
	return nil, err
}

func (s *BrowserService) handleActionCommon(w http.ResponseWriter, req *http.Request, actionDigest digest.Digest, executeResponse *remoteexecution.ExecuteResponse, isHistoricalExecuteResponse bool) {
	instanceName := actionDigest.GetInstanceName()
	actionInfo := struct {
		IsHistoricalExecuteResponse bool
		ActionDigest                digest.Digest
		Action                      *remoteexecution.Action

		Command *commandInfo

		ExecuteResponse *remoteexecution.ExecuteResponse
		StdoutInfo      *logInfo
		StderrInfo      *logInfo

		InputRoot *directoryInfo

		OutputDirectories  []*remoteexecution.OutputDirectory
		OutputSymlinks     []*remoteexecution.OutputSymlink
		OutputFiles        []*remoteexecution.OutputFile
		MissingDirectories []string
		MissingFiles       []string

		PreviousExecutionStats *previousExecutionStatsInfo
	}{
		IsHistoricalExecuteResponse: isHistoricalExecuteResponse,
		ActionDigest:                actionDigest,
		ExecuteResponse:             executeResponse,
	}

	ctx := extractContextFromRequest(req)
	actionResult := executeResponse.GetResult()
	if actionResult != nil {
		actionInfo.OutputDirectories = actionResult.OutputDirectories
		actionInfo.OutputSymlinks = actionResult.OutputFileSymlinks
		actionInfo.OutputFiles = actionResult.OutputFiles

		var err error
		actionInfo.StdoutInfo, err = s.getLogInfoFromActionResult(ctx, "Standard output", instanceName, actionResult.StdoutDigest, actionResult.StdoutRaw)
		if err != nil {
			s.renderError(w, err)
			return
		}
		actionInfo.StderrInfo, err = s.getLogInfoFromActionResult(ctx, "Standard error", instanceName, actionResult.StderrDigest, actionResult.StderrRaw)
		if err != nil {
			s.renderError(w, err)
			return
		}
	}

	actionMessage, err := s.contentAddressableStorage.Get(ctx, actionDigest).ToProto(&remoteexecution.Action{}, s.maximumMessageSizeBytes)
	if err == nil {
		action := actionMessage.(*remoteexecution.Action)
		actionInfo.Action = action

		commandDigest, err := instanceName.NewDigestFromProto(action.CommandDigest)
		if err != nil {
			s.renderError(w, err)
			return
		}
		commandMessage, err := s.contentAddressableStorage.Get(ctx, commandDigest).ToProto(&remoteexecution.Command{}, s.maximumMessageSizeBytes)
		if err == nil {
			command := commandMessage.(*remoteexecution.Command)
			actionInfo.Command = &commandInfo{
				Digest:        commandDigest,
				Command:       command,
				BBClientdPath: formatBBClientdPath(s.getBBClientdBlobPath(commandDigest, commandDirectoryComponent)),
			}

			foundDirectories := map[string]bool{}
			for _, outputDirectory := range actionInfo.OutputDirectories {
				foundDirectories[outputDirectory.Path] = true
			}
			for _, outputDirectory := range command.OutputDirectories {
				if _, ok := foundDirectories[outputDirectory]; !ok {
					actionInfo.MissingDirectories = append(actionInfo.MissingDirectories, outputDirectory)
				}
			}
			foundFiles := map[string]bool{}
			for _, outputSymlinks := range actionInfo.OutputSymlinks {
				foundFiles[outputSymlinks.Path] = true
			}
			for _, outputFiles := range actionInfo.OutputFiles {
				foundFiles[outputFiles.Path] = true
			}
			for _, outputFile := range command.OutputFiles {
				if _, ok := foundFiles[outputFile]; !ok {
					actionInfo.MissingFiles = append(actionInfo.MissingFiles, outputFile)
				}
			}
		} else if status.Code(err) != codes.NotFound {
			s.renderError(w, err)
			return
		}

		inputRootDigest, err := instanceName.NewDigestFromProto(action.InputRootDigest)
		if err != nil {
			s.renderError(w, err)
			return
		}
		directoryMessage, err := s.contentAddressableStorage.Get(ctx, inputRootDigest).ToProto(&remoteexecution.Directory{}, s.maximumMessageSizeBytes)
		if err == nil {
			actionInfo.InputRoot = &directoryInfo{
				Digest:        inputRootDigest,
				Directory:     directoryMessage.(*remoteexecution.Directory),
				BBClientdPath: formatBBClientdPath(s.getBBClientdBlobPath(inputRootDigest, directoryDirectoryComponent)),
			}
		} else if status.Code(err) != codes.NotFound {
			s.renderError(w, err)
			return
		}

		reducedActionDigest, err := blobstore.ISCCGetReducedActionDigest(actionDigest.GetDigestFunction(), action)
		if err != nil {
			s.renderError(w, err)
			return
		}
		previousExecutionStatsInfo, err := s.getPreviousExecutionStatsInfo(ctx, reducedActionDigest)
		if err == nil {
			actionInfo.PreviousExecutionStats = previousExecutionStatsInfo
		} else if status.Code(err) != codes.NotFound {
			s.renderError(w, err)
			return
		}
	} else if status.Code(err) != codes.NotFound {
		s.renderError(w, err)
		return
	}

	if actionMessage == nil && actionResult == nil {
		s.renderError(w, status.Error(codes.NotFound, "Could not find an action or action result"))
		return
	}

	if err := s.templates.ExecuteTemplate(w, "page_action.html", actionInfo); err != nil {
		log.Print(err)
	}
}

func (s *BrowserService) handleCommand(w http.ResponseWriter, req *http.Request) {
	digest, err := getDigestFromRequest(req)
	if err != nil {
		s.renderError(w, err)
		return
	}

	ctx := extractContextFromRequest(req)
	commandMessage, err := s.contentAddressableStorage.Get(ctx, digest).ToProto(&remoteexecution.Command{}, s.maximumMessageSizeBytes)
	if err != nil {
		s.renderError(w, err)
		return
	}
	command := commandMessage.(*remoteexecution.Command)

	if req.URL.Query().Get("format") == "sh" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		bw := bufio.NewWriter(w)
		if err := builder.ConvertCommandToShellScript(command, bw); err != nil {
			log.Print(err)
			panic(http.ErrAbortHandler)
		}
		if err := bw.Flush(); err != nil {
			log.Print(err)
			panic(http.ErrAbortHandler)
		}
	} else {
		if err := s.templates.ExecuteTemplate(w, "page_command.html", commandInfo{
			Digest:        digest,
			Command:       command,
			BBClientdPath: formatBBClientdPath(s.getBBClientdBlobPath(digest, commandDirectoryComponent)),
		}); err != nil {
			log.Print(err)
		}
	}
}

func (s *BrowserService) generateTarballDirectory(ctx context.Context, w *tar.Writer, instanceName digest.InstanceName, directory *remoteexecution.Directory, directoryPath *path.Trace, getDirectory func(context.Context, digest.Digest) (*remoteexecution.Directory, error), filesSeen map[string]string) error {
	// Emit child directories.
	for _, directoryNode := range directory.Directories {
		childName, ok := path.NewComponent(directoryNode.Name)
		if !ok {
			return status.Errorf(codes.InvalidArgument, "Directory %#v in directory %#v has an invalid name", directoryNode.Name, directoryPath.String())
		}
		childPath := directoryPath.Append(childName)

		if err := w.WriteHeader(&tar.Header{
			Typeflag: tar.TypeDir,
			Name:     childPath.String(),
			Mode:     0o777,
		}); err != nil {
			return err
		}
		childDigest, err := instanceName.NewDigestFromProto(directoryNode.Digest)
		if err != nil {
			return err
		}
		childDirectory, err := getDirectory(ctx, childDigest)
		if err != nil {
			return err
		}
		if err := s.generateTarballDirectory(ctx, w, instanceName, childDirectory, childPath, getDirectory, filesSeen); err != nil {
			return err
		}
	}

	// Emit symlinks.
	for _, symlinkNode := range directory.Symlinks {
		childName, ok := path.NewComponent(symlinkNode.Name)
		if !ok {
			return status.Errorf(codes.InvalidArgument, "Symbolic link %#v in directory %#v has an invalid name", symlinkNode.Name, directoryPath.String())
		}
		childPath := directoryPath.Append(childName)

		if err := w.WriteHeader(&tar.Header{
			Typeflag: tar.TypeSymlink,
			Name:     childPath.String(),
			Linkname: symlinkNode.Target,
			Mode:     0o777,
		}); err != nil {
			return err
		}
	}

	// Emit regular files.
	for _, fileNode := range directory.Files {
		childName, ok := path.NewComponent(fileNode.Name)
		if !ok {
			return status.Errorf(codes.InvalidArgument, "File %#v in directory %#v has an invalid name", fileNode.Name, directoryPath.String())
		}
		childPath := directoryPath.Append(childName)
		childPathString := childPath.String()

		childDigest, err := instanceName.NewDigestFromProto(fileNode.Digest)
		if err != nil {
			return err
		}

		childKey := childDigest.GetKey(digest.KeyWithoutInstance)
		if fileNode.IsExecutable {
			childKey += "+x"
		} else {
			childKey += "-x"
		}

		if linkPath, ok := filesSeen[childKey]; ok {
			// This file was already returned previously.
			// Emit a hardlink pointing to the first
			// occurrence.
			//
			// Not only does this reduce the size of the
			// tarball, it also makes the directory more
			// representative of what it looks like when
			// executed through bb_worker.
			if err := w.WriteHeader(&tar.Header{
				Typeflag: tar.TypeLink,
				Name:     childPathString,
				Linkname: linkPath,
			}); err != nil {
				return err
			}
		} else {
			// This is the first time we're returning this
			// file. Actually add it to the archive.
			mode := int64(0o666)
			if fileNode.IsExecutable {
				mode = 0o777
			}
			if err := w.WriteHeader(&tar.Header{
				Typeflag: tar.TypeReg,
				Name:     childPathString,
				Size:     fileNode.Digest.SizeBytes,
				Mode:     mode,
			}); err != nil {
				return err
			}

			if err := s.contentAddressableStorage.Get(ctx, childDigest).IntoWriter(w); err != nil {
				return err
			}

			filesSeen[childKey] = childPathString
		}
	}
	return nil
}

func (s *BrowserService) generateTarball(ctx context.Context, w http.ResponseWriter, digest digest.Digest, directory *remoteexecution.Directory, getDirectory func(context.Context, digest.Digest) (*remoteexecution.Directory, error)) {
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.tar.gz\"", digest.GetHashString()))
	w.Header().Set("Content-Type", "application/gzip")
	gzipWriter := gzip.NewWriter(w)
	tarWriter := tar.NewWriter(gzipWriter)
	filesSeen := map[string]string{}
	if err := s.generateTarballDirectory(ctx, tarWriter, digest.GetInstanceName(), directory, nil, getDirectory, filesSeen); err != nil {
		// TODO(edsch): Any way to propagate this to the client?
		log.Print(err)
		panic(http.ErrAbortHandler)
	}
	if err := tarWriter.Close(); err != nil {
		log.Print(err)
		panic(http.ErrAbortHandler)
	}
	if err := gzipWriter.Close(); err != nil {
		log.Print(err)
		panic(http.ErrAbortHandler)
	}
}

func (s *BrowserService) handleDirectory(w http.ResponseWriter, req *http.Request) {
	directoryDigest, err := getDigestFromRequest(req)
	if err != nil {
		s.renderError(w, err)
		return
	}

	ctx := extractContextFromRequest(req)
	directoryMessage, err := s.contentAddressableStorage.Get(ctx, directoryDigest).ToProto(&remoteexecution.Directory{}, s.maximumMessageSizeBytes)
	if err != nil {
		s.renderError(w, err)
		return
	}
	directory := directoryMessage.(*remoteexecution.Directory)

	if req.URL.Query().Get("format") == "tar" {
		s.generateTarball(ctx, w, directoryDigest, directory, func(ctx context.Context, digest digest.Digest) (*remoteexecution.Directory, error) {
			directoryMessage, err := s.contentAddressableStorage.Get(ctx, digest).ToProto(&remoteexecution.Directory{}, s.maximumMessageSizeBytes)
			if err != nil {
				return nil, err
			}
			return directoryMessage.(*remoteexecution.Directory), nil
		})
	} else {
		if err := s.templates.ExecuteTemplate(w, "page_directory.html", directoryInfo{
			Digest:        directoryDigest,
			Directory:     directory,
			BBClientdPath: formatBBClientdPath(s.getBBClientdBlobPath(directoryDigest, directoryDirectoryComponent)),
		}); err != nil {
			log.Print(err)
		}
	}
}

func (s *BrowserService) handleFile(w http.ResponseWriter, req *http.Request) {
	digest, err := getDigestFromRequest(req)
	if err != nil {
		s.renderError(w, err)
		return
	}

	ctx := extractContextFromRequest(req)
	r := s.contentAddressableStorage.Get(ctx, digest).ToReader()
	defer r.Close()

	// Attempt to read the first chunk of data to see whether we can
	// trigger an error. Only when no error occurs, we start setting
	// response headers.
	var first [4096]byte
	n, err := r.Read(first[:])
	if err != nil && err != io.EOF {
		s.renderError(w, err)
		return
	}

	w.Header().Set("Content-Length", strconv.FormatInt(digest.GetSizeBytes(), 10))
	if utf8.ValidString(string(first[:])) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	w.Write(first[:n])
	io.Copy(w, r)
}

// previousExecutionStatsInfo contains the information that we display
// for PreviousExecutionStats messages stored in the Initial Size Class
// Cache (ISCC).
type previousExecutionStatsInfo struct {
	ReducedActionDigest digest.Digest
	Stats               *iscc.PreviousExecutionStats
	ScatterPlot         template.HTML
}

func (s *BrowserService) getPreviousExecutionStatsInfo(ctx context.Context, reducedActionDigest digest.Digest) (*previousExecutionStatsInfo, error) {
	previousExecutionStatsMessage, err := s.initialSizeClassCache.Get(ctx, reducedActionDigest).
		ToProto(&iscc.PreviousExecutionStats{}, s.maximumMessageSizeBytes)
	if err != nil {
		return nil, err
	}
	previousExecutionStats := previousExecutionStatsMessage.(*iscc.PreviousExecutionStats)

	// Obtain list of size classes in increasing order.
	sizeClasses := make(sizeClassList, 0, len(previousExecutionStats.SizeClasses))
	for sizeClass := range previousExecutionStats.SizeClasses {
		sizeClasses = append(sizeClasses, sizeClass)
	}
	sort.Sort(sizeClasses)

	// Convert outcomes into samples of a scatter plot. Add some
	// jitter on the X axis to make it easier to tell individual
	// outcomes apart.
	rng := rand.New(rand.NewSource(0x4630324434464134))
	var successes, timeouts, failures plotter.XYs
	for idx, sizeClass := range sizeClasses {
		perSizeClassStats := previousExecutionStats.SizeClasses[sizeClass]
		for _, previousExecution := range perSizeClassStats.PreviousExecutions {
			switch outcome := previousExecution.Outcome.(type) {
			case *iscc.PreviousExecution_Succeeded:
				successes = append(successes, plotter.XY{
					X: float64(idx) + (rng.Float64()-0.5)/3,
					Y: outcome.Succeeded.AsDuration().Seconds(),
				})
			case *iscc.PreviousExecution_TimedOut:
				timeouts = append(timeouts, plotter.XY{
					X: float64(idx) + (rng.Float64()-0.5)/3,
					Y: outcome.TimedOut.AsDuration().Seconds(),
				})
			case *iscc.PreviousExecution_Failed:
				failures = append(failures, plotter.XY{
					X: float64(idx) + (rng.Float64()-0.5)/3,
				})
			}
		}
	}

	// Place all three groups of samples in the scatter plot.
	p := plot.New()
	p.X.Min = -0.5
	p.X.Max = float64(len(sizeClasses)) - 0.5
	p.Y.Label.Text = "Execution time (s)"
	p.Y.Min = 0

	scatterSuccess, err := plotter.NewScatter(successes)
	if err != nil {
		return nil, err
	}
	scatterSuccess.Color = color.RGBA{R: 40, G: 167, B: 69, A: 255}
	scatterSuccess.Radius = vg.Points(5)
	scatterSuccess.Shape = draw.PlusGlyph{}
	p.Add(scatterSuccess)

	scatterTimeout, err := plotter.NewScatter(timeouts)
	if err != nil {
		return nil, err
	}
	scatterTimeout.Color = color.RGBA{R: 255, G: 193, B: 7, A: 255}
	scatterTimeout.Radius = vg.Points(2.5)
	scatterTimeout.Shape = draw.CircleGlyph{}
	p.Add(scatterTimeout)

	scatterFailed, err := plotter.NewScatter(failures)
	if err != nil {
		return nil, err
	}
	scatterFailed.Color = color.RGBA{R: 220, G: 53, B: 69, A: 255}
	scatterFailed.Radius = vg.Points(5)
	scatterFailed.Shape = draw.CrossGlyph{}
	p.Add(scatterFailed)

	sizeClassLabels := make([]string, 0, len(sizeClasses))
	for _, sizeClass := range sizeClasses {
		sizeClassLabels = append(sizeClassLabels, fmt.Sprintf("Size class %d", sizeClass))
	}
	p.NominalX(sizeClassLabels...)

	// Convert the resulting scatter plot to SVG.
	var graph strings.Builder
	writerTo, err := p.WriterTo(vg.Length(len(sizeClasses)+1)*3*vg.Centimeter, 10*vg.Centimeter, "svg")
	if err != nil {
		return nil, err
	}
	if _, err := writerTo.WriteTo(&graph); err != nil {
		return nil, err
	}

	return &previousExecutionStatsInfo{
		ReducedActionDigest: reducedActionDigest,
		Stats:               previousExecutionStats,
		ScatterPlot:         template.HTML(graph.String()),
	}, nil
}

type sizeClassList []uint32

func (l sizeClassList) Len() int {
	return len(l)
}

func (l sizeClassList) Less(i, j int) bool {
	return l[i] < l[j]
}

func (l sizeClassList) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

func (s *BrowserService) handlePreviousExecutionStats(w http.ResponseWriter, req *http.Request) {
	digest, err := getDigestFromRequest(req)
	if err != nil {
		s.renderError(w, err)
		return
	}

	ctx := extractContextFromRequest(req)
	statsInfo, err := s.getPreviousExecutionStatsInfo(ctx, digest)
	if err != nil {
		s.renderError(w, err)
		return
	}

	if err := s.templates.ExecuteTemplate(w, "page_previous_execution_stats.html", statsInfo); err != nil {
		log.Print(err)
	}
}

func (s *BrowserService) handleTree(w http.ResponseWriter, req *http.Request) {
	treeDigest, err := getDigestFromRequest(req)
	if err != nil {
		s.renderError(w, err)
		return
	}

	ctx := extractContextFromRequest(req)
	treeMessage, err := s.contentAddressableStorage.Get(ctx, treeDigest).ToProto(&remoteexecution.Tree{}, s.maximumMessageSizeBytes)
	if err != nil {
		s.renderError(w, err)
		return
	}
	tree := treeMessage.(*remoteexecution.Tree)
	instanceName := treeDigest.GetInstanceName()
	treeInfo := struct {
		Directory          *remoteexecution.Directory
		HasParentDirectory bool
		BBClientdPath      string
		RootDirectory      string
	}{
		Directory: tree.Root,
	}

	// Construct map of all child directories.
	digestFunction := treeDigest.GetDigestFunction()
	children := map[string]*remoteexecution.Directory{}
	for _, child := range tree.Children {
		data, err := proto.Marshal(child)
		if err != nil {
			s.renderError(w, err)
			return
		}
		digestGenerator := digestFunction.NewGenerator()
		if _, err := digestGenerator.Write(data); err != nil {
			s.renderError(w, err)
			return
		}
		children[digestGenerator.Sum().GetKey(digest.KeyWithoutInstance)] = child
	}

	// In case additional directory components are provided, we need
	// to traverse the directories stored within. While there,
	// compute the inverse (i.e., "../../.."). This gets passed to
	// the template, so that we can still emit relative links to
	// other pages.
	bbClientdPath := s.getBBClientdBlobPath(treeDigest, treeDirectoryComponent)
	directoryDigest := treeDigest
	rootDirectory, scopeWalker := path.EmptyBuilder.Join(path.VoidScopeWalker)
	rootDirectoryWalker, _ := scopeWalker.OnScope(false)
	for _, component := range strings.FieldsFunc(
		mux.Vars(req)["subdirectory"],
		func(r rune) bool { return r == '/' }) {
		pathComponent, ok := path.NewComponent(component)
		if !ok {
			s.renderError(w, status.Errorf(codes.InvalidArgument, "Path contains invalid component %#v"))
			return
		}
		bbClientdPath = bbClientdPath.Append(pathComponent)
		rootDirectoryWalker, _ = rootDirectoryWalker.OnUp()

		// Find child with matching name.
		childNode := func() *remoteexecution.DirectoryNode {
			for _, directoryNode := range treeInfo.Directory.Directories {
				if component == directoryNode.Name {
					return directoryNode
				}
			}
			return nil
		}()
		if childNode == nil {
			s.renderError(w, status.Error(codes.NotFound, "Subdirectory in tree not found"))
			return
		}

		// Find corresponding child directory message.
		directoryDigest, err = instanceName.NewDigestFromProto(childNode.Digest)
		if err != nil {
			s.renderError(w, err)
			return
		}
		childDirectory, ok := children[directoryDigest.GetKey(digest.KeyWithoutInstance)]
		if !ok {
			s.renderError(w, status.Error(codes.InvalidArgument, "Failed to find child node in tree"))
			return
		}
		treeInfo.HasParentDirectory = true
		treeInfo.Directory = childDirectory
	}
	treeInfo.BBClientdPath = formatBBClientdPath(bbClientdPath)
	treeInfo.RootDirectory = rootDirectory.String()

	if req.URL.Query().Get("format") == "tar" {
		s.generateTarball(
			ctx, w, directoryDigest, treeInfo.Directory,
			func(ctx context.Context, directoryDigest digest.Digest) (*remoteexecution.Directory, error) {
				childDirectory, ok := children[directoryDigest.GetKey(digest.KeyWithoutInstance)]
				if !ok {
					return nil, errors.New("Failed to find child node in tree")
				}
				return childDirectory, nil
			})
	} else {
		if err := s.templates.ExecuteTemplate(w, "page_tree.html", &treeInfo); err != nil {
			log.Print(err)
		}
	}
}
