package main

import (
	"strings"

	"github.com/ryanuber/go-glob"
)

func hasAccess(user, host string, cfg Config) bool {
	h, ok := cfg.Hosts[host]
	if !ok {
		return false
	}

	if h.Public {
		return true
	} else if user == "" {
		return false
	}

	for group, pattern := range cfg.Globs {
		if glob.Glob(pattern, user) {
			if contains(h.AllowedGlobs, group) {
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
