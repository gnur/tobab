package tobab

import (
	matcher "github.com/ryanuber/go-glob"
)

type Config struct {
	Hostname     string
	CookieScope  string
	Secret       string
	Salt         string
	CertDir      string
	Email        string
	Staging      bool
	GoogleKey    string
	GoogleSecret string
	Loglevel     string
	DatabasePath string
	AdminGlobs   []Glob
}

type Host struct {
	Hostname string `storm:"id"`
	Backend  string
	Type     string
	Public   bool
	Globs    []Glob
}

type Glob string

func (g Glob) Match(s string) bool {
	return matcher.Glob(string(g), s)
}

func (h Host) HasAccess(user string) bool {

	if h.Public {
		return true
	} else if user == "" {
		return false
	}

	for _, g := range h.Globs {
		if g.Match(user) {
			return true
		}
	}

	return false
}
