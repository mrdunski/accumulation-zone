package restore

import (
	"fmt"
	"github.com/mrdunski/accumulation-zone/glacier"
	"github.com/mrdunski/accumulation-zone/volume"
)

type InventoryJobOptions struct {
	SkipCreateInventoryJob  bool `help:"Skips the creation of new inventory job if missing and fails instead" optional:""`
	ForceCreateInventoryJob bool `help:"Forces creation of new inventory job and ignores existing one" optional:""`
}

type IndexCmd struct {
	volume.Volume
	glacier.VaultConfig
	InventoryJobOptions
}

func (c IndexCmd) Run() error {
	connection, err := glacier.OpenConnection(c.VaultConfig)
	if err != nil {
		return fmt.Errorf("can't open connection: %w", err)
	}

	job, err := connection.FindNewestInventoryJob()
	if err != nil {
		return err
	}

	if job == nil || c.ForceCreateInventoryJob {
		_, err = connection.CreateInventoryJob()
		if err != nil {
			return err
		}
	} else if c.SkipCreateInventoryJob {
		return fmt.Errorf("inventory job is missing")
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
