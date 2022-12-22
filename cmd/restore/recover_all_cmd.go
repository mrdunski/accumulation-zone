package restore

import (
	"github.com/mrdunski/accumulation-zone/glacier"
	"github.com/mrdunski/accumulation-zone/volume"
)

type AllCmd struct {
	volume.Volume
	glacier.VaultConfig
	glacier.ArchiveRetrievalOptions
	InventoryJobOptions
}

func (c AllCmd) Run() error {
	if err := c.idx().Run(); err != nil {
		return err
	}

	return c.data().Run()
}

func (c AllCmd) idx() IndexCmd {
	return IndexCmd{
		Volume:              c.Volume,
		VaultConfig:         c.VaultConfig,
		InventoryJobOptions: c.InventoryJobOptions,
	}
}

func (c AllCmd) data() DataCmd {
	return DataCmd{
		Volume:                  c.Volume,
		VaultConfig:             c.VaultConfig,
		ArchiveRetrievalOptions: c.ArchiveRetrievalOptions,
	}
}
