package files

import (
	"fmt"
	"io"
	"os"
	"path"
)

type stdOS struct{}

func (f stdOS) ReadDir(name string) ([]os.DirEntry, error) {
	return os.ReadDir(name)
}

func (f stdOS) Open(name string) (io.ReadCloser, error) {
	return os.Open(name)
}

var stdOSInstance = stdOS{}

type FileAccess interface {
	ReadDir(name string) ([]os.DirEntry, error)
	Open(name string) (io.ReadCloser, error)
}

type TreeHashedFile struct {
	os       FileAccess
	basePath string
	path     string
	treeHash string
}

func (fh TreeHashedFile) Path() string {
	return fh.path
}

func (fh TreeHashedFile) Content() (io.ReadCloser, error) {
	return os.Open(path.Join(fh.basePath, fh.path))
}

func (fh TreeHashedFile) Equal(other TreeHashedFile) bool {
	return fh.path == other.path && fh.treeHash == other.treeHash
}

func (fh TreeHashedFile) String() string {
	return fmt.Sprintf("{path: %s, treeHash: %s}", fh.path, fh.treeHash)
}

func (fh TreeHashedFile) Hash() string {
	return fh.treeHash
}
