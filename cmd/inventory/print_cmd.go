package inventory

import (
	"fmt"
	"github.com/mrdunski/accumulation-zone/glacier"
)

type PrintCmd struct {
	glacier.VaultConfig
}

func (c PrintCmd) Run() error {
	connection, err := glacier.OpenConnection(c.VaultConfig)
	if err != nil {
		return fmt.Errorf("can't open connection: %w", err)
	}

	err = connection.PrintInventory()
	if err != nil {
		return fmt.Errorf("failed to print inventory: %w", err)
	}

	return nil
}
