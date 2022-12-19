package commit

import (
	"fmt"
	"github.com/mrdunski/accumulation-zone/directory"
)

type Cmd struct {
	directory.Directory
	IUnderstandConsequencesOfForceCommit bool `required:"" hidden:"" aliases:"force"`
}

func (c Cmd) Run() error {
	changes, idx, err := c.GetChanges()
	if err != nil {
		return fmt.Errorf("failed to calculate changes: %w", err)
	}
	for _, change := range changes {
		err := idx.CommitChange("", change) // TODO: do not cover err
		if err != nil {
			return fmt.Errorf("failed to commit change {%v}: %w", change, err)
		}
	}

	return nil
}
