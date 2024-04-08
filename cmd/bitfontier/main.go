package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/quasilyte/bitfontier"
)

func main() {
	var tagString string
	var onMissing string
	var debug bool
	var generateDocs bool
	var config bitfontier.Config
	flag.StringVar(&config.DataDir, "data-dir", "_data",
		"a path to a folder that contains font images")
	flag.StringVar(&config.OutDir, "out-dir", "",
		"where to put result package files; if empty, pkgname is used")
	flag.StringVar(&config.ResultPackage, "pkgname", "monofont",
		"a result package name")
	flag.StringVar(&tagString, "tags", "",
		"a comma-separated list of tags to include into a result bundle;\nan empty value includes everything")
	flag.StringVar(&onMissing, "on-missing", "emptymask",
		"a missing glyph resolution strategy (`emptymask`, `stub`, or `panic`)")
	flag.BoolVar(&debug, "v", false,
		"whether to enable verbose output")
	flag.BoolVar(&generateDocs, "generate-info", false,
		"whether to generate an additional fontinfo.md file with font stats")
	flag.Parse()

	switch onMissing {
	case "emptymask", "":
		config.MissingGlyphAction = bitfontier.EmptyMaskOnMissingGlyph
	case "stub":
		config.MissingGlyphAction = bitfontier.StubOnMissingGlyph
	case "panic":
		config.MissingGlyphAction = bitfontier.PanicOnMissingGlyph
	default:
		panic(fmt.Sprintf("unsupported on-missing: %q", onMissing))
	}

	for _, t := range strings.Split(tagString, ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			config.Tags = append(config.Tags, t)
		}
	}

	if debug {
		config.DebugPrint = func(message string) {
			fmt.Fprintf(os.Stderr, "info: %s\n", message)
		}
	}

	genResult, err := bitfontier.Generate(config)
	for _, w := range genResult.Warnings {
		fmt.Fprintf(os.Stderr, "warning: %v\n", w)
	}
	if err != nil {
		panic(fmt.Sprintf("error: %v", err))
	}

	if generateDocs {
		makeDoc(config, genResult)
	}
}

func makeDoc(config bitfontier.Config, genResult bitfontier.GenerationResult) {
	var sizes []string
	for _, s := range genResult.FontInfo.Sizes {
		sizes = append(sizes, fmt.Sprintf("`%v`", s))
	}
	for i, r := range genResult.FontInfo.Runes {
		switch r.Value {
		case '#', '\\', '|', '!', '.', '-', '+', '*', '(', ')', '{', '}', '_':
			// Escape markdown special symbols.
			genResult.FontInfo.Runes[i].StringValue = "\\" + r.StringValue
		}
	}
	d := genResult.FontInfo.Date
	dateString := fmt.Sprintf("%d of %s %d", d.Day(), d.Month(), d.Year())

	data := struct {
		FontName    string
		Result      bitfontier.GenerationResult
		SizesString string
		DateString  string
	}{
		FontName:    config.ResultPackage,
		Result:      genResult,
		SizesString: strings.Join(sizes, ", "),
		DateString:  dateString,
	}

	var buf bytes.Buffer
	if err := docTemplate.Execute(&buf, data); err != nil {
		panic(err)
	}
	filename := filepath.Join(config.OutDir, "fontinfo.md")
	if err := os.WriteFile(filename, buf.Bytes(), 0o644); err != nil {
		panic(err)
	}
}

var docTemplate = template.Must(template.New("fontinfo").Parse(`# {{.FontName}} Bitmap Font

## Overview

* Runes: {{len $.Result.FontInfo.Runes}}
* Sizes: {{$.SizesString}}
* Generation date: {{$.DateString}}

## UTF-8 Runes

| Rune | Code | Tag |
|---|---|---|
{{- range $.Result.FontInfo.Runes }}
| {{.StringValue}} | {{.Value}} | {{.Tag}} |
{{- end }}
`))
