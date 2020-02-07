package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"unicode/utf8"

	buildeventstream "github.com/bazelbuild/bazel/src/main/java/com/google/devtools/build/lib/buildeventstream/proto"
	remoteexecution "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
	"github.com/buildbarn/bb-browser/pkg/buildevents"
	"github.com/buildbarn/bb-storage/pkg/blobstore"
	"github.com/buildbarn/bb-storage/pkg/cas"
	"github.com/buildbarn/bb-storage/pkg/digest"
	"github.com/buildkite/terminal"
	"github.com/golang/protobuf/proto"
	"github.com/gorilla/mux"
	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"github.com/matttproud/golang_protobuf_extensions/pbutil"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func getDigestFromRequest(req *http.Request) (digest.Digest, error) {
	vars := mux.Vars(req)
	sizeBytes, err := strconv.ParseInt(vars["sizeBytes"], 10, 64)
	if err != nil {
		return digest.BadDigest, err
	}
	return digest.NewDigest(vars["instance"], vars["hash"], sizeBytes)
}

// Generates a Context from an incoming HTTP request, forwarding any
// 'Authorization' header as gRPC 'authorization' metadata.
func extractContextFromRequest(req *http.Request) context.Context {
	ctx := req.Context()
	md := metautils.ExtractIncoming(ctx)
	if authorization := req.Header.Get("Authorization"); authorization != "" {
		md.Set("authorization", authorization)
	}
	return md.ToIncoming(ctx)
}

// BrowserService implements a web service that can be used to explore
// data stored in the Content Addressable Storage and Action Cache. It
// can show the details of actions and download their input and output
// files.
type BrowserService struct {
	contentAddressableStorage           cas.ContentAddressableStorage
	contentAddressableStorageBlobAccess blobstore.BlobAccess
	actionCache                         blobstore.BlobAccess
	maximumMessageSizeBytes             int
	templates                           *template.Template
}

// NewBrowserService constructs a BrowserService that accesses storage
// through a set of handles.
func NewBrowserService(contentAddressableStorage cas.ContentAddressableStorage, contentAddressableStorageBlobAccess blobstore.BlobAccess, actionCache blobstore.BlobAccess, maximumMessageSizeBytes int, templates *template.Template, router *mux.Router) *BrowserService {
	s := &BrowserService{
		contentAddressableStorage:           contentAddressableStorage,
		contentAddressableStorageBlobAccess: contentAddressableStorageBlobAccess,
		actionCache:                         actionCache,
		maximumMessageSizeBytes:             maximumMessageSizeBytes,
		templates:                           templates,
	}
	router.HandleFunc("/", s.handleWelcome)
	router.HandleFunc("/action/{instance}/{hash}/{sizeBytes}/", s.handleAction)
	router.HandleFunc("/build_events/{instance}/{invocationID}", s.handleBuildEvents)
	router.HandleFunc("/command/{instance}/{hash}/{sizeBytes}/", s.handleCommand)
	router.HandleFunc("/directory/{instance}/{hash}/{sizeBytes}/", s.handleDirectory)
	router.HandleFunc("/file/{instance}/{hash}/{sizeBytes}/{name}", s.handleFile)
	router.HandleFunc("/tree/{instance}/{hash}/{sizeBytes}/{subdirectory:(?:.*/)?}", s.handleTree)
	router.HandleFunc("/uncached_action_result/{instance}/{hash}/{sizeBytes}/", s.handleUncachedActionResult)
	return s
}

func (s *BrowserService) handleWelcome(w http.ResponseWriter, req *http.Request) {
	if err := s.templates.ExecuteTemplate(w, "page_welcome.html", nil); err != nil {
		log.Print(err)
	}
}

type directoryInfo struct {
	Digest    digest.Digest
	Directory *remoteexecution.Directory
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
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := extractContextFromRequest(req)
	actionResult, err := s.actionCache.Get(ctx, digest).ToActionResult(s.maximumMessageSizeBytes)
	if err != nil && status.Code(err) != codes.NotFound {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.handleActionCommon(w, req, digest, &remoteexecution.ExecuteResponse{
		Result: actionResult,
	})
}

func (s *BrowserService) readBuildEventStream(ctx context.Context, parser *buildevents.StreamParser, digest digest.Digest) error {
	r := s.contentAddressableStorageBlobAccess.Get(ctx, digest).ToReader()
	defer r.Close()

	for {
		var event buildeventstream.BuildEvent
		if _, err := pbutil.ReadDelimited(r, &event); err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}
		if err := parser.AddBuildEvent(&event); err != nil {
			return err
		}
	}
}

