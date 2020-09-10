package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"reflect"
	"time"
)

var templateFunctions = template.FuncMap{
	"percent": func(a, b int) float64 {
		return float64(a) / float64(b) * 100
	},
	"safeHTML": func(s interface{}) template.HTML {
		return template.HTML(fmt.Sprint(s))
	},
	"crop": func(s string, i int) string {
		if len(s) > i {
			return s[0:(i-2)] + "..."
		}
		return s
	},
	"add": func(b, a interface{}) (interface{}, error) {
		av := reflect.ValueOf(a)
		bv := reflect.ValueOf(b)

		switch av.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			switch bv.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				return av.Int() + bv.Int(), nil
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				return av.Int() + int64(bv.Uint()), nil
			case reflect.Float32, reflect.Float64:
				return float64(av.Int()) + bv.Float(), nil
			default:
				return nil, fmt.Errorf("add: unknown type for %q (%T)", bv, b)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			switch bv.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				return int64(av.Uint()) + bv.Int(), nil
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				return av.Uint() + bv.Uint(), nil
			case reflect.Float32, reflect.Float64:
				return float64(av.Uint()) + bv.Float(), nil
			default:
				return nil, fmt.Errorf("add: unknown type for %q (%T)", bv, b)
			}
		case reflect.Float32, reflect.Float64:
			switch bv.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				return av.Float() + float64(bv.Int()), nil
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				return av.Float() + float64(bv.Uint()), nil
			case reflect.Float32, reflect.Float64:
				return av.Float() + bv.Float(), nil
			default:
				return nil, fmt.Errorf("add: unknown type for %q (%T)", bv, b)
			}
		default:
			return nil, fmt.Errorf("add: unknown type for %q (%T)", av, a)
		}
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
