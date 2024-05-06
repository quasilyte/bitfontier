package fontgen

import (
	"image"
	"math"
)

func measureLetter(img image.Image) (w, h int) {
	bounds := img.Bounds()
	minX := math.MaxInt
	minY := math.MaxInt
	maxX := 0
	maxY := 0

	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			clr := img.At(x, y)
			if _, _, _, a := clr.RGBA(); a == 0 {
				continue
			}
			if x < minX {
				minX = x
			}
			if x > maxX {
				maxX = x
			}
			if y < minY {
				minY = y
			}
			if y > maxY {
				maxY = y
			}
		}
	}

	return maxX - minX + 1, maxY - minY + 1
}