func (s *BrowserService) handleBuildEvents(w http.ResponseWriter, req *http.Request) {
	// Convert the invocation ID in the URL to a digest by hashing
	// it, so that a fictive Action Cache entry can be accessed.
	vars := mux.Vars(req)
	hash := sha256.Sum256([]byte(vars["invocationID"]))
	digest, err := digest.NewDigest(vars["instance"], hex.EncodeToString(hash[:]), 0)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Look up the invocation ID in the Action Cache. The output
	// files of this entry correspond to one or more blobs in the
	// CAS that, when concatenated, represents a Build Event Stream.
	ctx := extractContextFromRequest(req)
	actionResult, err := s.actionCache.Get(ctx, digest).ToActionResult(s.maximumMessageSizeBytes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	parser := buildevents.NewStreamParser()
	for _, file := range actionResult.OutputFiles {
		fileDigest, err := digest.NewDerivedDigest(file.Digest)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := s.readBuildEventStream(ctx, parser, fileDigest); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	started, eventsExpected, err := parser.Finalize()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Convert stdout and stderr stored in ActionCompletedNodes to a
	// format usable by the templates.
	logsForActionsCompleted := map[*buildevents.ActionCompletedNode][]*logInfo{}
	if started != nil {
		for progress := started.Progress; progress != nil; progress = progress.Progress {
			for _, actionCompleted := range progress.ActionsCompleted {
				if stdout, err := s.getLogInfoFromActionCompleted(ctx, "Standard output", actionCompleted.Payload.Stdout); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				} else if stdout != nil {
					logsForActionsCompleted[actionCompleted] = append(logsForActionsCompleted[actionCompleted], stdout)
				}

				if stderr, err := s.getLogInfoFromActionCompleted(ctx, "Standard error", actionCompleted.Payload.Stderr); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				} else if stderr != nil {
					logsForActionsCompleted[actionCompleted] = append(logsForActionsCompleted[actionCompleted], stderr)
				}
			}
		}
	}

	if err := s.templates.ExecuteTemplate(w, "page_build_events.html", struct {
		Started                 *buildevents.StartedNode
		EventsExpected          int
		LogsForActionsCompleted map[*buildevents.ActionCompletedNode][]*logInfo
	}{
		Started:                 started,
		EventsExpected:          eventsExpected,
		LogsForActionsCompleted: logsForActionsCompleted,
	}); err != nil {
		log.Print(err)
	}
}

