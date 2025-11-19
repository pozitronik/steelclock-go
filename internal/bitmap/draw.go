package bitmap

import (
	"image"
	"image/color"
	"math"

	"github.com/pozitronik/steelclock-go/internal/config"
)

// DrawHorizontalBar draws a horizontal progress bar
func DrawHorizontalBar(img *image.Gray, x, y, w, h int, percentage float64, fillColor uint8, drawBorder bool) {
	c := color.Gray{Y: fillColor}

	// Draw border if requested
	if drawBorder {
		for i := x; i < x+w; i++ {
			img.Set(i, y, c)
			img.Set(i, y+h-1, c)
		}
		for i := y; i < y+h; i++ {
			img.Set(x, i, c)
			img.Set(x+w-1, i, c)
		}

		// Fill inside border
		fillW := int(float64(w-2) * (percentage / 100.0))
		if fillW > 0 {
			for py := y + 1; py < y+h-1; py++ {
				for px := x + 1; px < x+1+fillW; px++ {
					img.Set(px, py, c)
				}
			}
		}
	} else {
		// Fill without border
		fillW := int(float64(w) * (percentage / 100.0))
		if fillW > 0 {
			for py := y; py < y+h; py++ {
				for px := x; px < x+fillW; px++ {
					img.Set(px, py, c)
				}
			}
		}
	}
}

// DrawVerticalBar draws a vertical progress bar (fills from bottom)
func DrawVerticalBar(img *image.Gray, x, y, w, h int, percentage float64, fillColor uint8, drawBorder bool) {
	c := color.Gray{Y: fillColor}

	fillH := int(float64(h) * (percentage / 100.0))
	fillY := y + h - fillH

	if drawBorder {
		// Draw border
		for i := x; i < x+w; i++ {
			img.Set(i, y, c)
			img.Set(i, y+h-1, c)
		}
		for i := y; i < y+h; i++ {
			img.Set(x, i, c)
			img.Set(x+w-1, i, c)
		}

		// Fill inside border (bottom to top)
		if fillH > 2 {
			startY := fillY
			if startY < y+1 {
				startY = y + 1
			}
			for py := startY; py < y+h-1; py++ {
				for px := x + 1; px < x+w-1; px++ {
					img.Set(px, py, c)
				}
			}
		}
	} else {
		// Fill without border (bottom to top)
		if fillH > 0 {
			for py := fillY; py < y+h; py++ {
				for px := x; px < x+w; px++ {
					img.Set(px, py, c)
				}
			}
		}
	}
}

// DrawGraph draws a history graph with filled area
func DrawGraph(img *image.Gray, x, y, w, h int, history []float64, maxHistory int, fillColor uint8) {
	if len(history) < 2 {
		return
	}

	c := color.Gray{Y: fillColor}
	cSemi := color.Gray{Y: fillColor / 2}

	// Calculate points
	points := make([][2]int, 0, len(history))
	offset := maxHistory - len(history)

	for i, sample := range history {
		px := x + int(float64(offset+i)/float64(maxHistory-1)*float64(w))
		py := y + h - int((sample/100.0)*float64(h))
		points = append(points, [2]int{px, py})
	}

	// Draw line
	for i := 0; i < len(points)-1; i++ {
		DrawLine(img, points[i][0], points[i][1], points[i+1][0], points[i+1][1], c)
	}

	// Fill area under a line
	for i := 0; i < len(points)-1; i++ {
		x1, y1 := points[i][0], points[i][1]
		x2, y2 := points[i+1][0], points[i+1][1]

		// Fill vertical strips between consecutive points
		for px := x1; px <= x2; px++ {
			// Interpolate y coordinate
			t := float64(px-x1) / float64(x2-x1+1)
			py := int(float64(y1) + t*float64(y2-y1))

			// Fill from py to bottom
			for fy := py; fy < y+h; fy++ {
				img.Set(px, fy, cSemi)
			}
		}
	}
}

