package files

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/glacier"
	"github.com/mrdunski/accumulation-zone/model"
	"io"
	"os"
	"path"
	"strings"
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

type Volume struct {
	os       FileAccess
	basePath string
	excludes []string
}

func NewLoader(basePath string, excludes ...string) Volume {
	return Volume{os: stdOSInstance, basePath: basePath, excludes: excludes}
}

func (l Volume) isExcluded(path string) bool {
	for _, exclude := range l.excludes {
		if strings.Contains(path, exclude) {
			return true
		}
	}

	return false
}

func (l Volume) LoadFile(subPath string) (_ TreeHashedFile, err error) {
	file, err := os.Open(path.Join(l.basePath, subPath))
	if err != nil {
		return
	}
	defer func(file *os.File) {
		closeErr := file.Close()
		if closeErr != nil && err == nil {
			err = closeErr
		}
	}(file)
	hash := glacier.ComputeHashes(file)

	if len(hash.TreeHash) == 0 {
		hash = glacier.ComputeHashes(strings.NewReader(""))
	}

	return TreeHashedFile{
		path:     subPath,
		treeHash: fmt.Sprintf("%x", hash.TreeHash),
		os:       l.os,
		basePath: l.basePath,
	}, nil
}

func (l Volume) createDirIfNotExist(content model.FileWithContent) error {
	dirPath := path.Join(l.basePath, path.Dir(content.Path()))
	return os.MkdirAll(dirPath, 0700)
}

func (l Volume) Save(content model.FileWithContent) (err error) {
	reader, err := content.Content()
	if err != nil {
		return
	}
	defer func(closer io.Closer) {
		closeErr := closer.Close()
		if closeErr != nil && err == nil {
			err = closeErr
		}
	}(reader)

	err = l.createDirIfNotExist(content)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(path.Join(l.basePath, content.Path()), os.O_TRUNC|os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return
	}
	defer func(file *os.File) {
		closeErr := file.Close()
		if closeErr != nil && err == nil {
			err = closeErr
		}
	}(file)

	_, err = io.Copy(file, reader)
	if err != nil {
		return err
	}

	return nil
}

func (l Volume) loadEntry(entrySubPath string, entry os.DirEntry) ([]model.FileWithContent, error) {
	if l.isExcluded(entrySubPath) {
		return nil, nil
	}

	if entry.IsDir() {
		dirEntries, err := l.loadSubPath(entrySubPath)
		if err != nil {
			return nil, err
		}
		return dirEntries, nil
	}

	fileHandle, err := l.LoadFile(entrySubPath)
	if err != nil {
		return nil, err
	}
	return []model.FileWithContent{fileHandle}, nil
}

func (l Volume) loadSubPath(subPath string) ([]model.FileWithContent, error) {
	absolutePath := path.Join(l.basePath, subPath)
	entries, err := os.ReadDir(absolutePath)
	if err != nil {
		return nil, err
	}

	var result []model.FileWithContent

	for _, entry := range entries {
		entrySubPath := path.Join(subPath, entry.Name())
		entryFiles, err := l.loadEntry(entrySubPath, entry)
		if err != nil {
			return nil, err
		}
		if len(entryFiles) > 0 {
			result = append(result, entryFiles...)
		}
	}

	return result, nil
}

func (l Volume) LoadTree() ([]model.FileWithContent, error) {
	return l.loadSubPath("")
}
