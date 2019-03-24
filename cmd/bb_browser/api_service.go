package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
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
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/gorilla/mux"
	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
)

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

type APIService struct {
	marshaler                           jsonpb.Marshaler
	contentAddressableStorage           cas.ContentAddressableStorage
	contentAddressableStorageBlobAccess blobstore.BlobAccess
}

func NewAPIService(contentAddressableStorage cas.ContentAddressableStorage, contentAddressableStorageBlobAccess blobstore.BlobAccess, actionCache ac.ActionCache, router *mux.Router) *APIService {
	s := &APIService{
		contentAddressableStorage:           contentAddressableStorage,
		contentAddressableStorageBlobAccess: contentAddressableStorageBlobAccess,
	}
	router.HandleFunc("/api/get_action", s.handleGetObject(
		func(ctx context.Context, digest *util.Digest) (proto.Message, error) {
			return contentAddressableStorage.GetAction(ctx, digest)
		}))
	router.HandleFunc("/api/get_action_result", s.handleGetObject(
		func(ctx context.Context, digest *util.Digest) (proto.Message, error) {
			return actionCache.GetActionResult(ctx, digest)
		}))
	router.HandleFunc("/api/get_command", s.handleGetObject(
		func(ctx context.Context, digest *util.Digest) (proto.Message, error) {
			return contentAddressableStorage.GetCommand(ctx, digest)
		}))
	router.HandleFunc("/api/get_directory", s.handleGetObject(
		func(ctx context.Context, digest *util.Digest) (proto.Message, error) {
			return contentAddressableStorage.GetDirectory(ctx, digest)
		}))
	router.HandleFunc("/api/get_directory_tarball", s.handleGetDirectoryTarball)
	router.HandleFunc("/api/get_file", s.handleGetFile)
	router.HandleFunc("/api/get_tree", s.handleGetObject(
		func(ctx context.Context, digest *util.Digest) (proto.Message, error) {
			return contentAddressableStorage.GetTree(ctx, digest)
		}))
	router.HandleFunc("/api/get_tree_tarball", s.handleGetTreeTarball)
	router.HandleFunc("/api/get_uncached_action_result", s.handleGetObject(
		func(ctx context.Context, digest *util.Digest) (proto.Message, error) {
			return contentAddressableStorage.GetUncachedActionResult(ctx, digest)
		}))
	return s
}

func getDigestFromQueryParameters(req *http.Request) (*util.Digest, error) {
	vars := req.URL.Query()
	sizeBytes, err := strconv.ParseInt(vars.Get("size_bytes"), 10, 64)
	if err != nil {
		return nil, err
	}
	return util.NewDigest(
		vars.Get("instance"),
		&remoteexecution.Digest{
			Hash:      vars.Get("hash"),
			SizeBytes: sizeBytes,
		})
}

func (s *APIService) handleGetObject(getter func(ctx context.Context, digest *util.Digest) (proto.Message, error)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		digest, err := getDigestFromQueryParameters(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		message, err := getter(extractContextFromRequest(req), digest)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := s.marshaler.Marshal(w, message); err != nil {
			log.Print(err)
		}
	}
}

func (s *APIService) generateTarballDirectory(ctx context.Context, w *tar.Writer, digest *util.Digest, directory *remoteexecution.Directory, directoryPath string, getDirectory func(context.Context, *util.Digest) (*remoteexecution.Directory, error)) error {
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
		if err := w.WriteHeader(&tar.Header{Typeflag: tar.TypeSymlink,
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

func (s *APIService) generateTarball(ctx context.Context, w http.ResponseWriter, digest *util.Digest, directory *remoteexecution.Directory, getDirectory func(context.Context, *util.Digest) (*remoteexecution.Directory, error)) {
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

func (s *APIService) handleGetDirectoryTarball(w http.ResponseWriter, req *http.Request) {
	digest, err := getDigestFromQueryParameters(req)
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

	s.generateTarball(ctx, w, digest, directory, s.contentAddressableStorage.GetDirectory)
}

func (s *APIService) handleGetFile(w http.ResponseWriter, req *http.Request) {
	digest, err := getDigestFromQueryParameters(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := extractContextFromRequest(req)
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

	// TODO(edsch): Set the preferred filename!
	// TODO(edsch): Use the function from net/http to sniff the
	// content type.
	w.Header().Set("Content-Length", strconv.FormatInt(digest.GetSizeBytes(), 10))
	if utf8.ValidString(string(first[:])) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	w.Write(first[:n])
	io.Copy(w, r)
}

func (s *APIService) handleGetTreeTarball(w http.ResponseWriter, req *http.Request) {
	digest, err := getDigestFromQueryParameters(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := extractContextFromRequest(req)
	tree, err := s.contentAddressableStorage.GetTree(ctx, digest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
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

	// Look up the directory within the tree that should be served.
	rootDirectory := tree.Root
	for _, component := range strings.FieldsFunc(
		req.URL.Query().Get("subdirectory"),
		func(r rune) bool { return r == '/' }) {
		// Find child with matching name.
		childNode := func() *remoteexecution.DirectoryNode {
			for _, directoryNode := range rootDirectory.Directories {
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
		rootDirectory = childDirectory
	}

	s.generateTarball(
		ctx, w, digest, rootDirectory,
		func(ctx context.Context, digest *util.Digest) (*remoteexecution.Directory, error) {
			childDirectory, ok := children[digest.GetKey(util.DigestKeyWithoutInstance)]
			if !ok {
				return nil, errors.New("Failed to find child node in tree")
			}
			return childDirectory, nil
		})
}
