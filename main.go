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
	"math/rand"
	"os"
	"time"
	// "github.com/kr/pretty"
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

type weightPair struct {
	tileHash string
	weight   int
}

// Container for a superposition tile.
type wfcSspTile struct {
	pairs []weightPair
}

func NewTileFromWfcTiles(tiles []*wfcTile) *wfcSspTile {
	wst := &wfcSspTile{}
	for _, v := range tiles {
		wst.pairs = append(wst.pairs, weightPair{v.hash, v.weight})
	}
	return wst
}

func (wst *wfcSspTile) Entropy() float32 {
	return float32(len(wst.pairs) - 1)
}

func (wst *wfcSspTile) ApplyRules(rules []wfcRule) bool {
	// Build whitelist from previous pairs.
	inc := make(map[string]bool)
	for _, p := range wst.pairs {
		inc[p.tileHash] = true
	}
	// Rebuild prop "pairs" from rules intersected with whitelist.
	wst.pairs = nil
	for _, rule := range rules {
		if _, ok := inc[rule.tileHash]; ok {
			wst.pairs = append(wst.pairs, weightPair{rule.tileHash, rule.weight})
		}
	}

	return len(wst.pairs) > 0
}

// Collapse selects a random pair to collapse considering the pairs.
func (wst *wfcSspTile) Collapse() (string, bool) {
	var slots []*weightPair
	for _, pair := range wst.pairs {
		for i := 0; i < pair.weight; i++ {
			slots = append(slots, &pair)
		}
	}
	// Pick one.
	i := rand.Intn(len(slots))
	p := slots[i]
	wst.pairs = []weightPair{*p}

	return p.tileHash, len(wst.pairs) > 0
}

func (wst *wfcSspTile) IsCollapsed() (bool, bool) {
	return len(wst.pairs) == 1, len(wst.pairs) > 0
}

type wfcSspTilemap struct {
	tiles  []*wfcSspTile
	rules  map[string]([]wfcRule)
	bounds image.Rectangle
}

func NewTilemapFromWfcTiles(tiles []*wfcTile, w, h int) *wfcSspTilemap {
	wtm := &wfcSspTilemap{
		bounds: image.Rect(0, 0, w, h),
	}
	for i := 0; i < w*h; i++ {
		wtm.tiles = append(wtm.tiles, NewTileFromWfcTiles(tiles))
	}
	wtm.rules = make(map[string]([]wfcRule))
	for _, tile := range tiles {
		wtm.rules[tile.hash] = tile.rules
	}
	return wtm
}

func (wtm *wfcSspTilemap) At(x, y int) (*wfcSspTile, error) {
	if x < wtm.bounds.Min.X || x >= wtm.bounds.Max.X || y < wtm.bounds.Min.Y || y >= wtm.bounds.Max.Y {
		return nil, fmt.Errorf("out of bounds (%d, %d)", x, y)
	}
	i := y*wtm.bounds.Dx() + x
	if i < 0 || i >= len(wtm.tiles) {
		return nil, fmt.Errorf("out of bounds (%d, %d)", x, y)
	}
	return wtm.tiles[i], nil
}

func (wtm *wfcSspTilemap) IndexToXY(i int) (int, int) {
	return i % wtm.bounds.Dx(), i / wtm.bounds.Dx()
}

func (wtm *wfcSspTilemap) PickCollapseTile() (int, int, bool) {
	var candidates []int
	bse := float32(1000)
	for i, tile := range wtm.tiles {
		if c, _ := tile.IsCollapsed(); c {
			continue
		}
		if tile.Entropy() < bse {
			candidates = nil
			bse = tile.Entropy()
		}
		if tile.Entropy() == bse {
			candidates = append(candidates, i)
		}
	}
	fmt.Println(candidates)
	if candidates != nil {
		bsi := rand.Intn(len(candidates))
		x, y := wtm.IndexToXY(candidates[bsi])
		return x, y, true
	}
	return 0, 0, false
}

func (wtm *wfcSspTilemap) Collapse(x, y int) error {
	t, err := wtm.At(x, y)
	if err != nil {
		return err
	}
	hash, ok := t.Collapse()
	if !ok {
		return fmt.Errorf("can't collapse tile at %d,%d", x, y)
	}

	// Check out the rules.
	if dt, err := wtm.At(x, y-1); err == nil {
		dt.ApplyRules(wtm.rules[hash])
	}
	if dt, err := wtm.At(x+1, y); err == nil {
		dt.ApplyRules(wtm.rules[hash])
	}
	if dt, err := wtm.At(x, y+1); err == nil {
		dt.ApplyRules(wtm.rules[hash])
	}
	if dt, err := wtm.At(x-1, y); err == nil {
		dt.ApplyRules(wtm.rules[hash])
	}
	return nil
}

func (wtm *wfcSspTilemap) PrintEntropyMap() {
	for i, tile := range wtm.tiles {
		fmt.Printf("%3.0f", tile.Entropy())
		if (i+1)%wtm.bounds.Dx() == 0 {
			fmt.Println()
		}
	}
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
	var values []byte
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			values = append(values, byte(r), byte(g), byte(b), byte(a))
		}
	}

	sum := sha256.Sum256(values)
	return fmt.Sprintf("%x", sum)
}

func main() {
	sizePtr := flag.Int("N", 32, "Size of each tile in input (NxN)")

	rand.Seed(time.Now().UnixNano())
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
	tileImg := make(map[string]image.Image)
	tiles := make(map[string]*wfcTile)
	tilemap := newTilemap(bounds.Dx() / *sizePtr, bounds.Dy() / *sizePtr)

	for y := bounds.Min.Y; y < bounds.Max.Y; y += *sizePtr {
		for x := bounds.Min.X; x < bounds.Max.X; x += *sizePtr {
			rect := image.Rect(x, y, x+*sizePtr, y+*sizePtr)
			simg := img.(SubImager).SubImage(rect)
			sum := hashImage(simg)
			tileImg[sum] = simg
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

	var tilesrr []*wfcTile
	for _, tile := range tiles {
		tilesrr = append(tilesrr, tile)
	}
	wtm := NewTilemapFromWfcTiles(tilesrr, 8, 8)

	for i := 0; i < 32; i++ {
		x, y, ok := wtm.PickCollapseTile()
		if !ok {
			panic("cant pick collapse tile")
		}
		fmt.Println(x, y)
		wtm.Collapse(x, y)
		wtm.PrintEntropyMap()
	}
}
