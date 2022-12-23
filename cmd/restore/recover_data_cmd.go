package restore

import (
	"github.com/mrdunski/accumulation-zone/glacier"
	"github.com/mrdunski/accumulation-zone/logger"
	"github.com/mrdunski/accumulation-zone/model"
	"github.com/mrdunski/accumulation-zone/volume"
)

type DataCmd struct {
	volume.Volume
	glacier.VaultConfig
	glacier.ArchiveRetrievalOptions
}

func (c DataCmd) Run() error {
	logger.Get().Info("Restoring data from Glacier")

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

		switch *job.StatusCode {
		case "Succeeded":
			logger.Get().Debugf("* recover job for file %s, status: %s\n", file.Path(), *job.StatusCode)
		case "Failed":
			logger.Get().Errorf("* recover job for file %s, status: %s\n", file.Path(), *job.StatusCode)
		default:
			logger.Get().Warnf("* recover job for file %s, status: %s\n", file.Path(), *job.StatusCode)

		}
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

	logger.Get().Info("Done")
	return nil
}
