package directory

import (
	"fmt"
	"github.com/mrdunski/accumulation-zone/files"
	"github.com/mrdunski/accumulation-zone/index"
	"github.com/mrdunski/accumulation-zone/model"
	"path"
)

type Directory struct {
	Path      string   `arg:"" name:"" help:"Path to synchronize." type:"path"`
	IndexFile string   `help:"File where synchronisation data will be kept." optional:"" default:".changes.log"`
	Excludes  []string `name:"exclude" help:"Exclude some files and directories by name" optional:"" sep:"none"`
}

func (c Directory) GetChanges() ([]model.Change, index.Index, error) {
	var excludes []string
	excludes = append(excludes, c.IndexFile)
	if len(c.Excludes) > 0 {
		excludes = append(excludes, c.Excludes...)
	}
	tree, err := files.NewLoader(c.Path, excludes...).LoadTree()
	if err != nil {
		return nil, index.Index{}, fmt.Errorf("failed to load tree {%s}: %w", c.Path, err)
	}

	idx, err := index.LoadIndexFile(path.Join(c.Path, c.IndexFile))
	if err != nil {
		return nil, index.Index{}, fmt.Errorf("failed to load changes file {%s/%s}: %w", c.Path, c.IndexFile, err)
	}

	return idx.CalculateChanges(model.AsHashedFiles(tree)), idx, nil
}
