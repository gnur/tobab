package main

import (
	"github.com/gnur/tobab"
)

func allowAdmin(user string, adminGlobs []tobab.Glob) bool {

	for _, g := range adminGlobs {
		if g.Match(user) {
			return true
		}
	}

	return false
}
