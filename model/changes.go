package model

import (
	"fmt"
)

type FileAdded struct {
	FileWithContent
}

func (f FileAdded) String() string {
	if f.FileWithContent == nil {
		return "{added: ?}"
	}
	return fmt.Sprintf("{added: {%s %s}}", f.Path(), f.Hash())
}

type FileDeleted struct {
	IdentifiableHashedFile
}

func (f FileDeleted) String() string {
	if f.IdentifiableHashedFile == nil {
		return "{deleted: ?}"
	}
	return fmt.Sprintf("{deleted: {%s %s | %s}}", f.Path(), f.Hash(), f.ChangeId())
}

type Changes struct {
	Additions []FileAdded
	Deletions []FileDeleted
}

func (c *Changes) Append(changes Changes) {
	c.Additions = append(c.Additions, changes.Additions...)
	c.Deletions = append(c.Deletions, changes.Deletions...)
}

func (c *Changes) String() string {
	return fmt.Sprintf("{added: %v, deleted: %v}", c.Additions, c.Deletions)
}

func (c *Changes) Len() int {
	return len(c.Additions) + len(c.Deletions)
}