// DrawDualGauge draws a nested/concentric double gauge with two needles
// Outer gauge (larger radius) for primary value, inner gauge (smaller radius) for secondary value
func DrawDualGauge(img *image.Gray, pos config.PositionConfig, outerPercentage, innerPercentage float64, outerGaugeColor, outerNeedleColor, innerGaugeColor, innerNeedleColor uint8) {
	centerX := pos.W / 2
	centerY := pos.H - 3 // Near bottom

	// Calculate radii - outer is larger, inner is ~60% of outer
	outerRadius := pos.H - 6
	if pos.W/2 < outerRadius {
		outerRadius = pos.W/2 - 3
	}

	innerRadius := int(float64(outerRadius) * 0.6)

	if outerRadius <= 0 || innerRadius <= 0 {
		return
	}

	outerGColor := color.Gray{Y: outerGaugeColor}
	outerNColor := color.Gray{Y: outerNeedleColor}
	innerGColor := color.Gray{Y: innerGaugeColor}
	innerNColor := color.Gray{Y: innerNeedleColor}

	// Draw outer gauge arc (semicircle from 180° to 0°)
	for angle := 180.0; angle >= 0; angle -= 2.0 {
		rad := angle * math.Pi / 180.0
		x := centerX + int(float64(outerRadius)*math.Cos(rad))
		y := centerY - int(float64(outerRadius)*math.Sin(rad))

		if x >= 0 && x < pos.W && y >= 0 && y < pos.H {
			img.Set(x, y, outerGColor)
		}
	}

	// Draw inner gauge arc (semicircle from 180° to 0°)
	for angle := 180.0; angle >= 0; angle -= 2.0 {
		rad := angle * math.Pi / 180.0
		x := centerX + int(float64(innerRadius)*math.Cos(rad))
		y := centerY - int(float64(innerRadius)*math.Sin(rad))

		if x >= 0 && x < pos.W && y >= 0 && y < pos.H {
			img.Set(x, y, innerGColor)
		}
	}

	// Draw outer gauge tick marks
	for tick := 0; tick <= 10; tick++ {
		angle := 180.0 - float64(tick)*18.0 // 0-180 degrees in 10 steps
		rad := angle * math.Pi / 180.0

		// Outer point
		x1 := centerX + int(float64(outerRadius)*math.Cos(rad))
		y1 := centerY - int(float64(outerRadius)*math.Sin(rad))

		// Inner point
		tickLen := 3
		if tick%5 == 0 {
			tickLen = 5 // Longer ticks at 0%, 50%, 100%
		}
		x2 := centerX + int(float64(outerRadius-tickLen)*math.Cos(rad))
		y2 := centerY - int(float64(outerRadius-tickLen)*math.Sin(rad))

		DrawLine(img, x1, y1, x2, y2, outerGColor)
	}

	// Draw inner gauge tick marks (smaller, fewer)
	for tick := 0; tick <= 10; tick += 2 { // Every other tick
		angle := 180.0 - float64(tick)*18.0
		rad := angle * math.Pi / 180.0

		// Outer point
		x1 := centerX + int(float64(innerRadius)*math.Cos(rad))
		y1 := centerY - int(float64(innerRadius)*math.Sin(rad))

		// Inner point
		tickLen := 2
		x2 := centerX + int(float64(innerRadius-tickLen)*math.Cos(rad))
		y2 := centerY - int(float64(innerRadius-tickLen)*math.Sin(rad))

		DrawLine(img, x1, y1, x2, y2, innerGColor)
	}

	// Draw outer needle (from inner radius to outer radius, doesn't overlap inner gauge)
	outerNeedleAngle := 180.0 - (outerPercentage / 100.0 * 180.0)
	outerNeedleRad := outerNeedleAngle * math.Pi / 180.0
	outerNeedleLen := outerRadius - 2

	// Start from inner radius edge (doesn't go through inner gauge)
	outerNeedleStartX := centerX + int(float64(innerRadius)*math.Cos(outerNeedleRad))
	outerNeedleStartY := centerY - int(float64(innerRadius)*math.Sin(outerNeedleRad))

	// End at outer radius
	outerNeedleEndX := centerX + int(float64(outerNeedleLen)*math.Cos(outerNeedleRad))
	outerNeedleEndY := centerY - int(float64(outerNeedleLen)*math.Sin(outerNeedleRad))

	DrawLine(img, outerNeedleStartX, outerNeedleStartY, outerNeedleEndX, outerNeedleEndY, outerNColor)

	// Draw inner needle (shorter, thinner)
	innerNeedleAngle := 180.0 - (innerPercentage / 100.0 * 180.0)
	innerNeedleRad := innerNeedleAngle * math.Pi / 180.0
	innerNeedleLen := innerRadius - 2

	innerNeedleX := centerX + int(float64(innerNeedleLen)*math.Cos(innerNeedleRad))
	innerNeedleY := centerY - int(float64(innerNeedleLen)*math.Sin(innerNeedleRad))

	DrawLine(img, centerX, centerY, innerNeedleX, innerNeedleY, innerNColor)

	// Draw center point (use inner needle color since only inner needle touches center)
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if centerX+dx >= 0 && centerX+dx < pos.W && centerY+dy >= 0 && centerY+dy < pos.H {
				img.Set(centerX+dx, centerY+dy, innerNColor)
			}
		}
	}
}

