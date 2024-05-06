package fontimpl

import (
	"fmt"
	"image"
	"unicode"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

type bitmapFont struct {
	img         *bitmapImage
	glyphWidth  int
	glyphHeight int
	id          int

	lastGlyphRune  rune
	lastGlyphIndex int

	MinRune      rune
	MaxRune      rune
	RuneMapping  []runeAndIndex
	GlyphBitSize uint
	CapHeight    int
	XHeight      int
	DotX         fixed.Int26_6
	DotY         fixed.Int26_6
}

func newBitmapFont(id int, img *bitmapImage, dotX, dotY int) *bitmapFont {
	return &bitmapFont{
		id:          id,
		img:         img,
		glyphWidth:  int(img.width),
		glyphHeight: int(img.height),
		DotX:        fixed.I(dotX),
		DotY:        fixed.I(dotY),
	}
}

func (f *bitmapFont) Close() error {
	return nil
}

func (f *bitmapFont) Glyph(dot fixed.Point26_6, r rune) (dr image.Rectangle, mask image.Image, maskp image.Point, advance fixed.Int26_6, ok bool) {
	// maskp remains a zero value as we don't need it.
	dr, mask, advance, ok = f.glyph(dot, r)
	return dr, mask, maskp, advance, ok
}

func (f *bitmapFont) glyph(dot fixed.Point26_6, r rune) (dr image.Rectangle, mask *bitmapImage, advance fixed.Int26_6, ok bool) {
	// First do a quick range check.
	if r > f.MaxRune || r < f.MinRune {
		return dr, mask, advance, false
	}

	// Map rune to its index inside the associated data.
	index, ok := f.getRuneDataIndex(r)
	if !ok {
		// onMissing is a const, so the compiler should eliminate
		// all checks here and keep the right case only.
		switch onMissing {
		case "emptymask":
			return dr, mask, advance, false
		case "stub":
			index = getStubImageIndex(f.id)
		case "panic":
			panic(fmt.Sprintf("requesting an undefined rune %v (%q)", r, r))
		}
	}

	rw := f.glyphWidth
	rh := f.glyphHeight
	dx := (dot.X - f.DotX).Floor()
	dy := (dot.Y - f.DotY).Floor()
	dr = image.Rect(dx, dy, dx+rw, dy+rh)

	offset := index * f.GlyphBitSize
	mask = f.img.WithOffset(offset)
	advance = fixed.I(rw)
	return dr, mask, advance, true
}

func (f *bitmapFont) GlyphAdvance(r rune) (advance fixed.Int26_6, ok bool) {
	if r > f.MaxRune || r < f.MinRune {
		return 0, false
	}
	return fixed.I(f.glyphWidth), true
}

func (f *bitmapFont) GlyphBounds(r rune) (bounds fixed.Rectangle26_6, advance fixed.Int26_6, ok bool) {
	if r > f.MaxRune || r < f.MinRune {
		return bounds, advance, false
	}
	bounds = fixed.Rectangle26_6{
		Min: fixed.Point26_6{X: -f.DotX, Y: -f.DotY},
		Max: fixed.Point26_6{
			X: -f.DotX + fixed.I(f.glyphWidth),
			Y: -f.DotY + fixed.I(f.glyphHeight),
		},
	}
	advance = fixed.I(f.glyphWidth)
	return bounds, advance, true
}

func (f *bitmapFont) Kern(r0, r1 rune) fixed.Int26_6 {
	if unicode.Is(unicode.Mn, r1) {
		return -fixed.I(f.glyphWidth)
	}
	return 0

}

func (f *bitmapFont) Metrics() font.Metrics {
	// TODO: make line height (Height) configurable?
	return font.Metrics{
		Height:    fixed.I(f.glyphHeight),
		XHeight:   fixed.I(f.XHeight),
		CapHeight: fixed.I(f.CapHeight),
		Ascent:    f.DotY,
		Descent:   fixed.I(f.glyphHeight) - f.DotY,
	}
}

func (f *bitmapFont) getRuneDataIndex(r rune) (uint, bool) {
	slice := f.RuneMapping

	// A heuristic search that depends on the previous binary search result.
	// Since most of the time we're looking for a rune from the
	// same language, these runes might be very close to each other.
	// If there are no gaps in between these two runes, we
	// can find the target rune data index with a simple calculation.
	//
	// If there is a gap, this strategy will fail and we'll proceed
	// to the binary search below.
	//
	// When it works, it gives 20-25% Glyph() speedup
	// and turns the lookup into O(1).
	{
		delta := int(r) - int(f.lastGlyphRune)
		index := uint(f.lastGlyphIndex + delta)
		if index < uint(len(slice)) {
			if rune(slice[index].r) == r {
				return uint(slice[index].i), true
			}
		}
	}

	// This is an inlined sort.Search specialized for our slice.

	i, j := 0, len(slice)
	for i < j {
		h := int(uint(i+j) >> 1)
		v := slice[h]
		// The explicit rune conversion is necessary here.
		// v.r could be uint16 when small rune optimization is in order.
		if rune(v.r) < r {
			i = h + 1
		} else {
			j = h
		}
	}

	if i < len(slice) && rune(slice[i].r) == r {
		// Save the results for the heuristic search above.
		f.lastGlyphRune = r
		f.lastGlyphIndex = i
		return uint(slice[i].i), true
	}

	return 0, false
}
