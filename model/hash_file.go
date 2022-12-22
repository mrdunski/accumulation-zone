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
type IdentifiableHashedFiles map[string]IdentifiableHashedFile

func (e HashedFiles) HasFile(path, hash string) bool {
	return hasFile(e, path, hash)
}

func (e IdentifiableHashedFiles) Replace(file IdentifiableHashedFile) {
	replace(e, file)
}

func (e IdentifiableHashedFiles) HasFile(path, hash string) bool {
	return hasFile(e, path, hash)
}

func (e HashedFiles) Replace(file HashedFile) {
	replace(e, file)
}

func hasFile[E HashedFile](e map[string]E, path, hash string) bool {
	file, ok := e[path]
	if !ok {
		return false
	}

	return file.Hash() == hash
}

func replace[E HashedFile](e map[string]E, file E) {
	e[file.Path()] = file
}

func NewIdentifiableHashedFiles[E IdentifiableHashedFile](files []E) IdentifiableHashedFiles {
	result := IdentifiableHashedFiles{}
	for _, file := range files {
		result.Replace(file)
	}

	return result
}
