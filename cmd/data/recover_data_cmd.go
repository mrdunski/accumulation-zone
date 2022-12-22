package data

import (
	"fmt"
	"github.com/mrdunski/accumulation-zone/directory"
	"github.com/mrdunski/accumulation-zone/glacier"
	"github.com/mrdunski/accumulation-zone/model"
)

type RecoverCmd struct {
	directory.Directory
	glacier.VaultConfig
	glacier.ArchiveRetrievalOptions
}

func (c RecoverCmd) Run() error {
	connection, err := glacier.OpenConnection(c.VaultConfig)
	if err != nil {
		return err
	}

	changes, _, err := c.GetChanges()
	if err != nil {
		return err
	}

	filesToRecover := model.NewIdentifiableHashedFiles(changes.Deletions)

	for _, file := range filesToRecover {
		job, err := connection.FindOrCreateArchiveJob(file, c.ArchiveRetrievalOptions)
		if err != nil {
			return err
		}

		fmt.Printf("* recover job for file %s, status: %s\n", file.Path(), *job.StatusCode)
	}

	for _, file := range filesToRecover {
		content, err := connection.LoadContentFromGlacier(file)
		if err != nil {
			return err
		}

		err = c.SaveFile(content)
		if err != nil {
			return err
		}
	}

	return nil
}
