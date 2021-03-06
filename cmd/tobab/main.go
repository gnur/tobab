package main

import (
	"errors"
	"fmt"
	"log"
	"net/rpc"
	"time"

	"github.com/alecthomas/kong"
	"github.com/gnur/tobab"
	"github.com/gnur/tobab/clirpc"
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

type HostCmd struct {
	List   HostListCmd   `cmd:"" help:"list all hosts"`
	Add    AddHostCmd    `cmd:"" help:"add a new proxy host"`
	Delete DeleteHostCmd `cmd:"" help:"delete a host"`
}

type DeleteHostCmd struct {
	Hostname string `help:"hostname to remove" kong:"required" short:"h"`
}

func (r *DeleteHostCmd) Run(ctx *Globals) error {
	client, err := rpc.DialHTTP("tcp", "localhost:1234")
	if err != nil {
		log.Fatal("dialing:", err)
	}
	in := &clirpc.DeleteHostIn{
		Hostname: r.Hostname,
	}
	var out clirpc.Empty
	err = client.Call("Tobab.DeleteHost", in, &out)
	if err != nil {
		log.Fatal("tobab error:", err)
	}
	fmt.Println("host deleted")
	return nil
}

type AddHostCmd struct {
	Hostname string       `help:"hostname to listen on" kong:"required"`
	Backend  string       `help:"Backend to connect to" kong:"required"`
	Public   bool         `help:"allows all connections"`
	Type     string       `help:"type of proxy" kong:"required"`
	Globs    []tobab.Glob `help:"if host is not public, globs of email addresses to allow access"`
}

func (r *AddHostCmd) Run(ctx *Globals) error {
	client, err := rpc.DialHTTP("tcp", "localhost:1234")
	if err != nil {
		log.Fatal("dialing:", err)
	}
	in := &clirpc.AddHostIn{
		Host: tobab.Host{
			Hostname: r.Hostname,
			Backend:  r.Backend,
			Public:   r.Public,
			Type:     r.Type,
			Globs:    r.Globs,
		},
	}
	var out clirpc.Empty
	err = client.Call("Tobab.AddHost", in, &out)
	if err != nil {
		log.Fatal("tobab error:", err)
	}
	fmt.Println("host added")
	return nil
}

type HostListCmd struct {
}

func (r *HostListCmd) Run(ctx *Globals) error {
	client, err := rpc.DialHTTP("tcp", "localhost:1234")
	if err != nil {
		log.Fatal("dialing:", err)
	}
	in := &clirpc.Empty{}
	var out clirpc.GetHostsOut
	err = client.Call("Tobab.GetHosts", in, &out)
	if err != nil {
		log.Fatal("arith error:", err)
	}
	for _, h := range out.Hosts {
		h.Print()
	}
	return nil
}

type VersionCmd struct {
}

func (r *VersionCmd) Run(ctx *Globals) error {
	fmt.Println(version)
	return nil
}

type TokenCmd struct {
	Create   CreateTokenCmd   `cmd:"" help:"generate a new token"`
	Validate ValidateTokenCmd `cmd:"" help:"Get fields from a token"`
}

type CreateTokenCmd struct {
	Email string `help:"email address to issue token to" kong:"required" short:"e"`
	TTL   string `help:"max age of token" kong:"required" short:"t"`
}

func (r *CreateTokenCmd) Run(ctx *Globals) error {
	ttl, err := time.ParseDuration(r.TTL)
	if err != nil {
		return fmt.Errorf("Invalid duration provided: %w", err)
	}
	client, err := rpc.DialHTTP("tcp", "localhost:1234")
	if err != nil {
		log.Fatal("dialing:", err)
	}
	in := &clirpc.CreateTokenIn{
		Email: r.Email,
		TTL:   ttl,
	}
	var out clirpc.CreateTokenOut
	err = client.Call("Tobab.CreateToken", in, &out)
	if err != nil {
		log.Fatal("tobab error:", err)
	}
	fmt.Println("token created")
	fmt.Println(out.Token)
	return nil
}

type ValidateTokenCmd struct {
	Token string `help:"plain text token" kong:"required" short:"t"`
}

func (r *ValidateTokenCmd) Run(ctx *Globals) error {
	client, err := rpc.DialHTTP("tcp", "localhost:1234")
	if err != nil {
		log.Fatal("dialing:", err)
	}
	in := &clirpc.ValidateTokenIn{
		Token: r.Token,
	}
	var out clirpc.ValidateTokenOut
	err = client.Call("Tobab.ValidateToken", in, &out)
	if err != nil {
		log.Fatal("tobab error:", err)
	}
	t := out.Token
	fmt.Printf(`
Issuer:	       %s
Subject:       %s
Issued at:     %s
Expires at:    %s
`, t.Issuer, t.Subject, t.IssuedAt, t.Expiration)
	return nil
}

var cli struct {
	Globals

	Run      RunCmd      `cmd:"" help:"start tobab server"`
	Validate ValidateCmd `cmd:"" help:"validate tobab config"`
	Host     HostCmd     `cmd:"" help:"various host related commands"`
	Version  VersionCmd  `cmd:"" help:"print tobab version"`
	Token    TokenCmd    `cmd:"" help:"various token related commands"`
}

func main() {
	ctx := kong.Parse(&cli, kong.UsageOnError())
	err := ctx.Run(&Globals{
		Debug:  cli.Debug,
		Config: cli.Config,
	})
	ctx.FatalIfErrorf(err)
}
