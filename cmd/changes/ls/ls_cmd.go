package ls

import (
	"fmt"
	"github.com/mrdunski/accumulation-zone/logger"
	"github.com/mrdunski/accumulation-zone/volume"
)

type Cmd struct {
	volume.Volume
}

func (c Cmd) Run() error {
	logger.Get().Info("Listing local changes")
	changes, _, err := c.GetChanges()
	if err != nil {
		return err
	}
	fmt.Println("Detected changes:")
	for _, change := range changes.Additions {
		fmt.Printf("+ %v\n", change)
	}
	for _, change := range changes.Deletions {
		fmt.Printf("- %v\n", change)
	}

	logger.Get().Info("Done")
	return nil
}
