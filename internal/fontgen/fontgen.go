package fontgen

import (
	"embed"
)

//go:embed all:_libfiles
var libFiles embed.FS

type Config struct {
	DataDir string

	ResultPackage string

	OutDir string

	Tags []string

	DebugPrint func(message string)

	MissingGlyphAction MissingGlyphAction
}

type MissingGlyphAction int

const (
	EmptyMaskOnMissingGlyph MissingGlyphAction = iota
	StubOnMissingGlyph
	PanicOnMissingGlyph
)

func (g MissingGlyphAction) String() string {
	switch g {
	case EmptyMaskOnMissingGlyph:
		return "empty"
	case StubOnMissingGlyph:
		return "stub"
	case PanicOnMissingGlyph:
		return "panic"
	default:
		return "?"
	}
}

type GenerationResult struct {
	Warnings []string
}

func Generate(config Config) (GenerationResult, error) {
	g := newGenerator(config)
	return g.Generate()
}
