package main

import (
	"errors"
	"fmt"

	"github.com/alecthomas/kong"
)

type Globals struct {
	Debug  bool
	Config string `help:"config location" type:"existingfile" short:"c"`
}

type RunCmd struct {
}

func (r *RunCmd) Run(ctx *Globals) error {
	run(ctx.Config)
	return errors.New("Server exited")
}

type ValidateCmd struct {
}

func (r *ValidateCmd) Run(ctx *Globals) error {
	return validateconf(ctx.Config)
}

type HostCmd struct {
	List HostListCmd `cmd`
}

type HostListCmd struct {
	Test string
}

func (r *HostListCmd) Run(ctx *Globals) error {
	fmt.Println("hoi?")
	return nil
}

type VersionCmd struct {
}

func (r *VersionCmd) Run(ctx *Globals) error {
	fmt.Println(version)
	return nil
}

var cli struct {
	Globals

	Run      RunCmd      `cmd help:"start tobab server"`
	Validate ValidateCmd `cmd help:"validate tobab config"`
	Host     HostCmd     `cmd help:"various host related commands"`
	Version  VersionCmd  `cmd help:"print tobab version"`
}

func main() {
	ctx := kong.Parse(&cli)
	// Call the Run() method of the selected parsed command.
	err := ctx.Run(&Globals{
		Debug:  cli.Debug,
		Config: cli.Config,
	})
	ctx.FatalIfErrorf(err)
}

func validateconf(path string) error {
	return errors.New("Invalid conf")
}
