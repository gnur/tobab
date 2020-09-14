package main

import (
	"strings"

	"github.com/gnur/tobab"
	matcher "github.com/ryanuber/go-glob"
)

func hasAccess(user, host string, cfg tobab.Config) bool {
	h, ok := cfg.Hosts[host]
	if !ok {
		return false
	}

	if h.Public {
		return true
	} else if user == "" {
		return false
	}

	for _, g := range cfg.Globs {
		if matcher.Glob(g.Matcher, user) {
			if contains(h.AllowedGlobs, g.Name) {
				return true
			}
		}
	}

	return false
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if strings.EqualFold(a, e) {
			return true
		}
	}
	return false
}

func allowAdmin(user string, cfg tobab.Config) bool {

	for _, globName := range cfg.AdminGlobs {
		for _, g := range cfg.Globs {
			if g.Name != globName {
				continue
			}
			if matcher.Glob(g.Matcher, user) {
				return true
			}
		}
	}

	return false
}
