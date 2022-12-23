package inventory

import (
	"fmt"
	"github.com/mrdunski/accumulation-zone/glacier"
	"github.com/mrdunski/accumulation-zone/logger"
)

type RetrieveCmd struct {
	glacier.VaultConfig
}

func (c RetrieveCmd) Run() error {
	logger.Get().Info("Running retrieve job on inventory")
	connection, err := glacier.OpenConnection(c.VaultConfig)
	if err != nil {
		return fmt.Errorf("can't open connection: %w", err)
	}

	_, err = connection.CreateInventoryJob()
	if err != nil {
		return fmt.Errorf("failed to create inventory job: %w", err)
	}

	logger.Get().Info("Done")
	return nil
}
