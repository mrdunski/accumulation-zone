package model

type HashedFile interface {
	Path() string
	Hash() string
}

type HashedFiles map[string]HashedFile

func (e HashedFiles) HasFile(path, hash string) bool {
	file, ok := e[path]
	if !ok {
		return false
	}

	return file.Hash() == hash
}

func (e HashedFiles) Add(file HashedFile) {
	e[file.Path()] = file
}

func AsHashedFiles[HF HashedFile](a []HF) []HashedFile {
	result := make([]HashedFile, 0, len(a))
	for _, hf := range a {
		result = append(result, hf)
	}

	return result
}
