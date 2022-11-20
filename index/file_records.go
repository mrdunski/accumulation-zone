package index

import (
	"bufio"
	"encoding/json"
	"errors"
	"github.com/mrdunski/accumulation-zone/model"
	"os"
	"time"
)

type record struct {
	OperationType model.ChangeType `json:"type"`
	Path          string           `json:"path"`
	Hash          string           `json:"hash"`
	ChangeId      string           `json:"id"`
	Time          time.Time        `json:"time"`
}

type fileRecords struct {
	filePath string
}

func (f fileRecords) add(entry Entry) error {
	r := record{
		OperationType: model.Added,
		Path:          entry.path,
		Hash:          entry.hash,
		Time:          entry.recordDate,
		ChangeId:      entry.changeId,
	}

	return f.writeRecord(r)
}

func (f fileRecords) remove(entry Entry) error {
	r := record{
		OperationType: model.Deleted,
		Path:          entry.path,
		Hash:          entry.hash,
		Time:          entry.recordDate,
		ChangeId:      entry.changeId,
	}

	return f.writeRecord(r)
}

func (f fileRecords) writeRecord(r record) (err error) {
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	file, err := f.openOrCreate()
	if err != nil {
		return err
	}

	defer func(file *os.File) {
		err = file.Close()
	}(file)

	_, err = file.Write(data)
	if err != nil {
		return err
	}

	_, err = file.Write([]byte("\n"))
	return err
}

func (f fileRecords) loadEntries() (_ entries, err error) {
	file, err := f.openOrCreate()
	if err != nil {
		return nil, err
	}

	defer func(file *os.File) {
		err = file.Close()
	}(file)

	result := entries{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		r := record{}
		if err = json.Unmarshal(scanner.Bytes(), &r); err != nil {
			return nil, err
		}

		switch r.OperationType {
		case model.Added:
			result.add(Entry{
				hash:       r.Hash,
				path:       r.Path,
				changeId:   r.ChangeId,
				recordDate: r.Time,
			})
		case model.Deleted:
			result.deleteEntryByChangeId(r.Path, r.ChangeId)
		default:
			return nil, errors.New("unsupported record")
		}

	}

	if scanner.Err() != nil {
		return nil, scanner.Err()
	}

	return result, nil
}

func (f fileRecords) openOrCreate() (*os.File, error) {
	file, err := os.OpenFile(f.filePath, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return nil, err
	}

	return file, nil
}
