package inventory

import (
	"fmt"
	"github.com/mrdunski/accumulation-zone/glacier"
	"github.com/mrdunski/accumulation-zone/logger"
)

type PrintCmd struct {
	glacier.VaultConfig
}

func (c PrintCmd) Run() error {
	logger.Get().Info("Printing inventory")

	connection, err := glacier.OpenConnection(c.VaultConfig)
	if err != nil {
		return fmt.Errorf("can't open connection: %w", err)
	}

	out, err := connection.InventoryContent()
	if err != nil {
		return fmt.Errorf("failed to print inventory: %w", err)
	}

	fmt.Println(out)
	return nil
}
