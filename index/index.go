package index

import (
	"errors"
	"github.com/mrdunski/accumulation-zone/model"
)

type committer interface {
	add(entry Entry) error
	remove(entry Entry) error
	clear() error
}

type voidCommitter struct{}

func (v voidCommitter) clear() error {
	return nil
}

func (v voidCommitter) add(_ Entry) error {
	return nil
}

func (v voidCommitter) remove(_ Entry) error {
	return nil
}

type Index struct {
	committer
	entries entries
}

func New(entryList []Entry) Index {
	index := Index{
		entries:   entries{},
		committer: voidCommitter{},
	}
	for _, entry := range entryList {
		index.entries.add(entry)
	}

	return index
}

func LoadIndexFile(filePath string) (Index, error) {
	records := fileRecords{filePath: filePath}
	data, err := records.loadEntries()
	if err != nil {
		return Index{}, err
	}

	return Index{
		committer: records,
		entries:   data,
	}, nil
}

func (i Index) CalculateChanges(files []model.FileWithContent) model.Changes {
	changes := model.Changes{}
	existing := model.HashedFiles{}

	for _, file := range files {
		existing.Replace(file)
		changes.Append(i.CalculateChange(file))
	}

	for _, pathEntries := range i.entries {
		for _, pathEntry := range pathEntries {
			if !existing.HasFile(pathEntry.path, pathEntry.hash) {
				changes.Deletions = append(changes.Deletions, model.FileDeleted{IdentifiableHashedFile: pathEntry})
			}
		}
	}

	return changes
}

func (i Index) CalculateChange(file model.FileWithContent) model.Changes {
	if !i.entries.hasEntryWithHash(file.Path(), file.Hash()) {
		return model.Changes{
			Additions: []model.FileAdded{{file}},
		}
	}

	return model.Changes{}
}

func (i Index) CommitAdd(changeId string, file model.HashedFile) error {
	if i.entries.hasEntryWithChangeId(file.Path(), changeId) && changeId != "" {
		return errors.New("file already exist")
	}
	entry := NewEntry(file.Path(), file.Hash(), changeId)
	if err := i.add(entry); err != nil {
		return err
	}
	i.entries.add(entry)

	return nil
}

func (i Index) CommitDelete(changeId string, file model.HashedFile) error {
	if !i.entries.hasEntryWithChangeId(file.Path(), changeId) {
		return errors.New("change doesn't exist")
	}
	entry := NewEntry(file.Path(), file.Hash(), changeId)
	if err := i.remove(entry); err != nil {
		return err
	}
	i.entries.deleteEntryByChangeId(file.Path(), changeId)

	return nil
}

func (i Index) Clear() error {
	for k := range i.entries {
		delete(i.entries, k)
	}

	return i.committer.clear()
}
