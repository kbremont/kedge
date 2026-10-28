// Command genicons draws the app icons. Keeping this in Go means the icons
// can be regenerated without any image tooling installed; run it via
// `make icons` after changing the colours below.
package main

import (
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"
	"path/filepath"
)

var (
	bg    = color.RGBA{0x16, 0x19, 0x1f, 0xff}
	plate = color.RGBA{0xf5, 0xa5, 0x24, 0xff}
	bar   = color.RGBA{0xe8, 0xea, 0xed, 0xff}
)

func main() {
	outDir := filepath.Join("internal", "assets", "static")
	for _, size := range []int{180, 192, 512} {
		name := filepath.Join(outDir, "icon-"+itoa(size)+".png")
		if err := write(name, draw(size)); err != nil {
			log.Fatalf("write %s: %v", name, err)
		}
		log.Printf("wrote %s", name)
	}
}

// draw renders a barbell: a horizontal bar with two plates on each side.
func draw(size int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	s := float64(size)

	// Rounded-square background.
	radius := s * 0.22
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			if insideRoundedRect(float64(x)+0.5, float64(y)+0.5, s, radius) {
				img.Set(x, y, bg)
			}
		}
	}

	mid := s / 2
	fill := func(x0, y0, x1, y1 float64, c color.RGBA) {
		for y := int(y0); y < int(y1); y++ {
			for x := int(x0); x < int(x1); x++ {
				if x >= 0 && y >= 0 && x < size && y < size {
					img.Set(x, y, c)
				}
			}
		}
	}

	// The bar itself.
	fill(s*0.16, mid-s*0.028, s*0.84, mid+s*0.028, bar)

	// Inner and outer plates on each side.
	for _, p := range []struct{ x, h float64 }{
		{0.245, 0.30}, // inner
		{0.155, 0.20}, // outer
	} {
		fill(s*p.x, mid-s*p.h, s*(p.x+0.075), mid+s*p.h, plate)
		fill(s*(1-p.x-0.075), mid-s*p.h, s*(1-p.x), mid+s*p.h, plate)
	}
	return img
}

func insideRoundedRect(x, y, size, r float64) bool {
	cx := math.Min(math.Max(x, r), size-r)
	cy := math.Min(math.Max(y, r), size-r)
	dx, dy := x-cx, y-cy
	return dx*dx+dy*dy <= r*r
}

func write(name string, img image.Image) error {
	f, err := os.Create(name)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
