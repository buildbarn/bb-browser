package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"strconv"
	"strings"
	"unicode/utf8"

	remoteexecution "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
	"github.com/buildbarn/bb-storage/pkg/ac"
	"github.com/buildbarn/bb-storage/pkg/blobstore"
	"github.com/buildbarn/bb-storage/pkg/cas"
	"github.com/buildbarn/bb-storage/pkg/util"
	"github.com/buildkite/terminal"
	"github.com/golang/protobuf/proto"
	"github.com/gorilla/mux"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func getDigestFromRequest(req *http.Request) (*util.Digest, error) {
	vars := mux.Vars(req)
	sizeBytes, err := strconv.ParseInt(vars["sizeBytes"], 10, 64)
	if err != nil {
		return nil, err
	}
	return util.NewDigest(
		vars["instance"],
		&remoteexecution.Digest{
			Hash:      vars["hash"],
			SizeBytes: sizeBytes,
		})
}

// BrowserService implements a web service that can be used to explore
// data stored in the Content Addressable Storage and Action Cache. It
// can show the details of actions and download their input and output
// files.
type BrowserService struct {
	contentAddressableStorage           cas.ContentAddressableStorage
	contentAddressableStorageBlobAccess blobstore.BlobAccess
	actionCache                         ac.ActionCache
	templates                           *template.Template
}

// NewBrowserService constructs a BrowserService that accesses storage
// through a set of handles.
func NewBrowserService(contentAddressableStorage cas.ContentAddressableStorage, contentAddressableStorageBlobAccess blobstore.BlobAccess, actionCache ac.ActionCache, templates *template.Template, router *mux.Router) *BrowserService {
	s := &BrowserService{
		contentAddressableStorage:           contentAddressableStorage,
		contentAddressableStorageBlobAccess: contentAddressableStorageBlobAccess,
		actionCache:                         actionCache,
		templates:                           templates,
	}
	router.HandleFunc("/action/{instance}/{hash}/{sizeBytes}/", s.handleAction)
	router.HandleFunc("/uncached_action_result/{instance}/{hash}/{sizeBytes}/", s.handleUncachedActionResult)
	router.HandleFunc("/command/{instance}/{hash}/{sizeBytes}/", s.handleCommand)
	router.HandleFunc("/directory/{instance}/{hash}/{sizeBytes}/", s.handleDirectory)
	router.HandleFunc("/file/{instance}/{hash}/{sizeBytes}/{name}", s.handleFile)
	router.HandleFunc("/tree/{instance}/{hash}/{sizeBytes}/{subdirectory:(?:.*/)?}", s.handleTree)
	return s
}

type directoryInfo struct {
	Digest    *util.Digest
	Directory *remoteexecution.Directory
}

