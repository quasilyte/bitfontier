package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/quasilyte/bitfontier"
)

func main() {
	var tagString string
	var onMissing string
	var debug bool
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
}
