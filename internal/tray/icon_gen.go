// Package tray (icon_gen.go) generates tray icons at runtime based on
// the engine state. Each state gets a distinct background color with a
// white glyph so the user can see the proxy status at a glance without
// opening the menu:
//
//   - Running: green with a checkmark
//   - Starting: yellow with a clock
//   - Stopping: orange with a clock
//   - Stopped: gray with a pause bar
//   - Error: red with an exclamation mark
package tray

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"

	"fyne.io/fyne/v2"

	"subrelay/internal/core"
	"subrelay/internal/state"
)

// iconSize is the width and height of the generated tray icon in pixels.
const iconSize = 32

// iconForSnapshot returns a Fyne resource whose color and glyph reflect
// the current engine state and error status.
//
// Args:
//   - snap: the current state snapshot.
//
// Returns:
//   - A fyne.Resource wrapping a PNG encoded at iconSize x iconSize.
func iconForSnapshot(snap state.Snapshot) fyne.Resource {
	bg, glyph := paletteForSnapshot(snap)
	img := drawIcon(bg, glyph)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		// Encoding a 32x32 RGBA never fails in practice; fall back
		// to the static embedded icon on the impossible failure.
		return iconResource()
	}
	return fyne.NewStaticResource("tray-icon-dynamic.png", buf.Bytes())
}

// paletteForSnapshot selects a background color and glyph for the icon
// based on the engine state and error condition.
func paletteForSnapshot(snap state.Snapshot) (bg color.RGBA, glyph glyphType) {
	if snap.LastError != "" {
		return color.RGBA{200, 40, 40, 255}, glyphError
	}
	switch snap.EngineState {
	case core.StateRunning:
		return color.RGBA{40, 160, 70, 255}, glyphCheck
	case core.StateStarting:
		return color.RGBA{220, 180, 30, 255}, glyphClock
	case core.StateStopping:
		return color.RGBA{220, 130, 30, 255}, glyphClock
	default:
		return color.RGBA{120, 120, 120, 255}, glyphPause
	}
}

// glyphType identifies which overlay symbol to draw on the icon.
type glyphType int

const (
	glyphNone glyphType = iota
	glyphCheck
	glyphClock
	glyphPause
	glyphError
)

// drawIcon renders a filled background with a white glyph centered on
// it.
//
// Args:
//   - bg: the background color.
//   - glyph: the overlay symbol to draw.
//
// Returns:
//   - An image.RGBA of size iconSize x iconSize.
func drawIcon(bg color.RGBA, glyph glyphType) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, iconSize, iconSize))

	// Fill background.
	draw.Draw(img, img.Bounds(), &image.Uniform{bg}, image.Point{}, draw.Src)

	// Draw a subtle darker border.
	border := darker(bg, 40)
	for x := 0; x < iconSize; x++ {
		img.Set(x, 0, border)
		img.Set(x, iconSize-1, border)
	}
	for y := 0; y < iconSize; y++ {
		img.Set(0, y, border)
		img.Set(iconSize-1, y, border)
	}

	white := color.RGBA{255, 255, 255, 255}
	switch glyph {
	case glyphCheck:
		drawCheck(img, white)
	case glyphClock:
		drawClock(img, white)
	case glyphPause:
		drawPause(img, white)
	case glyphError:
		drawError(img, white)
	}

	return img
}

// drawCheck draws a white checkmark centered in the icon.
func drawCheck(img *image.RGBA, c color.Color) {
	pts := []image.Point{{8, 16}, {13, 21}, {23, 10}}
	drawPolyline(img, pts, c, 2)
}

// drawClock draws a white clock face (circle with hands) centered in
// the icon.
func drawClock(img *image.RGBA, c color.Color) {
	cx, cy := 16, 16
	r := 9
	// Circle outline.
	for t := 0; t < 360; t += 3 {
		drawPoint(img, cx+int(float64(r)*math.Cos(float64(t)*deg)), cy+int(float64(r)*math.Sin(float64(t)*deg)), c)
	}
	// Hour hand pointing to 10 o'clock.
	drawPolyline(img, []image.Point{{cx, cy}, {cx - 4, cy - 5}}, c, 1)
	// Minute hand pointing to 2 o'clock.
	drawPolyline(img, []image.Point{{cx, cy}, {cx + 6, cy - 3}}, c, 1)
}

// drawPause draws two vertical white bars (pause symbol).
func drawPause(img *image.RGBA, c color.Color) {
	for x := 11; x <= 13; x++ {
		for y := 10; y <= 22; y++ {
			img.Set(x, y, c)
		}
	}
	for x := 18; x <= 20; x++ {
		for y := 10; y <= 22; y++ {
			img.Set(x, y, c)
		}
	}
}

// drawError draws a white exclamation mark.
func drawError(img *image.RGBA, c color.Color) {
	// Vertical bar.
	for y := 8; y <= 19; y++ {
		img.Set(15, y, c)
		img.Set(16, y, c)
	}
	// Dot.
	for y := 22; y <= 24; y++ {
		img.Set(15, y, c)
		img.Set(16, y, c)
	}
}

// drawPolyline draws connected line segments between the given points
// with the specified thickness.
func drawPolyline(img *image.RGBA, pts []image.Point, c color.Color, thickness int) {
	for i := 0; i < len(pts)-1; i++ {
		drawLine(img, pts[i], pts[i+1], c, thickness)
	}
}

// drawLine draws a single line segment using Bresenham's algorithm,
// thickened by drawing parallel offsets.
func drawLine(img *image.RGBA, p0, p1 image.Point, c color.Color, thickness int) {
	dx := abs(p1.X - p0.X)
	dy := abs(p1.Y - p0.Y)
	sx := step(p0.X, p1.X)
	sy := step(p0.Y, p1.Y)
	err := dx - dy
	x, y := p0.X, p0.Y
	for {
		for ox := -thickness; ox <= thickness; ox++ {
			for oy := -thickness; oy <= thickness; oy++ {
				px, py := x+ox, y+oy
				if px >= 0 && px < iconSize && py >= 0 && py < iconSize {
					img.Set(px, py, c)
				}
			}
		}
		if x == p1.X && y == p1.Y {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x += sx
		}
		if e2 < dx {
			err += dx
			y += sy
		}
	}
}

// drawPoint sets a single pixel if within bounds.
func drawPoint(img *image.RGBA, x, y int, c color.Color) {
	if x >= 0 && x < iconSize && y >= 0 && y < iconSize {
		img.Set(x, y, c)
	}
}

// darker returns a color with each RGB channel reduced by n, clamped to 0.
func darker(c color.RGBA, n uint8) color.RGBA {
	return color.RGBA{
		R: subU8(c.R, n),
		G: subU8(c.G, n),
		B: subU8(c.B, n),
		A: c.A,
	}
}

// subU8 subtracts n from v with clamping to 0.
func subU8(v, n uint8) uint8 {
	if v < n {
		return 0
	}
	return v - n
}

// abs returns the absolute value of x.
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// step returns the sign of the direction from a to b (-1, 0, or 1).
func step(a, b int) int {
	if a < b {
		return 1
	}
	if a > b {
		return -1
	}
	return 0
}

// deg is the conversion factor from degrees to radians.
const deg = math.Pi / 180.0
