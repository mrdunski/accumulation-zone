package commit

import (
	"fmt"
	"github.com/mrdunski/accumulation-zone/volume"
)

type Cmd struct {
	volume.Volume
	IUnderstandConsequencesOfForceCommit bool `required:"" hidden:""`
}

func (c Cmd) Run() error {
	changes, idx, err := c.GetChanges()
	if err != nil {
		return fmt.Errorf("failed to calculate changes: %w", err)
	}
	for _, change := range changes.Deletions {
		err := idx.CommitDelete(change.ChangeId(), change)
		if err != nil {
			return fmt.Errorf("failed to commit change {%v}: %w", change, err)
		}
	}

	for _, change := range changes.Additions {
		err := idx.CommitAdd("", change)
		if err != nil {
			return fmt.Errorf("failed to commit change {%v}: %w", change, err)
		}
	}

	return nil
}
