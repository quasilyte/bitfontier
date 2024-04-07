package bitfontier

import (
	"github.com/quasilyte/bitfontier/internal/fontgen"
)

// Config contains all exported font generator options.
type Config = fontgen.Config

// MissingGlyphAction affects the code generated for the font package.
//
// Whether a font user tries to render a rune that is not present in the font,
// some resolution strategy should be followed.
// This enumeration provides such strategies.
type MissingGlyphAction = fontgen.MissingGlyphAction

const (
	// EmptyMaskOnMissingGlyph makes [Face.Glyph()] return empty values
	// when undefined rune is requested.
	EmptyMaskOnMissingGlyph = fontgen.EmptyMaskOnMissingGlyph

	// StubOnMissingGlyph makes [Face.Glyph()] return a stub image
	// when undefined rune is requested.
	// A stub image is an opaque rectangle.
	StubOnMissingGlyph = fontgen.StubOnMissingGlyph

	// PanicOnMissingGlyph makes [Face.Glyph()] panic with error
	// when undefined rune is requested.
	// This mode is not recommended for production apps,
	// but it can be useful when testing/debugging.
	PanicOnMissingGlyph = fontgen.PanicOnMissingGlyph
)

type GenerationResult = fontgen.GenerationResult

// Generate creates a bitmap font package following the
// options specified in config.
//
// Its main output will be stored on a local filesystem.
// See [Config.OutDir].
func Generate(config Config) (GenerationResult, error) {
	return fontgen.Generate(config)
}
