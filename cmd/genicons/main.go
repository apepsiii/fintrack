package main

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
)

func main() {
	sizes := []int{72, 96, 128, 144, 152, 192, 384, 512}
	os.MkdirAll("../../static/icons", 0755)

	for _, size := range sizes {
		generateIcon(size, "../../static/icons/icon-"+itoa(size)+"x"+itoa(size)+".png")
	}

	// Shortcut icons
	generateIcon(96, "../../static/icons/shortcut-add.png")
	generateIcon(96, "../../static/icons/shortcut-stats.png")

	// Badge
	generateSmallBadge(72, "../../static/icons/badge-72x72.png")
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := [20]byte{}
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[pos:])
}

func generateIcon(size int, path string) {
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	darkGreen := color.RGBA{1, 56, 27, 255}
	lime := color.RGBA{195, 245, 69, 255}

	// Fill background dark green
	draw.Draw(img, img.Bounds(), &image.Uniform{darkGreen}, image.Point{}, draw.Src)

	// Draw rounded rect background (simulate by drawing circle corners)
	radius := float64(size) * 0.22
	cx := float64(size) / 2
	cy := float64(size) / 2

	// Draw lime circle in center
	circleR := float64(size) * 0.30
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x) - cx
			dy := float64(y) - cy
			if dx*dx+dy*dy <= circleR*circleR {
				img.Set(x, y, lime)
			}
		}
	}

	// Draw "F" letter in dark green on lime circle
	drawLetter(img, size, darkGreen)

	// Draw small coin/chart accent bottom right
	accentR := float64(size) * 0.12
	accentX := cx + float64(size)*0.28
	accentY := cy + float64(size)*0.28
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x) - accentX
			dy := float64(y) - accentY
			if dx*dx+dy*dy <= accentR*accentR {
				img.Set(x, y, color.RGBA{255, 255, 255, 180})
			}
		}
	}

	_ = radius

	f, _ := os.Create(path)
	defer f.Close()
	png.Encode(f, img)
}

func drawLetter(img *image.RGBA, size int, col color.RGBA) {
	cx := float64(size) / 2
	cy := float64(size) / 2
	unit := float64(size) * 0.04

	// Draw bold "F" using filled rectangles
	// Vertical bar
	fillRect(img, cx-unit*1.5, cy-unit*4, unit*3, unit*8, col)
	// Top horizontal bar
	fillRect(img, cx-unit*1.5, cy-unit*4, unit*5, unit*2, col)
	// Middle horizontal bar
	fillRect(img, cx-unit*1.5, cy-unit*0.5, unit*4, unit*1.8, col)
}

func fillRect(img *image.RGBA, x, y, w, h float64, col color.RGBA) {
	x0 := int(math.Round(x))
	y0 := int(math.Round(y))
	x1 := int(math.Round(x + w))
	y1 := int(math.Round(y + h))
	bounds := img.Bounds()
	for py := y0; py < y1; py++ {
		for px := x0; px < x1; px++ {
			if px >= bounds.Min.X && px < bounds.Max.X && py >= bounds.Min.Y && py < bounds.Max.Y {
				img.Set(px, py, col)
			}
		}
	}
}

func generateSmallBadge(size int, path string) {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	lime := color.RGBA{195, 245, 69, 255}
	darkGreen := color.RGBA{1, 56, 27, 255}
	cx := float64(size) / 2
	cy := float64(size) / 2
	r := float64(size) / 2

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x) - cx
			dy := float64(y) - cy
			if dx*dx+dy*dy <= r*r {
				img.Set(x, y, lime)
			}
		}
	}
	fillRect(img, cx-float64(size)*0.08, cy-float64(size)*0.3, float64(size)*0.16, float64(size)*0.6, darkGreen)
	fillRect(img, cx-float64(size)*0.3, cy-float64(size)*0.08, float64(size)*0.6, float64(size)*0.16, darkGreen)

	f, _ := os.Create(path)
	defer f.Close()
	png.Encode(f, img)
}
