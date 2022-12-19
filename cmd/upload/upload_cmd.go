package upload

import (
	"fmt"
	"github.com/mrdunski/accumulation-zone/directory"
	"github.com/mrdunski/accumulation-zone/glacier"
)

type Cmd struct {
	directory.Directory
	glacier.VaultConfig
}

func (c Cmd) Run() error {
	changes, idx, err := c.GetChanges()
	if err != nil {
		return fmt.Errorf("failed to calculate changes: %w", err)
	}
	connection, err := glacier.OpenConnection(c.VaultConfig)
	if err != nil {
		return fmt.Errorf("failed to open connection to backup: %w", err)
	}
	err = connection.Process(idx, changes)
	if err != nil {
		return fmt.Errorf("failed to process changes: %w", err)
	}
	return nil
}
