package upload

import (
	"fmt"

	"github.com/mrdunski/accumulation-zone/glacier"
	"github.com/mrdunski/accumulation-zone/logger"
	"github.com/mrdunski/accumulation-zone/volume"
)

type Cmd struct {
	volume.Volume
	glacier.VaultConfig
}

func (c Cmd) Run() error {
	logger.Get().Info("Uploading local changes")

	changes, idx, err := c.GetChanges()
	if err != nil {
		return err
	}
	connection, err := glacier.OpenConnection(c.VaultConfig)
	if err != nil {
		return fmt.Errorf("failed to open connection to backup: %w", err)
	}
	err = connection.Process(idx, changes)
	if err != nil {
		return fmt.Errorf("failed to process changes: %w", err)
	}

	logger.Get().Infof("Done. Added: %d, deleted: %d.", len(changes.Additions), len(changes.Additions))
	return nil
}
