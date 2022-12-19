package index

import (
	"errors"
	"github.com/mrdunski/accumulation-zone/model"
)

type committer interface {
	add(entry Entry) error
	remove(entry Entry) error
}

type voidCommitter struct{}

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

func (i Index) CalculateChanges(files []model.HashedFile) []model.Change {
	var changes []model.Change
	existing := model.HashedFiles{}

	for _, file := range files {
		existing.Add(file)
		if change, isChanged := i.CalculateChange(file); isChanged {
			changes = append(changes, change)
		}
	}

	for _, pathEntries := range i.entries {
		for _, pathEntry := range pathEntries {
			if !existing.HasFile(pathEntry.path, pathEntry.hash) {
				changes = append(changes, model.Change{
					ChangeType: model.Deleted,
					HashedFile: pathEntry,
				})
			}
		}
	}

	return changes
}

func (i Index) CalculateChange(file model.HashedFile) (model.Change, bool) {
	if !i.entries.hasEntryWithHash(file.Path(), file.Hash()) {
		return model.Change{
			HashedFile: file,
			ChangeType: model.Added,
		}, true
	}

	return model.Change{}, false
}

func (i Index) commitAdd(changeId string, change model.Change) error {
	if i.entries.hasEntryWithChangeId(change.Path(), changeId) && changeId != "" {
		return errors.New("change already exist")
	}
	entry := NewEntry(change.Path(), change.Hash(), changeId)
	if err := i.add(entry); err != nil {
		return err
	}
	i.entries.add(entry)

	return nil
}

func (i Index) commitDelete(changeId string, change model.Change) error {
	if !i.entries.hasEntryWithChangeId(change.Path(), changeId) {
		return errors.New("change doesn't exist")
	}
	entry := NewEntry(change.Path(), change.Hash(), changeId)
	if err := i.remove(entry); err != nil {
		return err
	}
	i.entries.deleteEntryByChangeId(change.Path(), changeId)

	return nil
}

func (i Index) CommitChange(changeId string, change model.Change) error {
	switch change.ChangeType {
	case model.Added:
		return i.commitAdd(changeId, change)
	case model.Deleted:
		return i.commitDelete(changeId, change)
	default:
		return errors.New("unsupported change")
	}
}
