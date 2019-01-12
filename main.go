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

	"github.com/kr/pretty"
)

type Direction int

const (
	Up Direction = iota
	Right
	Down
	Left
)

type wfcRule struct {
	direction Direction
	tileHash  string
	weight    int
}

type wfcTile struct {
	hash   string
	weight int
	chridx int
	rules  []wfcRule
}

func (wt *wfcTile) CountRule(dir Direction, hash string) {
	for i, _ := range wt.rules {
		if wt.rules[i].direction == dir && wt.rules[i].tileHash == hash {
			wt.rules[i].weight = wt.rules[i].weight + 1
			return
		}
	}
	// If not found on the loop, add new rule.
	wt.rules = append(wt.rules, wfcRule{dir, hash, 1})
}

type tilemap struct {
	tiles  []string
	bounds image.Rectangle
}

func newTilemap(w, h int) tilemap {
	return tilemap{
		tiles:  make([]string, w*h),
		bounds: image.Rect(0, 0, w, h),
	}
}

func (tm tilemap) Set(x, y int, v string) error {
	if x < tm.bounds.Min.X || x >= tm.bounds.Max.X || y < tm.bounds.Min.Y || y >= tm.bounds.Max.Y {
		return fmt.Errorf("out of bounds (%d, %d)", x, y)
	}
	tm.tiles[y*tm.bounds.Dx()+x] = v
	return nil
}

func (tm tilemap) At(x, y int) (string, error) {
	if x < tm.bounds.Min.X || x >= tm.bounds.Max.X || y < tm.bounds.Min.Y || y >= tm.bounds.Max.Y {
		return "", fmt.Errorf("out of bounds (%d, %d)", x, y)
	}
	return tm.tiles[y*tm.bounds.Dx()+x], nil
}

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

	bounds := img.Bounds()
	tiles := make(map[string]*wfcTile)
	tilemap := newTilemap(bounds.Dx() / *sizePtr, bounds.Dy() / *sizePtr)

	for y := bounds.Min.Y; y < bounds.Max.Y; y += *sizePtr {
		for x := bounds.Min.X; x < bounds.Max.X; x += *sizePtr {
			rect := image.Rect(x, y, x+*sizePtr, y+*sizePtr)
			simg := img.(SubImager).SubImage(rect)
			sum := hashImage(simg)
			tile, ok := tiles[sum]
			if !ok {
				tile = &wfcTile{
					hash:   sum,
					chridx: len(tiles),
				}
				tiles[sum] = tile
			}
			tile.weight = tile.weight + 1

			tilemap.Set((x-bounds.Min.X)/(*sizePtr), (y-bounds.Min.Y)/(*sizePtr), sum)
		}
	}

	for y := tilemap.bounds.Min.Y; y < tilemap.bounds.Max.Y; y++ {
		for x := tilemap.bounds.Min.X; x < tilemap.bounds.Max.X; x++ {
			sum, _ := tilemap.At(x, y)
			tile, ok := tiles[sum]
			if !ok {
				log.Printf("unknown tile %s", sum)
			}
			// Check out the rules.
			if rsum, err := tilemap.At(x, y-1); err == nil {
				tiles[sum].CountRule(Up, rsum)
			}
			if rsum, err := tilemap.At(x+1, y); err == nil {
				tiles[sum].CountRule(Right, rsum)
			}
			if rsum, err := tilemap.At(x, y+1); err == nil {
				tiles[sum].CountRule(Down, rsum)
			}
			if rsum, err := tilemap.At(x-1, y); err == nil {
				tiles[sum].CountRule(Left, rsum)
			}

			fmt.Printf("%2d", tile.chridx)
		}
		fmt.Println()
	}

	for _, tile := range tiles {
		pretty.Print(tile)
	}
}
