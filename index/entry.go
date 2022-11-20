package index

import "time"

type Entry struct {
	hash       string
	path       string
	changeId   string
	recordDate time.Time
}

func NewEntry(path, hash, changeId string) Entry {
	return Entry{
		hash:       hash,
		path:       path,
		changeId:   changeId,
		recordDate: time.Now(),
	}
}

type entries map[string][]Entry

func emptySliceWithOptimisticCap(original []Entry) []Entry {
	if len(original) == 0 {
		return nil
	}
	return make([]Entry, 0, len(original)-1)
}

func (e Entry) Path() string {
	return e.path
}

func (e Entry) Hash() string {
	return e.hash
}

func (e Entry) ChangeId() string {
	return e.changeId
}

func (e entries) hasEntryWithHash(path, hash string) bool {
	return e.hasEntryMatching(path, func(e Entry) bool {
		return e.hash == hash
	})
}

func (e entries) hasEntryWithChangeId(path, changeId string) bool {
	return e.hasEntryMatching(path, func(e Entry) bool {
		return e.changeId == changeId
	})
}

func (e entries) hasEntryMatching(path string, filter func(e Entry) bool) bool {
	elements, ok := e[path]
	if !ok {
		return false
	}

	for _, entry := range elements {
		if filter(entry) {
			return true
		}
	}

	return false
}

func (e entries) deleteEntryByChangeId(path, changeId string) {
	elements, ok := e[path]
	if !ok {
		return
	}

	newElements := emptySliceWithOptimisticCap(elements)

	for _, entry := range elements {
		if entry.changeId != changeId {
			newElements = append(newElements, entry)
		}
	}

	e[path] = newElements
}

func (e entries) add(entry Entry) {
	pathEntries, _ := e[entry.path]
	e[entry.path] = append(pathEntries, entry)
}

func (e entries) flatten() []Entry {
	var result []Entry
	for _, items := range e {
		result = append(result, items...)
	}

	return result
}
