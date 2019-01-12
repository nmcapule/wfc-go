package main

import (
	"bufio"
	"crypto/sha256"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
)

type SubImager interface {
	SubImage(image.Rectangle) image.Image
}

func equalColor(a, b color.Color) bool {
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()
	return ar == br && ag == bg && ab == bb && aa == ba
}

func equalImage(a, b image.Image) bool {
	ab, bb := a.Bounds(), b.Bounds()
	if ab.Dx() != bb.Dx() || ab.Dy() != bb.Dy() {
		return false
	}

	w := ab.Dx()
	h := ab.Dy()

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			apix := a.At(x+ab.Min.X, y+ab.Min.Y)
			bpix := b.At(x+bb.Min.X, y+bb.Min.Y)
			if !equalColor(apix, bpix) {
				return false
			}
		}
	}
	return true
}

func hashImage(img image.Image) string {
	var values string
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			values = fmt.Sprintf("%s%x%x%x%x", values, r, g, b, a)
		}
	}

	sum := sha256.Sum256([]byte(values))
	return fmt.Sprintf("%x", sum)
}

func main() {
	sizePtr := flag.Int("N", 32, "Size of each tile in input (NxN)")

	flag.Parse()

	path := flag.Arg(0)
	if path == "" {
		log.Fatal("Missing input filename")
	}
	f, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	img, err := png.Decode(bufio.NewReader(f))
	if err != nil {
		log.Fatal(err)
	}

	chtable := make(map[string]int)
	weights := make(map[string]int)

	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y += *sizePtr {
		for x := bounds.Min.X; x < bounds.Max.X; x += *sizePtr {
			rect := image.Rect(x, y, x+*sizePtr, y+*sizePtr)
			simg := img.(SubImager).SubImage(rect)
			sum := hashImage(simg)
			weights[sum] = weights[sum] + 1
			ch, ok := chtable[sum]
			if !ok {
				ch = len(chtable)
				chtable[sum] = ch
			}
			fmt.Printf("%2d ", ch)
		}
		fmt.Println()
	}
}
