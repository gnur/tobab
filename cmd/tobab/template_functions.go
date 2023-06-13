package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"time"

	"github.com/gnur/tobab"
)

var templateFunctions = template.FuncMap{
	"percent": func(a, b int) float64 {
		return float64(a) / float64(b) * 100
	},
	"safeHTML": func(s interface{}) template.HTML {
		return template.HTML(fmt.Sprint(s))
	},
	"contains": func(s []string, e string) bool {
		for _, a := range s {
			if a == e {
				return true
			}
		}
		return false
	},
	"canAccess": func(u tobab.User, h string) bool {
		return u.CanAccess(h)
	},
	"crop": func(s string, i int) string {
		if len(s) > i {
			return s[0:(i-2)] + "..."
		}
		return s
	},
	"prettyTime": func(s interface{}) template.HTML {
		t, ok := s.(time.Time)
		if !ok {
			return ""
		}
		if t.IsZero() {
			return template.HTML("never")
		}
		return template.HTML(t.Format("2006-01-02 15:04:05"))
	},
	"json": func(s interface{}) template.HTML {
		json, _ := json.MarshalIndent(s, "", "  ")
		return template.HTML(string(json))
	},
	"relativeTime": func(s interface{}) template.HTML {
		t, ok := s.(time.Time)
		if !ok {
			return ""
		}
		if t.IsZero() {
			return template.HTML("never")
		}
		tense := "ago"
		diff := time.Since(t)
		seconds := int64(diff.Seconds())
		if seconds < 0 {
			tense = "from now"
		}
		var quantifier string

		if seconds < 60 {
			quantifier = "s"
		} else if seconds < 3600 {
			quantifier = "m"
			seconds /= 60
		} else if seconds < 86400 {
			quantifier = "h"
			seconds /= 3600
		} else if seconds < 604800 {
			quantifier = "d"
			seconds /= 86400
		} else if seconds < 31556736 {
			quantifier = "w"
			seconds /= 604800
		} else {
			quantifier = "y"
			seconds /= 31556736
		}

		return template.HTML(fmt.Sprintf("%v%s %s", seconds, quantifier, tense))
	},
}
