package main

import (
	"html/template"
	"io/ioutil"
	"os"
	"strings"

	"github.com/markbates/pkger"
)

func loadTemplates() (*template.Template, error) {
	tpl := template.New("")
	tpl.Funcs(templateFunctions)

	err := pkger.Walk("/templates", func(path string, info os.FileInfo, err error) error {
		if strings.HasSuffix(path, ".html") {
			f, err := pkger.Open(path)
			if err != nil {
				return err
			}
			sl, err := ioutil.ReadAll(f)
			if err != nil {
				return err
			}
			_, err = tpl.Parse(string(sl))
			if err != nil {
				return err
			}
		}
		return nil
	})

	return tpl, err
}
