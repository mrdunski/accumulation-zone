package main

import (
	"github.com/alecthomas/kong"
	"github.com/mrdunski/accumulation-zone/cmd/changes/commit"
	"github.com/mrdunski/accumulation-zone/cmd/changes/ls"
	"github.com/mrdunski/accumulation-zone/cmd/changes/upload"
	"github.com/mrdunski/accumulation-zone/cmd/inventory"
	"github.com/mrdunski/accumulation-zone/cmd/restore"
	"github.com/mrdunski/accumulation-zone/logger"
)

type CommandInput struct {
	logger.LogConfig

	Recover struct {
		Index restore.IndexCmd `cmd:"" help:"Recovers index file from glacier."`
		Data  restore.DataCmd  `cmd:"" help:"Recovers data from glacier."`
		All   restore.AllCmd   `cmd:"" help:"Recovers index and data from glacier."`
	} `cmd:"" help:"Various backup recovery options." group:"Recover"`

	Changes struct {
		Upload upload.Cmd `cmd:"" help:"Uploads all changes to AWS vault and commits them as processed." group:"Backup"`
		Ls     ls.Cmd     `cmd:"" help:"List changes in the directory."`
		Commit commit.Cmd `cmd:"" help:"DANGER: marks all detected changes as processed and it won't be processed in the future."`
	} `cmd:"" help:"Changes management." group:"Manage Changes"`

	Inventory struct {
		Retrieve inventory.RetrieveCmd `cmd:"" help:"Starts retrieval job for inventory."`
		Print    inventory.PrintCmd    `cmd:"" help:"Awaits latest job completion and prints inventory."`
		Purge    inventory.PurgeCmd    `cmd:"" help:"DANGER: removes all archives from vault. It requires existing retrieval job (in progress or completed)."`
	} `cmd:"" help:"Admin operations on glacier inventory." group:"Manage Glacier Inventory"`
}

func main() {
	var input = CommandInput{}
	kongCtx := kong.Parse(&input)
	input.LogConfig.InitLogger(kongCtx)

	err := kongCtx.Run()
	if err != nil {
		logger.Get().WithError(err).Fatalf("Command failed")
	}
}
