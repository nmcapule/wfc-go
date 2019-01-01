package main

import (
	"bufio"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
)

// Converts a perfectly fine and innocent image to a sad grayscale.
func grayImage(img image.Image) *image.Gray {
	bounds := img.Bounds()
	out := image.NewGray(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			out.Set(x, y, color.GrayModel.Convert(img.At(x, y)))
		}
	}
	return out
}

// Computes the average gray color for a given gray image.
func averageColor(img *image.Gray) color.Gray {
	bounds := img.Bounds()
	var sum uint32
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			gray := img.At(x, y).(color.Gray)
			sum += uint32(gray.Y)
		}
	}
	size := uint32(bounds.Dx() * bounds.Dy())
	if size == 0 {
		return color.Gray{}
	}
	return color.Gray{uint8(sum / size)}
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
	gimg := grayImage(img)

	// Try to print input image to console in 5 levels of gray.
	bounds := gimg.Bounds()
	levels := []string{" ", "░", "▒", "▓", "█"}
	for y := bounds.Min.Y; y < bounds.Max.Y; y += *sizePtr {
		for x := bounds.Min.X; x < bounds.Max.X; x += *sizePtr {
			rect := image.Rect(x, y, x+*sizePtr, y+*sizePtr)
			sub := gimg.SubImage(rect)
			gsub := grayImage(sub)
			avg := averageColor(gsub)
			ch := avg.Y / 51
			fmt.Printf(levels[ch])
		}
		fmt.Println()
	}
}
