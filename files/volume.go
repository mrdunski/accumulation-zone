package files

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/glacier"
	"github.com/mrdunski/accumulation-zone/logger"
	"github.com/mrdunski/accumulation-zone/model"
	"io"
	"os"
	"path"
	"strings"
)

// Volume is a backup source
type Volume struct {
	os       FileAccess
	basePath string
	excludes []string
}

// NewVolume creates a volume for specified path. excludes define paths that should not be synchronized.
func NewVolume(basePath string, excludes ...string) Volume {
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

// LoadFile loads specified file
func (l Volume) LoadFile(subPath string) (_ TreeHashedFile, err error) {
	logger.WithComponent("volume").Debugf("Loading %s/%s", l.basePath, subPath)
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
		logger.WithComponent("volume").Warnf("Empty file %s/%s - hash will be empty as well", l.basePath, subPath)
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

// Save stores a file in volume
func (l Volume) Save(content model.FileWithContent) (err error) {
	logger.WithComponent("volume").Debugf("Saving file: %s/%s", l.basePath, content.Path())
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
	logger.WithComponent("volume").Debugf("Loading %s/%s", l.basePath, subPath)
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

// LoadTree loads all files from the Volume
func (l Volume) LoadTree() ([]model.FileWithContent, error) {
	return l.loadSubPath("")
}