func (s *BrowserService) handleUncachedActionResult(w http.ResponseWriter, req *http.Request) {
	digest, err := getDigestFromRequest(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	ctx := extractContextFromRequest(req)
	uncachedActionResult, err := s.contentAddressableStorage.GetUncachedActionResult(ctx, digest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	actionDigest, err := digest.NewDerivedDigest(uncachedActionResult.ActionDigest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.handleActionCommon(w, req, actionDigest, uncachedActionResult.ExecuteResponse)
}

func (s *BrowserService) getLogInfoFromActionResult(ctx context.Context, name string, instance string, logDigest *remoteexecution.Digest, rawLogBody []byte) (*logInfo, error) {
	var blobDigest digest.Digest
	if logDigest != nil {
		var err error
		blobDigest, err = digest.NewDigestFromPartialDigest(instance, logDigest)
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

func (s *BrowserService) getLogInfoFromActionCompleted(ctx context.Context, name string, file *buildeventstream.File) (*logInfo, error) {
	if file == nil {
		return nil, nil
	}
	switch data := file.File.(type) {
	case *buildeventstream.File_Uri:
		// A pathname or URL is provided. Load the log from the
		// Content Addressable Storage if it's in the form of a
		// bytestream:// URL.
		u, err := url.Parse(data.Uri)
		if err != nil || u.Scheme != "bytestream" {
			return &logInfo{
				Name:     name,
				NotFound: true,
			}, nil
		}
		digest, err := digest.NewDigestFromBytestreamPath(u.Path)
		if err != nil {
			return &logInfo{
				Name:     name,
				NotFound: true,
			}, nil
		}
		return s.getLogInfoForDigest(ctx, name, digest)
	case *buildeventstream.File_Contents:
		// Log body is small enough to be provided inline.
		return &logInfo{
			Name: name,
			HTML: template.HTML(data.Contents),
		}, nil
	default:
		return &logInfo{
			Name:     name,
			NotFound: true,
		}, nil
	}
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

	data, err := s.contentAddressableStorageBlobAccess.Get(ctx, digest).ToByteSlice(maximumLogSizeBytes)
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

func (s *BrowserService) handleActionCommon(w http.ResponseWriter, req *http.Request, digest digest.Digest, executeResponse *remoteexecution.ExecuteResponse) {
	instance := digest.GetInstance()
	actionInfo := struct {
		Instance string
		Action   *remoteexecution.Action

		Command *remoteexecution.Command

		ExecuteResponse *remoteexecution.ExecuteResponse
		StdoutInfo      *logInfo
		StderrInfo      *logInfo

		InputRoot *directoryInfo

		OutputDirectories  []*remoteexecution.OutputDirectory
		OutputSymlinks     []*remoteexecution.OutputSymlink
		OutputFiles        []*remoteexecution.OutputFile
		MissingDirectories []string
		MissingFiles       []string
	}{
		Instance:        instance,
		ExecuteResponse: executeResponse,
	}

	ctx := extractContextFromRequest(req)
	actionResult := executeResponse.GetResult()
	if actionResult != nil {
		actionInfo.OutputDirectories = actionResult.OutputDirectories
		actionInfo.OutputSymlinks = actionResult.OutputFileSymlinks
		actionInfo.OutputFiles = actionResult.OutputFiles

		var err error
		actionInfo.StdoutInfo, err = s.getLogInfoFromActionResult(ctx, "Standard output", instance, actionResult.StdoutDigest, actionResult.StdoutRaw)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		actionInfo.StderrInfo, err = s.getLogInfoFromActionResult(ctx, "Standard error", instance, actionResult.StderrDigest, actionResult.StderrRaw)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	action, err := s.contentAddressableStorage.GetAction(ctx, digest)
	if err == nil {
		actionInfo.Action = action

		commandDigest, err := digest.NewDerivedDigest(action.CommandDigest)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		command, err := s.contentAddressableStorage.GetCommand(ctx, commandDigest)
		if err == nil {
			actionInfo.Command = command

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
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		inputRootDigest, err := digest.NewDerivedDigest(action.InputRootDigest)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		directory, err := s.contentAddressableStorage.GetDirectory(ctx, inputRootDigest)
		if err == nil {
			actionInfo.InputRoot = &directoryInfo{
				Digest:    inputRootDigest,
				Directory: directory,
			}
		} else if status.Code(err) != codes.NotFound {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	} else if status.Code(err) != codes.NotFound {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if action == nil && actionResult == nil {
		http.Error(w, "Could not find an action or action result", http.StatusNotFound)
		return
	}

	if err := s.templates.ExecuteTemplate(w, "page_action.html", actionInfo); err != nil {
		log.Print(err)
	}
}

func (s *BrowserService) handleCommand(w http.ResponseWriter, req *http.Request) {
	digest, err := getDigestFromRequest(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := extractContextFromRequest(req)
	command, err := s.contentAddressableStorage.GetCommand(ctx, digest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.templates.ExecuteTemplate(w, "page_command.html", command); err != nil {
		log.Print(err)
	}
}

func (s *BrowserService) generateTarballDirectory(ctx context.Context, w *tar.Writer, digest digest.Digest, directory *remoteexecution.Directory, directoryPath string, getDirectory func(context.Context, digest.Digest) (*remoteexecution.Directory, error)) error {
	// Emit child directories.
	for _, directoryNode := range directory.Directories {
		childPath := path.Join(directoryPath, directoryNode.Name)
		if err := w.WriteHeader(&tar.Header{
			Typeflag: tar.TypeDir,
			Name:     childPath,
			Mode:     0777,
		}); err != nil {
			return err
		}
		childDigest, err := digest.NewDerivedDigest(directoryNode.Digest)
		if err != nil {
			return err
		}
		childDirectory, err := getDirectory(ctx, childDigest)
		if err != nil {
			return err
		}
		if err := s.generateTarballDirectory(ctx, w, childDigest, childDirectory, childPath, getDirectory); err != nil {
			return err
		}
	}

	// Emit symlinks.
	for _, symlinkNode := range directory.Symlinks {
		childPath := path.Join(directoryPath, symlinkNode.Name)
		if err := w.WriteHeader(&tar.Header{
			Typeflag: tar.TypeSymlink,
			Name:     childPath,
			Linkname: symlinkNode.Target,
			Mode:     0777,
		}); err != nil {
			return err
		}
	}

	// Emit regular files.
	for _, fileNode := range directory.Files {
		childPath := path.Join(directoryPath, fileNode.Name)
		mode := int64(0666)
		if fileNode.IsExecutable {
			mode = 0777
		}
		if err := w.WriteHeader(&tar.Header{
			Typeflag: tar.TypeReg,
			Name:     childPath,
			Size:     fileNode.Digest.SizeBytes,
			Mode:     mode,
		}); err != nil {
			return err
		}

		childDigest, err := digest.NewDerivedDigest(fileNode.Digest)
		if err != nil {
			return err
		}
		if err := s.contentAddressableStorageBlobAccess.Get(ctx, childDigest).IntoWriter(w); err != nil {
			return err
		}
	}
	return nil
}

func (s *BrowserService) generateTarball(ctx context.Context, w http.ResponseWriter, digest digest.Digest, directory *remoteexecution.Directory, getDirectory func(context.Context, digest.Digest) (*remoteexecution.Directory, error)) {
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.tar.gz\"", digest.GetHashString()))
	w.Header().Set("Content-Type", "application/gzip")
	gzipWriter := gzip.NewWriter(w)
	tarWriter := tar.NewWriter(gzipWriter)
	if err := s.generateTarballDirectory(ctx, tarWriter, digest, directory, "", getDirectory); err != nil {
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
	digest, err := getDigestFromRequest(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := extractContextFromRequest(req)
	directory, err := s.contentAddressableStorage.GetDirectory(ctx, digest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.URL.Query().Get("format") == "tar" {
		s.generateTarball(ctx, w, digest, directory, s.contentAddressableStorage.GetDirectory)
	} else {
		if err := s.templates.ExecuteTemplate(w, "page_directory.html", directoryInfo{
			Digest:    digest,
			Directory: directory,
		}); err != nil {
			log.Print(err)
		}
	}
}

func (s *BrowserService) handleFile(w http.ResponseWriter, req *http.Request) {
	digest, err := getDigestFromRequest(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := extractContextFromRequest(req)
	r := s.contentAddressableStorageBlobAccess.Get(ctx, digest).ToReader()
	defer r.Close()

	// Attempt to read the first chunk of data to see whether we can
	// trigger an error. Only when no error occurs, we start setting
	// response headers.
	var first [4096]byte
	n, err := r.Read(first[:])
	if err != nil && err != io.EOF {
		// TODO(edsch): Convert error code.
		http.Error(w, err.Error(), http.StatusBadRequest)
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

func (s *BrowserService) handleTree(w http.ResponseWriter, req *http.Request) {
	treeDigest, err := getDigestFromRequest(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := extractContextFromRequest(req)
	tree, err := s.contentAddressableStorage.GetTree(ctx, treeDigest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	treeInfo := struct {
		Instance           string
		Directory          *remoteexecution.Directory
		HasParentDirectory bool
	}{
		Instance:  treeDigest.GetInstance(),
		Directory: tree.Root,
	}

	// Construct map of all child directories.
	children := map[string]*remoteexecution.Directory{}
	for _, child := range tree.Children {
		data, err := proto.Marshal(child)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		digestGenerator := treeDigest.NewGenerator()
		if _, err := digestGenerator.Write(data); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		children[digestGenerator.Sum().GetKey(digest.KeyWithoutInstance)] = child
	}

	// In case additional directory components are provided, we need
	// to traverse the directories stored within.
	directoryDigest := treeDigest
	for _, component := range strings.FieldsFunc(
		mux.Vars(req)["subdirectory"],
		func(r rune) bool { return r == '/' }) {
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
			http.Error(w, "Subdirectory in tree not found", http.StatusNotFound)
			return
		}

		// Find corresponding child directory message.
		directoryDigest, err = directoryDigest.NewDerivedDigest(childNode.Digest)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		childDirectory, ok := children[directoryDigest.GetKey(digest.KeyWithoutInstance)]
		if !ok {
			http.Error(w, "Failed to find child node in tree", http.StatusBadRequest)
			return
		}
		treeInfo.HasParentDirectory = true
		treeInfo.Directory = childDirectory
	}

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
