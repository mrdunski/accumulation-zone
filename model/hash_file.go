//go:generate mockgen -destination=mock_model/hash_file.go . HashedFile,FileWithContent,ChangeIdHolder,IdentifiableHashedFile
package model

import "io"

type HashedFile interface {
	Path() string
	Hash() string
}

type FileWithContent interface {
	HashedFile
	Content() (io.ReadCloser, error)
}

type ChangeIdHolder interface {
	ChangeId() string
}

type IdentifiableHashedFile interface {
	HashedFile
	ChangeIdHolder
}

type HashedFiles map[string]HashedFile

func (e HashedFiles) HasFile(path, hash string) bool {
	file, ok := e[path]
	if !ok {
		return false
	}

	return file.Hash() == hash
}

func (e HashedFiles) Replace(file HashedFile) {
	e[file.Path()] = file
}
