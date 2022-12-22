package index

import (
	"fmt"
	"github.com/mrdunski/accumulation-zone/directory"
	"github.com/mrdunski/accumulation-zone/glacier"
)

type RecoverCmd struct {
	directory.Directory
	glacier.VaultConfig
}

func (c RecoverCmd) Run() error {
	connection, err := glacier.OpenConnection(c.VaultConfig)
	if err != nil {
		return fmt.Errorf("can't open connection: %w", err)
	}

	idx, err := c.CreateIndex()
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	if err := connection.AddInventoryToIndex(idx); err != nil {
		return fmt.Errorf("failed to list files in inventory: %w", err)
	}

	return nil
}
