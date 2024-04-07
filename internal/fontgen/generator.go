package fontgen

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"go/format"
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type generator struct {
	config Config

	font     *bitmapFont
	warnings []string
}

func newGenerator(config Config) *generator {
	return &generator{config: config}
}

func (g *generator) Generate() (GenerationResult, error) {
	type step struct {
		name string
		fn   func() error
	}

	var result GenerationResult

	steps := []step{
		{"validate config", g.validateConfig},
		{"prepare outdir", g.prepareOutdir},
		{"parse font", g.parseFont},
		{"validate font", g.validateFont},
		{"process font", g.processFont},
		{"create bitmap", g.createBitmap},
		{"create package", g.createPackage},
		{"copy lib files", g.copyLibFiles},
	}
	for _, s := range steps {
		if err := s.fn(); err != nil {
			return result, fmt.Errorf("%s: %w", s.name, err)
		}
	}

	result.Warnings = g.warnings
	return result, nil
}

func (g *generator) validateConfig() error {
	if g.config.ResultPackage == "" {
		return fmt.Errorf("ResultPackage can't be empty")
	}
	if g.config.DataDir == "" {
		return fmt.Errorf("DataDir can't be empty")
	}

	if g.config.DebugPrint == nil {
		g.config.DebugPrint = func(message string) {}
	}
	if g.config.OutDir == "" {
		g.config.OutDir = g.config.ResultPackage
	}

	return nil
}

func (g *generator) prepareOutdir() error {
	if err := os.RemoveAll(g.config.OutDir); err != nil {
		return err
	}

	if err := os.MkdirAll(g.config.OutDir, os.ModePerm); err != nil {
		return err
	}

	return nil
}

func (g *generator) parseFont() error {
	p := fontParser{config: g.config}
	f, err := p.Parse()
	g.font = f
	return err
}

func (g *generator) validateFont() error {
	if g.font.Size1 == nil {
		return fmt.Errorf("can't find size=1 images")
	}

	for _, sf := range g.font.Sized {
		set := map[rune]string{}
		for i, r := range sf.Runes {
			if r.Value == '.' {
				sf.Dot = &sf.Runes[i]
			}
			b := r.Img.Bounds()
			if b.Dx() != sf.GlyphWidth || b.Dy() != sf.GlyphHeight {
				return fmt.Errorf("%s: found %dx%d image size, expected %dx%d", r, b.Dx(), b.Dy(), sf.GlyphWidth, sf.GlyphHeight)
			}
			if prevTag, ok := set[r.Value]; ok {
				return fmt.Errorf("%s: duplicated rune, previously defined at %q", r, prevTag)
			}
			set[r.Value] = r.Tag
		}

		if sf.Dot == nil {
			return fmt.Errorf("%.2f: missing a period `.` symbol (charcode=46)", sf.Size)
		}
	}

	// Build a base set of glyphs based on the size1 runes.
	size1runes := map[rune]string{}
	for _, r := range g.font.Size1.Runes {
		size1runes[r.Value] = r.Tag
	}

	for _, sf := range g.font.Sized {
		if sf == g.font.Size1 {
			continue
		}

		placeholder := image.NewNRGBA(image.Rectangle{
			Max: image.Pt(sf.GlyphWidth, sf.GlyphHeight),
		})
		for y := 0; y < sf.GlyphHeight; y++ {
			for x := 0; x < sf.GlyphWidth; x++ {
				if x == 0 || x == sf.GlyphWidth-1 {
					continue
				}
				if y == 0 || y == sf.GlyphHeight-1 {
					continue
				}
				placeholder.Set(x, y, color.NRGBA{A: 0xff})
			}
		}

		runes := map[rune]struct{}{}
		for _, r := range sf.Runes {
			runes[r.Value] = struct{}{}
			// Having an extra rune is an error.
			if _, ok := size1runes[r.Value]; !ok {
				return fmt.Errorf("%s: this rune is missing in size=1 variant", r)
			}
		}
		for r, tag := range size1runes {
			if _, ok := runes[r]; !ok {
				br := bitmapRune{
					Value: r,
					Img:   placeholder,
					Tag:   tag,
					Size:  sf.Size,
				}
				sf.Runes = append(sf.Runes, br)
				g.warnings = append(g.warnings, fmt.Sprintf("%s: using a placeholder image", br))
			}
		}
	}

	return nil
}

