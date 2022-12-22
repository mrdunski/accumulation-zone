package glacier

import (
	"encoding/json"
	"fmt"
	"github.com/mrdunski/accumulation-zone/model"
	"time"
)

type inventoryArchive struct {
	ArchiveDescription string
	SHA256TreeHash     string
	ArchiveId          string
	CreationDate       time.Time
}

func (a inventoryArchive) ChangeId() string {
	return a.ArchiveId
}

func (a inventoryArchive) Path() string {
	return a.ArchiveDescription
}

func (a inventoryArchive) Hash() string {
	return a.SHA256TreeHash
}

type inventoryArchives map[string]inventoryArchive

func (a inventoryArchives) addNewest(archive inventoryArchive) {
	previous, previousExist := a[archive.Path()]

	if !previousExist || previous.CreationDate.Before(archive.CreationDate) {
		a[archive.Path()] = archive
	}
}

func (a inventoryArchives) asHashFiles() map[string]model.IdentifiableHashedFile {
	result := map[string]model.IdentifiableHashedFile{}

	for k, v := range a {
		result[k] = v
	}

	return result
}

type inventory struct {
	ArchiveList []inventoryArchive
}

func unmarshalInventory(data []byte) (inventory, error) {
	i := inventory{}
	err := json.Unmarshal(data, &i)
	if err != nil {
		return inventory{}, fmt.Errorf("can't unmarshal inventory: %w", err)
	}

	return i, nil
}

func (iv inventory) asIdentifiableHashedFiles() []model.IdentifiableHashedFile {
	result := make([]model.IdentifiableHashedFile, len(iv.ArchiveList))

	for i := range iv.ArchiveList {
		result[i] = iv.ArchiveList[i]
	}

	return result
}

func (iv inventory) newestHashFiles() map[string]model.IdentifiableHashedFile {
	result := inventoryArchives{}

	for _, archive := range iv.ArchiveList {
		result.addNewest(archive)
	}

	return result.asHashFiles()
}
