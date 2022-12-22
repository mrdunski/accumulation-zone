package main

import (
	"github.com/alecthomas/kong"
	"github.com/mrdunski/accumulation-zone/cmd/commit"
	"github.com/mrdunski/accumulation-zone/cmd/index"
	"github.com/mrdunski/accumulation-zone/cmd/inventory"
	"github.com/mrdunski/accumulation-zone/cmd/ls"
	"github.com/mrdunski/accumulation-zone/cmd/upload"
)

type CommandInput struct {
	Changes struct {
		Ls     ls.Cmd     `cmd:"" help:"List changes in the directory."`
		Commit commit.Cmd `cmd:"" help:"DANGER: marks all detected changes as processed and it won't be processed in the future."`
		Upload upload.Cmd `cmd:"" help:"Uploads all changes to AWS vault and commits them as processed." group:"Backup"`
	} `cmd:"" help:"Changes management." group:"Manage Changes"`

	Inventory struct {
		Retrieve inventory.RetrieveCmd `cmd:"" help:"Starts retrieval job for inventory."`
		Print    inventory.PrintCmd    `cmd:"" help:"Awaits latest job completion and prints inventory."`
		Purge    inventory.PurgeCmd    `cmd:"" help:"DANGER: removes all archives from vault. It requires existing retrieval job (in progress or completed)."`
	} `cmd:"" help:"Admin operations on glacier inventory." group:"Manage Glacier Inventory"`

	Recover struct {
		Index index.RecoverCmd `cmd:"" help:"Recovers index file from glacier"`
	} `cmd:"" help:"Various backup recovery options." group:"Recover"`
}

func main() {
	var input = CommandInput{}
	kongCtx := kong.Parse(&input)

	err := kongCtx.Run()
	kongCtx.FatalIfErrorf(err)
}