func (g *generator) processFont() error {
	for _, sf := range g.font.Sized {
		sort.Slice(sf.Runes, func(i, j int) bool {
			return sf.Runes[i].Value < sf.Runes[j].Value
		})
	}

	for _, sf := range g.font.Sized {
		sf.SizeTag = strings.Replace(fmt.Sprintf("%.2f", sf.Size), ".", "_", 1)
		sf.ShortSizeTag = strings.Replace(fmt.Sprintf("%v", sf.Size), ".", "_", 1)
	}

	for _, sf := range g.font.Sized {
		sf.MinRune = rune(math.MaxInt32)
		sf.MaxRune = rune(math.MinInt32)
		for _, r := range sf.Runes {
			sf.MinRune = min(sf.MinRune, r.Value)
			sf.MaxRune = max(sf.MaxRune, r.Value)
		}
	}

	for _, sf := range g.font.Sized {
		minX := math.MaxInt
		for _, r := range sf.Runes {
			w := r.Img.Bounds().Dx()
			h := r.Img.Bounds().Dy()
			for y := 0; y < h; y++ {
				for x := 0; x < w; x++ {
					clr := r.Img.At(x, y)
					if _, _, _, a := clr.RGBA(); a != 0 {
						minX = min(minX, x)
					}
				}
			}
		}

		dotY := 0
	FindDotY:
		for y := sf.GlyphHeight - 1; y >= 0; y-- {
			for x := 0; x < sf.GlyphWidth; x++ {
				clr := sf.Dot.Img.At(x, y)
				if _, _, _, a := clr.RGBA(); a != 0 {
					dotY = y
					break FindDotY
				}
			}
		}

		sf.DotX = minX
		sf.DotY = dotY
	}

	imgKey := func(img image.Image) string {
		var buf strings.Builder
		bounds := img.Bounds()
		buf.Grow(bounds.Dx()*bounds.Dy() + bounds.Dy())
		for y := 0; y < bounds.Dy(); y++ {
			for x := 0; x < bounds.Dx(); x++ {
				_, _, _, a := img.At(x, y).RGBA()
				if a == 0 {
					buf.WriteByte('0')
				} else {
					buf.WriteByte('1')
				}
			}
		}
		return buf.String()
	}
	for _, sf := range g.font.Sized {
		imgSet := make(map[string]int, len(sf.Runes))
		for i, r := range sf.Runes {
			k := imgKey(r.Img)
			index, ok := imgSet[k]
			if ok {
				g.config.DebugPrint(fmt.Sprintf("%s: re-use image from %s", r, sf.Runes[index]))
				sf.Runes[i].ImgIndex = index
				continue
			}
			imgSet[k] = i
		}
	}

	return nil
}

