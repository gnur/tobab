package main

import (
	"embed"
	_ "embed"
	"fmt"
	"html/template"
)

//go:embed templates
var templateFiles embed.FS

func loadTemplates() (*template.Template, error) {
	tpl := template.New("")
	tpl.Funcs(templateFunctions)

	tpl, err := tpl.ParseFS(templateFiles, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("Unabled to parse templates: %w", err)
	}

	return tpl, nil
}
