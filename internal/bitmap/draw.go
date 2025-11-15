package bitmap

import (
	"image"
	"image/color"
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
		drawLine(img, points[i][0], points[i][1], points[i+1][0], points[i+1][1], c)
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

// drawLine draws a line between two points (Bresenham's algorithm)
func drawLine(img *image.Gray, x0, y0, x1, y1 int, c color.Gray) {
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

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
