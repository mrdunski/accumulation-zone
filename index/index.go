package index

import (
	"errors"
	"github.com/mrdunski/accumulation-zone/logger"
	"github.com/mrdunski/accumulation-zone/model"
	"github.com/sirupsen/logrus"
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

// Index tracks changes made on files
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

// LoadIndexFile loads entries from specified path or creates new index file
func LoadIndexFile(filePath string) (Index, error) {
	logger.WithComponent("index").Debugf("Loading index: %s", filePath)
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

// CalculateChanges for given list of files, finds model.Changes for those files comparing to version stored in Index
func (i Index) CalculateChanges(files []model.FileWithContent) model.Changes {
	switch {
	case logger.Get().IsLevelEnabled(logrus.DebugLevel):
		logger.WithComponent("index").Debugf("Calculating changed files for %d files", len(files))
	case logger.Get().IsLevelEnabled(logrus.TraceLevel):
		logger.WithComponent("index").Debugf("Calculating changed files for %v files", files)
	}

	changes := model.Changes{}
	existing := model.HashedFiles{}

	for _, file := range files {
		existing.Replace(file)
		if i.IsChanged(file) {
			changes.Additions = append(changes.Additions, model.FileAdded{FileWithContent: file})
		}
	}

	for _, pathEntries := range i.entries {
		for _, pathEntry := range pathEntries {
			if !existing.HasFile(pathEntry.path, pathEntry.hash) {
				changes.Deletions = append(changes.Deletions, model.FileDeleted{IdentifiableHashedFile: pathEntry})
			}
		}
	}

	switch {
	case logger.Get().IsLevelEnabled(logrus.DebugLevel):
		logger.WithComponent("index").Debugf("Calculated changes. Added: %d, Removed: %d", len(changes.Additions), len(changes.Deletions))
	case logger.Get().IsLevelEnabled(logrus.TraceLevel):
		logger.WithComponent("index").Debugf("Calculated changes. %v", changes)
	}
	return changes
}

func (i Index) IsChanged(file model.FileWithContent) bool {
	return !i.entries.hasEntryWithHash(file.Path(), file.Hash())
}

func (i Index) CommitAdd(changeId string, file model.HashedFile) error {
	logger.WithComponent("index").Debugf("Commiting add %s %s", changeId, file.Path())
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
	logger.WithComponent("index").Debugf("Commiting delete %s %s", changeId, file.Path())
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
	logger.WithComponent("index").Debugf("Clearing index")
	for k := range i.entries {
		delete(i.entries, k)
	}

	return i.committer.clear()
}