// DrawGauge draws a semicircular gauge with needle
func DrawGauge(img *image.Gray, pos config.PositionConfig, percentage float64, gaugeColor, needleColor uint8) {
	centerX := pos.W / 2
	centerY := pos.H - 3 // Near bottom

	// Radius of the gauge arc
	radius := pos.H - 6
	if pos.W/2 < radius {
		radius = pos.W/2 - 3
	}

	if radius <= 0 {
		return
	}

	gColor := color.Gray{Y: gaugeColor}
	nColor := color.Gray{Y: needleColor}

	// Draw gauge arc (semicircle from 180° to 0°)
	for angle := 180.0; angle >= 0; angle -= 2.0 {
		rad := angle * math.Pi / 180.0
		x := centerX + int(float64(radius)*math.Cos(rad))
		y := centerY - int(float64(radius)*math.Sin(rad))

		if x >= 0 && x < pos.W && y >= 0 && y < pos.H {
			img.Set(x, y, gColor)
		}
	}

	// Draw tick marks
	for tick := 0; tick <= 10; tick++ {
		angle := 180.0 - float64(tick)*18.0 // 0-180 degrees in 10 steps
		rad := angle * math.Pi / 180.0

		// Outer point
		x1 := centerX + int(float64(radius)*math.Cos(rad))
		y1 := centerY - int(float64(radius)*math.Sin(rad))

		// Inner point
		tickLen := 3
		if tick%5 == 0 {
			tickLen = 5 // Longer ticks at 0%, 50%, 100%
		}
		x2 := centerX + int(float64(radius-tickLen)*math.Cos(rad))
		y2 := centerY - int(float64(radius-tickLen)*math.Sin(rad))

		DrawLine(img, x1, y1, x2, y2, gColor)
	}

	// Draw needle based on percentage
	needleAngle := 180.0 - (percentage / 100.0 * 180.0)
	needleRad := needleAngle * math.Pi / 180.0
	needleLen := radius - 2

	needleX := centerX + int(float64(needleLen)*math.Cos(needleRad))
	needleY := centerY - int(float64(needleLen)*math.Sin(needleRad))

	DrawLine(img, centerX, centerY, needleX, needleY, nColor)

	// Draw center point
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if centerX+dx >= 0 && centerX+dx < pos.W && centerY+dy >= 0 && centerY+dy < pos.H {
				img.Set(centerX+dx, centerY+dy, nColor)
			}
		}
	}
}

// DrawLine draws a line between two points (Bresenham's algorithm)
func DrawLine(img *image.Gray, x0, y0, x1, y1 int, c color.Gray) {
	dx := abs(x1 - x0)
	dy := abs(y1 - y0)
	sx := -1
	if x0 < x1 {
		sx = 1
	}
	sy := -1
	if y0 < y1 {
		sy = 1
	}
	err := dx - dy

	for {
		img.Set(x0, y0, c)

		if x0 == x1 && y0 == y1 {
			break
		}

		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x0 += sx
		}
		if e2 < dx {
			err += dx
			y0 += sy
		}
	}
}

// DrawCircle draws a circle with the given center and radius
func DrawCircle(img *image.Gray, centerX, centerY, radius int, c color.Gray) {
	// Bresenham's circle algorithm
	x := 0
	y := radius
	d := 3 - 2*radius

	drawCirclePoints(img, centerX, centerY, x, y, c)

	for x <= y {
		if d <= 0 {
			d = d + 4*x + 6
		} else {
			d = d + 4*(x-y) + 10
			y--
		}
		x++
		drawCirclePoints(img, centerX, centerY, x, y, c)
	}
}

// drawCirclePoints draws 8 symmetric points for a circle
func drawCirclePoints(img *image.Gray, centerX, centerY, x, y int, c color.Gray) {
	bounds := img.Bounds()

	points := [][2]int{
		{centerX + x, centerY + y},
		{centerX - x, centerY + y},
		{centerX + x, centerY - y},
		{centerX - x, centerY - y},
		{centerX + y, centerY + x},
		{centerX - y, centerY + x},
		{centerX + y, centerY - x},
		{centerX - y, centerY - x},
	}

	for _, p := range points {
		if p[0] >= bounds.Min.X && p[0] < bounds.Max.X &&
			p[1] >= bounds.Min.Y && p[1] < bounds.Max.Y {
			img.Set(p[0], p[1], c)
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
