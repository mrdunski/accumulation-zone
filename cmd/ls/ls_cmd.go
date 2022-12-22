package ls

import (
	"fmt"
	"github.com/mrdunski/accumulation-zone/directory"
)

type Cmd struct {
	directory.Directory
}

func (c Cmd) Run() error {
	changes, _, err := c.GetChanges()
	if err != nil {
		return fmt.Errorf("failed to calculate changes: %w", err)
	}
	println("Detected changes:")
	for _, change := range changes.Additions {
		fmt.Printf("+ %v\n", change)
	}
	for _, change := range changes.Deletions {
		fmt.Printf("- %v\n", change)
	}

	return nil
}