func (g *generator) createBitmap() error {
	outDir := g.config.OutDir

	for _, sf := range g.font.Sized {
		sf.BitmapFilename = sf.SizeTag + ".data.gz"

		numUniqueImages := 0
		for _, r := range sf.Runes {
			if r.ImgIndex != -1 {
				continue
			}
			numUniqueImages++
		}
		g.config.DebugPrint(fmt.Sprintf("%.2f: %d/%d images are unique", sf.Size, numUniqueImages, len(sf.Runes)))

		numBits := numUniqueImages * int(sf.GlyphBitSize)
		data := make([]byte, (numBits/8)+1)
		g.config.DebugPrint(fmt.Sprintf("%.2f: allocated %d bytes", sf.Size, len(data)))

		bitIndex := 0
		dataIndex := 0
		for i, r := range sf.Runes {
			if r.ImgIndex != -1 {
				// Duplicates do not advance bitIndex/dataIndex.
				continue
			}
			for y := 0; y < sf.GlyphHeight; y++ {
				for x := 0; x < sf.GlyphWidth; x++ {
					clr := r.Img.At(x, y)
					v := 0
					if _, _, _, a := clr.RGBA(); a != 0 {
						v = 1
					}
					bytePos := bitIndex / 8
					byteShift := bitIndex % 8
					data[bytePos] |= byte(v << byteShift)
					bitIndex++
				}
			}
			sf.Runes[i].DataIndex = dataIndex
			dataIndex++
		}
		for i, r := range sf.Runes {
			if r.ImgIndex == -1 {
				continue
			}
			sf.Runes[i].DataIndex = sf.Runes[r.ImgIndex].DataIndex
		}
		g.config.DebugPrint(fmt.Sprintf("%.2f: filled %d/%d bytes", sf.Size, bitIndex/8, len(data)))

		var compressed bytes.Buffer
		gzw := gzip.NewWriter(&compressed)
		if _, err := gzw.Write(data); err != nil {
			return fmt.Errorf("%.2f: %w", sf.Size, err)
		}
		if err := gzw.Flush(); err != nil {
			return fmt.Errorf("%.2f: %w", sf.Size, err)
		}
		if err := gzw.Close(); err != nil {
			return fmt.Errorf("%.2f: %w", sf.Size, err)
		}

		if err := os.WriteFile(filepath.Join(outDir, sf.BitmapFilename), compressed.Bytes(), 0o644); err != nil {
			return fmt.Errorf("%.2f: %w", sf.Size, err)
		}
	}

	return nil
}

func (g *generator) createPackage() error {
	data := &templateData{
		PkgName:   g.config.ResultPackage,
		Fonts:     g.font.Sized,
		OnMissing: g.config.MissingGlyphAction.String(),
	}

	// The rune mapping will use a binary search.
	// Glyph() is the only method that requires this mapping
	// and its results are usually cached.
	// Therefore, we can trade some of the execution speed
	// for a smaller mapping data structure.
	// Using a LUT or other approach may increase the mapping
	// speed at the cost of extra memory/code size.
	//
	// The exact mapping method should not concern the users.
	for _, sf := range g.font.Sized {
		mapping := []runeAndIndex{}
		for _, r := range sf.Runes {
			mapping = append(mapping, runeAndIndex{
				Rune:  r.Value,
				Index: r.DataIndex,
			})
		}
		data.RuneMappings = append(data.RuneMappings, &runeMapping{
			Slice:      mapping,
			SizeApprox: len(mapping) * 8,
		})
	}

	var buf bytes.Buffer
	if err := fontfaceTemplate.Execute(&buf, data); err != nil {
		return err
	}

	pretty, err := format.Source(buf.Bytes())
	if err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(g.config.OutDir, "fontface.go"), pretty, 0o644); err != nil {
		return err
	}

	return nil
}

func (g *generator) copyLibFiles() error {
	fontimplDir := "_libfiles/fontimpl"
	files, err := libFiles.ReadDir(fontimplDir)
	if err != nil {
		return err
	}
	for _, f := range files {
		if f.Name() == "stubs.go" {
			continue
		}
		code, err := libFiles.ReadFile(filepath.Join(fontimplDir, f.Name()))
		if err != nil {
			return err
		}
		code = bytes.TrimPrefix(code, []byte("package fontimpl"))
		code = append([]byte("package "+g.config.ResultPackage), code...)
		if err := os.WriteFile(filepath.Join(g.config.OutDir, f.Name()), code, 0o644); err != nil {
			return err
		}
	}

	return nil
}
