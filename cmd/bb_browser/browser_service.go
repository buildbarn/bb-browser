package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"path"
	"strconv"
	"strings"
	"unicode/utf8"

	remoteexecution "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
	"github.com/buildbarn/bb-storage/pkg/blobstore"
	"github.com/buildbarn/bb-storage/pkg/digest"
	cas_proto "github.com/buildbarn/bb-storage/pkg/proto/cas"
	"github.com/buildbarn/bb-storage/pkg/util"
	"github.com/buildkite/terminal"
	"github.com/golang/protobuf/proto"
	"github.com/gorilla/mux"
	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func getDigestFromRequest(req *http.Request) (digest.Digest, error) {
	vars := mux.Vars(req)
	instanceName, err := digest.NewInstanceName(vars["instanceName"])
	if err != nil {
		return digest.BadDigest, util.StatusWrapf(err, "Invalid instance name %#v", vars["instanceName"])
	}
	sizeBytes, err := strconv.ParseInt(vars["sizeBytes"], 10, 64)
	if err != nil {
		return digest.BadDigest, util.StatusWrapf(err, "Invalid blob size %#v", vars["sizeBytes"])
	}
	return instanceName.NewDigest(vars["hash"], sizeBytes)
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
	contentAddressableStorage blobstore.BlobAccess
	actionCache               blobstore.BlobAccess
	maximumMessageSizeBytes   int
	templates                 *template.Template
}

