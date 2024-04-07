package fontgen

import (
	"bytes"
	"fmt"
	"image"
	_ "image/png"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

type bitmapFont struct {
	Sized []*sizedBitmapFont
	Size1 *sizedBitmapFont
}

type sizedBitmapFont struct {
	Size         float64
	Runes        []bitmapRune
	Dot          *bitmapRune
	GlyphWidth   int
	GlyphHeight  int
	GlyphBitSize int
	Index        int

	// Fields below are initialized during the font processing phase.
	MinRune      rune
	MaxRune      rune
	DotX         int
	DotY         int
	ShortSizeTag string
	SizeTag      string

	// Fields below are initialized during bitmap generation phase.
	BitmapFilename string
}

type bitmapRune struct {
	Value rune
	Img   image.Image
	Tag   string
	Size  float64

	// This field later is used to re-use the duplicated images.
	// For runes that have identical images, this index
	// will point to the rune that should be used as "original".
	ImgIndex int

	DataIndex int
}

func (r bitmapRune) String() string {
	return fmt.Sprintf("%.2f/%s/%v(%q)", r.Size, r.Tag, r.Value, r.Value)
}

type fontParser struct {
	config Config

	result *bitmapFont
}

func (p *fontParser) Parse() (*bitmapFont, error) {
	result := &bitmapFont{}
	p.result = result

	files, err := os.ReadDir(p.config.DataDir)
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		sizeString := f.Name()
		size, err := strconv.ParseFloat(sizeString, 64)
		if err != nil {
			return nil, fmt.Errorf("parsing %q as a font size: %w", sizeString, err)
		}
		path := filepath.Join(p.config.DataDir, sizeString)
		sized, err := p.parseSized(path, size)
		if err != nil {
			return nil, fmt.Errorf("size %.2f: %w", size, err)
		}
		if size == 1.0 {
			result.Size1 = sized
		}
		sized.Index = len(result.Sized)
		result.Sized = append(result.Sized, sized)
	}

	return result, nil
}

func (p *fontParser) parseSized(path string, size float64) (*sizedBitmapFont, error) {
	sized := &sizedBitmapFont{
		Size: size,
	}

	files, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		tagString := f.Name()
		if len(p.config.Tags) > 0 && !slices.Contains(p.config.Tags, tagString) {
			p.config.DebugPrint(fmt.Sprintf("%.2f: skip %q tag", size, tagString))
			continue
		}
		path := filepath.Join(path, tagString)
		runes, err := p.parseRunes(path, tagString, size)
		if err != nil {
			return nil, fmt.Errorf("%q: %w", tagString, err)
		}
		if sized.GlyphWidth == 0 && len(runes) > 0 {
			r := runes[0]
			sized.GlyphWidth = r.Img.Bounds().Dx()
			sized.GlyphHeight = r.Img.Bounds().Dy()
			sized.GlyphBitSize = sized.GlyphWidth * sized.GlyphHeight
		}
		sized.Runes = append(sized.Runes, runes...)
	}

	return sized, nil
}

func (p *fontParser) parseRunes(path, tag string, size float64) ([]bitmapRune, error) {
	files, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	runes := make([]bitmapRune, 0, len(files))
	for _, f := range files {
		runeValueString := strings.TrimSuffix(f.Name(), ".png")
		runeValue, err := strconv.Atoi(runeValueString)
		if err != nil {
			return nil, fmt.Errorf("parse filename as rune value: %w", err)
		}
		imgBytes, err := os.ReadFile(filepath.Join(path, f.Name()))
		if err != nil {
			return nil, err
		}
		img, _, err := image.Decode(bytes.NewReader(imgBytes))
		if err != nil {
			return nil, fmt.Errorf("decode image: %w", err)
		}
		runes = append(runes, bitmapRune{
			Value:    rune(runeValue),
			Img:      img,
			Tag:      tag,
			Size:     size,
			ImgIndex: -1,
		})
	}

	return runes, nil
}
