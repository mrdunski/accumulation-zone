package main

import (
	"fmt"
	"github.com/alecthomas/kong"
	"github.com/mrdunski/accumulation-zone/files"
	"github.com/mrdunski/accumulation-zone/index"
	"github.com/mrdunski/accumulation-zone/model"
	"path"
)

type ChangesCmdConfig struct {
	Path      string   `arg:"" name:"" help:"Path to synchronize." type:"path"`
	IndexFile string   `name:"" help:"File where synchronisation data will be kept." optional:"" default:".index.log"`
	Excludes  []string `name:"exclude" help:"Exclude some files and directories by name" optional:"" sep:"none"`
}

var CLI struct {
	Changes struct {
		Ls     ChangesCmdConfig `cmd:"" help:"list changes in the directory"`
		Commit ChangesCmdConfig `cmd:"" help:"marks all detected changes as processed"`
	} `cmd:"" help:"changes management"`
}

func main() {
	ctx := kong.Parse(&CLI)

	switch ctx.Command() {
	case "changes ls <path>":
		changes, _ := getChanges(CLI.Changes.Ls)
		println("Detected changes:")
		for _, change := range changes {
			fmt.Printf("* %v\n", change)
		}
	case "changes commit <path>":
		changes, idx := getChanges(CLI.Changes.Commit)
		for _, change := range changes {
			err := idx.CommitChange("", change)
			if err != nil {
				panic(err)
			}
		}
		println("Done")
	default:
		panic(ctx.Command())
	}
}

func getChanges(cfg ChangesCmdConfig) ([]model.Change, index.Index) {
	var excludes []string
	excludes = append(excludes, cfg.IndexFile)
	if len(cfg.Excludes) > 0 {
		excludes = append(excludes, cfg.Excludes...)
	}
	tree, err := files.NewLoader(cfg.Path, excludes...).LoadTree()
	if err != nil {
		panic(err)
	}

	idx, err := index.LoadIndexFile(path.Join(cfg.Path, cfg.IndexFile))
	if err != nil {
		panic(err)
	}

	return idx.CalculateChanges(model.AsHashedFiles(tree)), idx
}
