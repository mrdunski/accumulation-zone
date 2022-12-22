package inventory

import (
	"fmt"
	"github.com/mrdunski/accumulation-zone/glacier"
)

type PurgeCmd struct {
	glacier.VaultConfig
	IReallyWantToDeleteAllArchivesInVault string `required:"" hidden:"" placeholder:"VAULT_NAME"`
}

func (c PurgeCmd) Run() error {
	if c.VaultName != c.IReallyWantToDeleteAllArchivesInVault {
		return fmt.Errorf("unsafe operation: vault name doesn't match %s != %s", c.VaultName, c.IReallyWantToDeleteAllArchivesInVault)
	}
	connection, err := glacier.OpenConnection(c.VaultConfig)
	if err != nil {
		return fmt.Errorf("can't open connection: %w", err)
	}

	allFiles, err := connection.ListInventoryAllFiles()
	if err != nil {
		return fmt.Errorf("failed to print inventory: %w", err)
	}

	for _, file := range allFiles {
		err := connection.Delete(file.ChangeId())
		if err != nil {
			return fmt.Errorf("can't delete file %s: %w", file.ChangeId(), err)
		}
	}

	return nil
}
