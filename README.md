# bitfontier

## Overview

`bitfontier` is a tool to generate bitmap fonts for Go programs.

To be more specific:

* it takes a set of images organized in a particular way for an input,
* generates a Go package that can be used to create [font.Face](https://pkg.go.dev/golang.org/x/image/font#Face) objects

These `font.Face` objects can then be used to render text in your Go programs. One example would be using [Ebitengine](https://pkg.go.dev/golang.org/x/image/font#Face) - the fonts generated by this program are suitable for videogames!

This tool takes a lot of inspiration from [hajimehoshi/bitmapfont](https://github.com/hajimehoshi/bitmapfont).

Features:

* Efficient bitmap generation
* Outputs a ready-to-use Go package
* Produced Go packages ("fonts") can be bundled as Go modules
* The generator has options to fine-tune the produced font behavior
* Uses separate images for glyphs as input (easy to edit)

You can basically create a bitmap font for your Go app by using just this tool and some sprite editor (GIMP, Aseprite, etc).

## Installation

```bash
$ go install github.com/quasilyte/bitfontier/cmd/bitfontier@latest
```

## Usage

First, you need to create a set of images that would form a bitmap font. The font can have multiple base sizes and support multiple languages. We'll start with a single-size English set of images.

The general layout expected by this tool is:

```
$size/
  $tag/
    65.png
    66.png
    ...
  ...
...
```

> The root of this folder structure is called `data-dir`.

* `$size` is a base font size (e.g. `1`, `1.3`)
* `$tag` is an arbitrary tag for the set of glyphs (e.g. `en`, `common`, `symbols`)

The image filename consist of an utf-8 code (in decimal form) and extension (it's advised to use PNGs).

All images inside the size folder should have identical bounds (e.g. `8x16`). Images can use any non-transparent color for the letter mask: this library only checks for alpha channel to build a bitmap.

After you're ready, run the tool:

```bash
# Use --help to learn about the other flags!
./bitfontier --data-dir ./_data --pkgname myfont
```

This will produce a folder called `myfont` containing a Go package. Copy that package to your app's folder and use it as an ordinary package. Or push it as a Go module on GitHub and install it in a proper way.

After installing the generated font package, you can instantiate `font.Face` objects:

```go
func example() {
    // ff is a font.Face, the "1" suffix comes from the
    // data dir, there will be more constructors if your
    // bitmap font defines them.
    ff := myfont.New1()

    // It's possible to create programmatically scaled versions
    // of your base font sizes. It won't be blurry.
    // Only whole scaling factors are available (2, 3, 4, ...)
    ff2 := myfont.Scale(ff, 2)
    ff3 := myfont.Scale(ff, 3)
}
```

Let's assume your font has both `size=1` and `size=1.3` base variants. We can squeeze a wide range of font sizes out of it using `Scale`:

* 1 (base size)
* 1.3 (base size)
* 2 (1*2)
* 2.6 (1.3*2)
* 3 (1*3)
* 3.9 (1.3*3)
* 4
