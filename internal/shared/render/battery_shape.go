package render

import (
	"image"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
)

// BatteryShapeConfig holds parameters for drawing a battery shape.
type BatteryShapeConfig struct {
	Orientation string // "horizontal" or "vertical"
	Percentage  int    // 0-100 fill level
	FillColor   uint8  // color for the proportional fill bar
	BorderColor uint8  // color for the outline and terminal nub
	Padding     int    // inner padding from the region edges
}

// DrawBatteryShape draws a battery outline with proportional fill into the
// given rectangular region (x, y, w, h) of the image. The shape includes a
// body rectangle and a positive terminal nub (right side for horizontal,
// top for vertical). The caller is responsible for any overlays (text,
// status icons, etc.).
func DrawBatteryShape(img *image.Gray, x, y, w, h int, cfg BatteryShapeConfig) {
	// Clamp percentage to valid range
	pct := cfg.Percentage
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}

	if cfg.Orientation == "vertical" {
		drawBatteryVertical(img, x, y, w, h, pct, cfg)
	} else {
		drawBatteryHorizontal(img, x, y, w, h, pct, cfg)
	}
}

// drawBatteryHorizontal draws a horizontal battery with the nub on the right.
func drawBatteryHorizontal(img *image.Gray, x, y, w, h, pct int, cfg BatteryShapeConfig) {
	nubW := 4
	batteryX := x + cfg.Padding
	batteryY := y + cfg.Padding
	batteryW := w - cfg.Padding*2 - nubW - 1
	batteryH := h - cfg.Padding*2

	if batteryW < 8 {
		batteryW = 8
	}
	if batteryH < 6 {
		batteryH = 6
	}

	// Body outline
	bitmap.DrawRectangle(img, batteryX, batteryY, batteryW, batteryH, cfg.BorderColor)

	// Positive terminal nub on the right
	nubH := batteryH / 3
	if nubH < 4 {
		nubH = 4
	}
	nubX := batteryX + batteryW
	nubY := batteryY + (batteryH-nubH)/2
	bitmap.DrawFilledRectangle(img, nubX, nubY, nubW, nubH, cfg.BorderColor)

	// Proportional fill inside the body
	fillMargin := 2
	fillX := batteryX + fillMargin
	fillY := batteryY + fillMargin
	fillMaxW := batteryW - 2*fillMargin
	fillH := batteryH - 2*fillMargin
	fillW := int(float64(fillMaxW) * float64(pct) / 100.0)

	if fillW > 0 {
		bitmap.DrawFilledRectangle(img, fillX, fillY, fillW, fillH, cfg.FillColor)
	}
}

// drawBatteryVertical draws a vertical battery with the nub on top.
func drawBatteryVertical(img *image.Gray, x, y, w, h, pct int, cfg BatteryShapeConfig) {
	nubH := 4
	batteryX := x + cfg.Padding
	batteryY := y + cfg.Padding + nubH + 1
	batteryW := w - cfg.Padding*2
	batteryH := h - cfg.Padding*2 - nubH - 1

	if batteryW < 6 {
		batteryW = 6
	}
	if batteryH < 8 {
		batteryH = 8
	}

	// Body outline
	bitmap.DrawRectangle(img, batteryX, batteryY, batteryW, batteryH, cfg.BorderColor)

	// Positive terminal nub on top
	nubW := batteryW / 3
	if nubW < 4 {
		nubW = 4
	}
	nubX := batteryX + (batteryW-nubW)/2
	nubY := y + cfg.Padding
	bitmap.DrawFilledRectangle(img, nubX, nubY, nubW, nubH, cfg.BorderColor)

	// Proportional fill inside the body (fills from bottom up)
	fillMargin := 2
	fillX := batteryX + fillMargin
	fillMaxH := batteryH - 2*fillMargin
	fillW := batteryW - 2*fillMargin
	fillH := int(float64(fillMaxH) * float64(pct) / 100.0)
	fillY := batteryY + fillMargin + fillMaxH - fillH

	if fillH > 0 {
		bitmap.DrawFilledRectangle(img, fillX, fillY, fillW, fillH, cfg.FillColor)
	}
}