// NewBrowserService constructs a BrowserService that accesses storage
// through a set of handles.
func NewBrowserService(contentAddressableStorage blobstore.BlobAccess, actionCache blobstore.BlobAccess, maximumMessageSizeBytes int, templates *template.Template, router *mux.Router) *BrowserService {
	s := &BrowserService{
		contentAddressableStorage: contentAddressableStorage,
		actionCache:               actionCache,
		maximumMessageSizeBytes:   maximumMessageSizeBytes,
		templates:                 templates,
	}
	router.HandleFunc("/", s.handleWelcome)
	router.HandleFunc("/action/{instanceName}/{hash}/{sizeBytes}/", s.handleAction)
	router.HandleFunc("/command/{instanceName}/{hash}/{sizeBytes}/", s.handleCommand)
	router.HandleFunc("/directory/{instanceName}/{hash}/{sizeBytes}/", s.handleDirectory)
	router.HandleFunc("/file/{instanceName}/{hash}/{sizeBytes}/{name}", s.handleFile)
	router.HandleFunc("/tree/{instanceName}/{hash}/{sizeBytes}/{subdirectory:(?:.*/)?}", s.handleTree)
	router.HandleFunc("/uncached_action_result/{instanceName}/{hash}/{sizeBytes}/", s.handleUncachedActionResult)
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
	var actionResult *remoteexecution.ActionResult
	if m, err := s.actionCache.Get(ctx, digest).ToProto(
		&remoteexecution.ActionResult{},
		s.maximumMessageSizeBytes); err == nil {
		actionResult = m.(*remoteexecution.ActionResult)
	} else if status.Code(err) != codes.NotFound {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.handleActionCommon(w, req, digest, &remoteexecution.ExecuteResponse{
		Result: actionResult,
	}, false)
}

func (s *BrowserService) handleUncachedActionResult(w http.ResponseWriter, req *http.Request) {
	digest, err := getDigestFromRequest(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	ctx := extractContextFromRequest(req)
	m, err := s.contentAddressableStorage.Get(ctx, digest).ToProto(&cas_proto.UncachedActionResult{}, s.maximumMessageSizeBytes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	uncachedActionResult := m.(*cas_proto.UncachedActionResult)
	actionDigest, err := digest.GetInstanceName().NewDigestFromProto(uncachedActionResult.ActionDigest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.handleActionCommon(w, req, actionDigest, uncachedActionResult.ExecuteResponse, true)
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

func (s *BrowserService) handleActionCommon(w http.ResponseWriter, req *http.Request, actionDigest digest.Digest, executeResponse *remoteexecution.ExecuteResponse, isUncachedActionResult bool) {
	instanceName := actionDigest.GetInstanceName()
	actionInfo := struct {
		IsUncachedActionResult bool
		ActionDigest           digest.Digest
		Action                 *remoteexecution.Action

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
		IsUncachedActionResult: isUncachedActionResult,
		ActionDigest:           actionDigest,
		ExecuteResponse:        executeResponse,
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
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		actionInfo.StderrInfo, err = s.getLogInfoFromActionResult(ctx, "Standard error", instanceName, actionResult.StderrDigest, actionResult.StderrRaw)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	actionMessage, err := s.contentAddressableStorage.Get(ctx, actionDigest).ToProto(&remoteexecution.Action{}, s.maximumMessageSizeBytes)
	if err == nil {
		action := actionMessage.(*remoteexecution.Action)
		actionInfo.Action = action

		commandDigest, err := instanceName.NewDigestFromProto(action.CommandDigest)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		commandMessage, err := s.contentAddressableStorage.Get(ctx, commandDigest).ToProto(&remoteexecution.Command{}, s.maximumMessageSizeBytes)
		if err == nil {
			command := commandMessage.(*remoteexecution.Command)
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

		inputRootDigest, err := instanceName.NewDigestFromProto(action.InputRootDigest)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		directoryMessage, err := s.contentAddressableStorage.Get(ctx, inputRootDigest).ToProto(&remoteexecution.Directory{}, s.maximumMessageSizeBytes)
		if err == nil {
			actionInfo.InputRoot = &directoryInfo{
				Digest:    inputRootDigest,
				Directory: directoryMessage.(*remoteexecution.Directory),
			}
		} else if status.Code(err) != codes.NotFound {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	} else if status.Code(err) != codes.NotFound {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if actionMessage == nil && actionResult == nil {
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
	commandMessage, err := s.contentAddressableStorage.Get(ctx, digest).ToProto(&remoteexecution.Command{}, s.maximumMessageSizeBytes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	command := commandMessage.(*remoteexecution.Command)

	if err := s.templates.ExecuteTemplate(w, "page_command.html", command); err != nil {
		log.Print(err)
	}
}

func (s *BrowserService) generateTarballDirectory(ctx context.Context, w *tar.Writer, instanceName digest.InstanceName, directory *remoteexecution.Directory, directoryPath string, getDirectory func(context.Context, digest.Digest) (*remoteexecution.Directory, error), filesSeen map[string]string) error {
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

		childPath := path.Join(directoryPath, fileNode.Name)
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
				Name:     childPath,
				Linkname: linkPath,
			}); err != nil {
				return err
			}
		} else {
			// This is the first time we're returning this
			// file. Actually add it to the archive.
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

			if err := s.contentAddressableStorage.Get(ctx, childDigest).IntoWriter(w); err != nil {
				return err
			}

			filesSeen[childKey] = childPath
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
	if err := s.generateTarballDirectory(ctx, tarWriter, digest.GetInstanceName(), directory, "", getDirectory, filesSeen); err != nil {
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
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := extractContextFromRequest(req)
	directoryMessage, err := s.contentAddressableStorage.Get(ctx, directoryDigest).ToProto(&remoteexecution.Directory{}, s.maximumMessageSizeBytes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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
			Digest:    directoryDigest,
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
	r := s.contentAddressableStorage.Get(ctx, digest).ToReader()
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
	treeMessage, err := s.contentAddressableStorage.Get(ctx, treeDigest).ToProto(&remoteexecution.Tree{}, s.maximumMessageSizeBytes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	tree := treeMessage.(*remoteexecution.Tree)
	instanceName := treeDigest.GetInstanceName()
	treeInfo := struct {
		InstanceName       digest.InstanceName
		Directory          *remoteexecution.Directory
		HasParentDirectory bool
	}{
		InstanceName: instanceName,
		Directory:    tree.Root,
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
		directoryDigest, err = instanceName.NewDigestFromProto(childNode.Digest)
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
