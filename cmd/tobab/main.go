package main

import (
	"errors"
	"fmt"

	"github.com/alecthomas/kong"
	"github.com/gnur/tobab"
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
	_, err := tobab.LoadConf(ctx.Config)
	if err == nil {
		fmt.Println("Config ok")
	}

	return err
}

type VersionCmd struct {
}

func (r *VersionCmd) Run(ctx *Globals) error {
	fmt.Println(version)
	return nil
}

var cli struct {
	Globals

	Run      RunCmd      `cmd:"" help:"start tobab server"`
	Validate ValidateCmd `cmd:"" help:"validate tobab config"`
	Version  VersionCmd  `cmd:"" help:"print tobab version"`
}

func main() {
	ctx := kong.Parse(&cli, kong.UsageOnError())
	err := ctx.Run(&Globals{
		Debug:  cli.Debug,
		Config: cli.Config,
	})
	ctx.FatalIfErrorf(err)
}
