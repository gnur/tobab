package clirpc

import (
	"github.com/gnur/tobab"
)

type Empty struct{}
type GetHostsOut struct {
	Hosts []tobab.Host
}
type AddHostIn struct {
	Host tobab.Host
}

type DeleteHostIn struct {
	Hostname string
}
