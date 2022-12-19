package inventory

import (
	"fmt"
	"github.com/mrdunski/accumulation-zone/glacier"
)

type RetrieveCmd struct {
	glacier.VaultConfig
}

func (c RetrieveCmd) Run() error {
	connection, err := glacier.OpenConnection(c.VaultConfig)
	if err != nil {
		return fmt.Errorf("can't open connection: %w", err)
	}

	_, err = connection.CreateInventoryJob()
	if err != nil {
		return fmt.Errorf("failed to create inventory job: %w", err)
	}

	return nil
}
