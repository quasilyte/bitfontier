package fontimpl

// This file is needed to keep other files typecheckable.
// It will not be copied along them during the font compilation.
// Instead, the generator would inject the appropriate values on its own.

const (
	onMissing = "empty"
)

type runeAndIndex struct {
	r rune
	i uint32
}
