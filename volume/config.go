package volume

import (
	"fmt"
	"github.com/mrdunski/accumulation-zone/files"
	"github.com/mrdunski/accumulation-zone/index"
	"github.com/mrdunski/accumulation-zone/model"
	"path"
)

type Volume struct {
	Path      string   `arg:"" env:"PATH_TO_BACKUP" help:"Path to synchronize." type:"path" group:"Volume"`
	IndexFile string   `help:"File where synchronisation data will be kept." optional:"" default:".changes.log" group:"Volume"`
	Excludes  []string `name:"exclude" env:"BACKUP_EXCLUDES" help:"Exclude some files and directories by name" optional:"" sep:"none" group:"Volume"`
}

func (c Volume) allExcludes() []string {
	var excludes []string
	excludes = append(excludes, c.IndexFile)
	if len(c.Excludes) > 0 {
		excludes = append(excludes, c.Excludes...)
	}

	return excludes
}

func (c Volume) GetChanges() (model.Changes, index.Index, error) {
	tree, err := files.NewVolume(c.Path, c.allExcludes()...).LoadTree()
	if err != nil {
		return model.Changes{}, index.Index{}, fmt.Errorf("failed to load tree {%s}: %w", c.Path, err)
	}

	idx, err := c.CreateIndex()
	if err != nil {
		return model.Changes{}, idx, err
	}

	return idx.CalculateChanges(tree), idx, nil
}

func (c Volume) CreateIndex() (index.Index, error) {
	idx, err := index.LoadIndexFile(path.Join(c.Path, c.IndexFile))
	if err != nil {
		return index.Index{}, fmt.Errorf("failed to load changes file {%s/%s}: %w", c.Path, c.IndexFile, err)
	}

	return idx, nil
}

func (c Volume) SaveFile(file model.FileWithContent) error {
	return files.NewVolume(c.Path, c.allExcludes()...).Save(file)
}
