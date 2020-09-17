package clirpc

import (
	"time"

	"github.com/gnur/tobab"
	"github.com/o1egl/paseto/v2"
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

type CreateTokenIn struct {
	Email string
	TTL   time.Duration
}

type CreateTokenOut struct {
	Token string
}

type ValidateTokenIn struct {
	Token string
}

type ValidateTokenOut struct {
	Token paseto.JSONToken
}
