package model

import (
	"fmt"
)

type ChangeType string

const (
	Added   ChangeType = "added"
	Deleted ChangeType = "deleted"
)

type Change struct {
	HashedFile
	ChangeType ChangeType
}

func (c Change) String() string {
	if c.HashedFile == nil {
		return fmt.Sprintf("{%s: ?}", c.ChangeType)
	}
	return fmt.Sprintf("{%s: {%s %s}}", c.ChangeType, c.Path(), c.Hash())
}