type logInfo struct {
	Name     string
	Instance string
	Digest   *remoteexecution.Digest
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

	ctx := req.Context()
	actionResult, err := s.actionCache.GetActionResult(ctx, digest)
	if err != nil && status.Code(err) != codes.NotFound {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.handleActionCommon(w, req, digest, actionResult)
}

func (s *BrowserService) handleUncachedActionResult(w http.ResponseWriter, req *http.Request) {
	digest, err := getDigestFromRequest(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	ctx := req.Context()
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
	s.handleActionCommon(w, req, actionDigest, uncachedActionResult.ActionResult)
}

func (s *BrowserService) getLogInfo(ctx context.Context, name string, instance string, logDigest *remoteexecution.Digest) (*logInfo, error) {
	if logDigest == nil {
		return nil, nil
	}
	digest, err := util.NewDigest(instance, logDigest)
	if err != nil {
		return nil, err
	}
	if size := digest.GetSizeBytes(); size == 0 {
		// No log file present.
		return nil, nil
	} else if size > 100000 {
		// Log file too large to show inline.
		return &logInfo{
			Name:     name,
			Instance: instance,
			Digest:   logDigest,
			TooLarge: true,
		}, nil
	}

	_, r, err := s.contentAddressableStorageBlobAccess.Get(ctx, digest)
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(r)
	r.Close()
	if err == nil {
		// Log found. Convert ANSI escape sequences to HTML.
		return &logInfo{
			Name:     name,
			Instance: instance,
			Digest:   logDigest,
			HTML:     template.HTML(terminal.Render(data)),
		}, nil
	} else if status.Code(err) == codes.NotFound {
		// Not found.
		return &logInfo{
			Name:     name,
			Instance: instance,
			Digest:   logDigest,
			NotFound: true,
		}, nil
	} else {
		return nil, err
	}
}

func (s *BrowserService) handleActionCommon(w http.ResponseWriter, req *http.Request, digest *util.Digest, actionResult *remoteexecution.ActionResult) {
	instance := digest.GetInstance()
	actionInfo := struct {
		Instance string
		Action   *remoteexecution.Action

		Command *remoteexecution.Command

		ActionResult *remoteexecution.ActionResult
		StdoutInfo   *logInfo
		StderrInfo   *logInfo

		InputRoot *directoryInfo

		OutputDirectories  []*remoteexecution.OutputDirectory
		OutputSymlinks     []*remoteexecution.OutputSymlink
		OutputFiles        []*remoteexecution.OutputFile
		MissingDirectories []string
		MissingFiles       []string
	}{
		Instance:     instance,
		ActionResult: actionResult,
	}

	ctx := req.Context()
	if actionResult != nil {
		actionInfo.OutputDirectories = actionResult.OutputDirectories
		actionInfo.OutputSymlinks = actionResult.OutputFileSymlinks
		actionInfo.OutputFiles = actionResult.OutputFiles

		// TODO(edsch): Should we support Std{out,err}Raw as well? Buildbarn doesn't generate them.
		var err error
		if actionResult.StdoutDigest != nil {
			actionInfo.StdoutInfo, err = s.getLogInfo(ctx, "Standard output", instance, actionResult.StdoutDigest)
		} else if actionResult.StdoutRaw != nil {
			actionInfo.StdoutInfo = &logInfo{
				Name:     "Standard output",
				Instance: instance,
				HTML:     template.HTML(terminal.Render(actionResult.StdoutRaw)),
			}
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if actionResult.StderrDigest != nil {
			actionInfo.StderrInfo, err = s.getLogInfo(ctx, "Standard error", instance, actionResult.StderrDigest)
		} else if actionResult.StderrRaw != nil {
			actionInfo.StderrInfo = &logInfo{
				Name:     "Standard error",
				Instance: instance,
				HTML:     template.HTML(terminal.Render(actionResult.StderrRaw)),
			}
		}
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

	ctx := req.Context()
	command, err := s.contentAddressableStorage.GetCommand(ctx, digest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.templates.ExecuteTemplate(w, "page_command.html", command); err != nil {
		log.Print(err)
	}
}

func (s *BrowserService) generateTarballDirectory(ctx context.Context, w *tar.Writer, digest *util.Digest, directory *remoteexecution.Directory, directoryPath string, getDirectory func(context.Context, *util.Digest) (*remoteexecution.Directory, error)) error {
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
		_, r, err := s.contentAddressableStorageBlobAccess.Get(ctx, childDigest)
		if err != nil {
			return err
		}
		_, err = io.Copy(w, r)
		r.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *BrowserService) generateTarball(ctx context.Context, w http.ResponseWriter, digest *util.Digest, directory *remoteexecution.Directory, getDirectory func(context.Context, *util.Digest) (*remoteexecution.Directory, error)) {
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.tar.gz\"", digest.GetHashString()))
	w.Header().Set("Content-Type", "application/gzip")
	gzipWriter := gzip.NewWriter(w)
	tarWriter := tar.NewWriter(gzipWriter)
	if err := s.generateTarballDirectory(ctx, tarWriter, digest, directory, "", getDirectory); err != nil {
		// TODO(edsch): Any way to propagate this to the client?
		log.Print(err)
		return
	}
	if err := tarWriter.Close(); err != nil {
		log.Print(err)
		return
	}
	if err := gzipWriter.Close(); err != nil {
		log.Print(err)
		return
	}
}

func (s *BrowserService) handleDirectory(w http.ResponseWriter, req *http.Request) {
	digest, err := getDigestFromRequest(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := req.Context()
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

	ctx := req.Context()
	_, r, err := s.contentAddressableStorageBlobAccess.Get(ctx, digest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
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
	digest, err := getDigestFromRequest(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := req.Context()
	tree, err := s.contentAddressableStorage.GetTree(ctx, digest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	treeInfo := struct {
		Instance           string
		Directory          *remoteexecution.Directory
		HasParentDirectory bool
	}{
		Instance:  digest.GetInstance(),
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
		digestGenerator := digest.NewDigestGenerator()
		if _, err := digestGenerator.Write(data); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		children[digestGenerator.Sum().GetKey(util.DigestKeyWithoutInstance)] = child
	}

	// In case additional directory components are provided, we need
	// to traverse the directories stored within.
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
		digest, err = digest.NewDerivedDigest(childNode.Digest)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		childDirectory, ok := children[digest.GetKey(util.DigestKeyWithoutInstance)]
		if !ok {
			http.Error(w, "Failed to find child node in tree", http.StatusBadRequest)
			return
		}
		treeInfo.HasParentDirectory = true
		treeInfo.Directory = childDirectory
	}

	if req.URL.Query().Get("format") == "tar" {
		s.generateTarball(
			ctx, w, digest, treeInfo.Directory,
			func(ctx context.Context, digest *util.Digest) (*remoteexecution.Directory, error) {
				childDirectory, ok := children[digest.GetKey(util.DigestKeyWithoutInstance)]
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
