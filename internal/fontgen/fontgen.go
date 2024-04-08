package fontgen

import (
	"embed"
	"time"
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
		return "emptymask"
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

	FontInfo FontInfo
}

type FontInfo struct {
	Runes []RuneInfo

	Sizes []float64

	Date time.Time
}

type RuneInfo struct {
	Value       rune
	StringValue string
	Tag         string
}

func Generate(config Config) (GenerationResult, error) {
	g := newGenerator(config)
	return g.Generate()
}
