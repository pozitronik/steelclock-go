// Package claudecode provides bubble drawing for comic-style speech and thought bubbles.
package claudecode

import (
	"image"
	"image/color"
)

// BubbleType defines the type of comic bubble
type BubbleType int

const (
	BubbleThought BubbleType = iota // Cloud-like thought bubble
	BubbleSpeech                    // Speech bubble with tail
)

// drawBubble draws a comic bubble at the specified position
// x, y is the top-left corner of the bubble content area
// w, h is the size of the content area (text will go inside)
// tailX, tailY is where the tail should point (towards Clawd)
func drawBubble(img *image.Gray, bubbleType BubbleType, x, y, w, h, tailX, tailY int) {
	switch bubbleType {
	case BubbleThought:
		drawThoughtBubble(img, x, y, w, h, tailX, tailY)
	case BubbleSpeech:
		drawSpeechBubble(img, x, y, w, h, tailX, tailY)
	}
}

// drawSpeechBubble draws a rounded speech bubble with a triangular tail pointing to Clawd
func drawSpeechBubble(img *image.Gray, x, y, w, h, tailX, tailY int) {
	white := color.Gray{Y: 255} // Use white for outline on black background

	// Padding around content for the bubble (increased for readability)
	padX := 5
	padY := 3

	// Bubble bounds (outer edge)
	bx := x - padX
	by := y - padY
	bw := w + padX*2
	bh := h + padY*2

	// Corner radius
	radius := 3
	if radius > bh/2 {
		radius = bh / 2
	}
	if radius > bw/2 {
		radius = bw / 2
	}

	// Tail: starts from bottom-left area of bubble, points down-left toward Clawd
	tailStartX := bx + 2      // Start near left edge of bubble
	tailStartY := by + bh - 1 // Start at bottom of bubble
	tailTipX := tailX         // Point toward Clawd X
	tailTipY := tailY         // Point toward Clawd Y
	tailWidth := 4            // Width at base

	// Draw the bubble border (rounded rectangle outline)
	// Top edge (between corners)
	for px := bx + radius; px < bx+bw-radius; px++ {
		setPixelSafe(img, px, by, white)
	}
	// Bottom edge (skip tail opening)
	for px := bx + radius; px < bx+bw-radius; px++ {
		// Skip the tail opening area
		if px >= tailStartX && px < tailStartX+tailWidth {
			continue
		}
		setPixelSafe(img, px, by+bh-1, white)
	}
	// Left edge
	for py := by + radius; py < by+bh-radius; py++ {
		setPixelSafe(img, bx, py, white)
	}
	// Right edge
	for py := by + radius; py < by+bh-radius; py++ {
		setPixelSafe(img, bx+bw-1, py, white)
	}

	// Draw corners (quarter circles)
	drawCorner(img, bx+radius, by+radius, radius, white, 0)           // top-left
	drawCorner(img, bx+bw-radius-1, by+radius, radius, white, 1)      // top-right
	drawCorner(img, bx+radius, by+bh-radius-1, radius, white, 2)      // bottom-left
	drawCorner(img, bx+bw-radius-1, by+bh-radius-1, radius, white, 3) // bottom-right

	// Draw the tail pointing toward Clawd (diagonal line from bubble to Clawd)
	// Left edge of tail
	drawLine(img, tailStartX, tailStartY, tailTipX, tailTipY, white)
	// Right edge of tail
	drawLine(img, tailStartX+tailWidth, tailStartY, tailTipX, tailTipY, white)
}

// drawCorner draws a quarter circle for rounded corners
// quadrant: 0=top-left, 1=top-right, 2=bottom-left, 3=bottom-right
func drawCorner(img *image.Gray, cx, cy, radius int, c color.Gray, quadrant int) {
	for angle := 0; angle <= 90; angle++ {
		// Calculate point on circle
		var dx, dy int
		switch quadrant {
		case 0: // top-left
			dx = -radius + angle*radius/90
			dy = -radius + (90-angle)*radius/90
		case 1: // top-right
			dx = radius - angle*radius/90
			dy = -radius + (90-angle)*radius/90
		case 2: // bottom-left
			dx = -radius + angle*radius/90
			dy = radius - (90-angle)*radius/90
		case 3: // bottom-right
			dx = radius - angle*radius/90
			dy = radius - (90-angle)*radius/90
		}
		setPixelSafe(img, cx+dx, cy+dy, c)
	}
}

// drawThoughtBubble draws a cloud-like thought bubble with trailing circles (no fill)
func drawThoughtBubble(img *image.Gray, x, y, w, h, tailX, tailY int) {
	white := color.Gray{Y: 255} // Use white for outline on black background

	// Padding around content (increased for readability)
	padX := 5
	padY := 3
	bx := x - padX
	by := y - padY
	bw := w + padX*2
	bh := h + padY*2

	// Draw cloud body border (bumpy rounded rectangle, no fill)
	// Top edge with bumps
	for px := bx + 2; px < bx+bw-2; px++ {
		offset := 0
		if (px/3)%2 == 0 {
			offset = -1
		}
		setPixelSafe(img, px, by+offset, white)
	}

	// Bottom edge with bumps
	for px := bx + 2; px < bx+bw-2; px++ {
		offset := 0
		if (px/3)%2 == 0 {
			offset = 1
		}
		setPixelSafe(img, px, by+bh-1+offset, white)
	}

	// Left edge with bumps
	for py := by + 2; py < by+bh-2; py++ {
		offset := 0
		if (py/3)%2 == 0 {
			offset = -1
		}
		setPixelSafe(img, bx+offset, py, white)
	}

	// Right edge with bumps
	for py := by + 2; py < by+bh-2; py++ {
		offset := 0
		if (py/3)%2 == 0 {
			offset = 1
		}
		setPixelSafe(img, bx+bw-1+offset, py, white)
	}

	// Corners
	setPixelSafe(img, bx+1, by, white)
	setPixelSafe(img, bx, by+1, white)
	setPixelSafe(img, bx+bw-2, by, white)
	setPixelSafe(img, bx+bw-1, by+1, white)
	setPixelSafe(img, bx+1, by+bh-1, white)
	setPixelSafe(img, bx, by+bh-2, white)
	setPixelSafe(img, bx+bw-2, by+bh-1, white)
	setPixelSafe(img, bx+bw-1, by+bh-2, white)

	// Draw trailing thought circles from top of Clawd's head to bubble (outline only)
	// tailX, tailY is at the top of Clawd's head
	// Circles go from head to middle of bubble's left border
	startX, startY := tailX, tailY // Top of Clawd's head
	endX, endY := bx, by+bh/2      // Middle of bubble's left border

	// Place 3 circles along the vertical path, getting larger toward bubble
	// Circle 1 (smallest, closest to Clawd's head)
	c1x := startX + (endX-startX)/4
	c1y := startY + (endY-startY)/4
	setPixelSafe(img, c1x, c1y, white)
	setPixelSafe(img, c1x+1, c1y, white)

	// Circle 2 (medium, middle of path)
	c2x := startX + (endX-startX)/2
	c2y := startY + (endY-startY)/2
	drawCircleOutline(img, c2x, c2y, 1, white)

	// Circle 3 (largest, closest to bubble)
	c3x := startX + (endX-startX)*3/4
	c3y := startY + (endY-startY)*3/4
	drawCircleOutline(img, c3x, c3y, 2, white)
}

// drawLine draws a line using Bresenham's algorithm
func drawLine(img *image.Gray, x0, y0, x1, y1 int, c color.Gray) {
	dx := abs(x1 - x0)
	dy := -abs(y1 - y0)
	sx := 1
	if x0 >= x1 {
		sx = -1
	}
	sy := 1
	if y0 >= y1 {
		sy = -1
	}
	err := dx + dy

	for {
		setPixelSafe(img, x0, y0, c)
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

// drawCircleOutline draws a circle outline (no fill)
func drawCircleOutline(img *image.Gray, cx, cy, radius int, c color.Gray) {
	x := radius
	y := 0
	err := 0

	for x >= y {
		setPixelSafe(img, cx+x, cy+y, c)
		setPixelSafe(img, cx+y, cy+x, c)
		setPixelSafe(img, cx-y, cy+x, c)
		setPixelSafe(img, cx-x, cy+y, c)
		setPixelSafe(img, cx-x, cy-y, c)
		setPixelSafe(img, cx-y, cy-x, c)
		setPixelSafe(img, cx+y, cy-x, c)
		setPixelSafe(img, cx+x, cy-y, c)

		y++
		if err <= 0 {
			err += 2*y + 1
		}
		if err > 0 {
			x--
			err -= 2*x + 1
		}
	}
}

// abs returns absolute value of an integer
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// setPixelSafe sets a pixel if it's within bounds
func setPixelSafe(img *image.Gray, x, y int, c color.Gray) {
	if x >= 0 && y >= 0 && x < img.Bounds().Dx() && y < img.Bounds().Dy() {
		img.SetGray(x, y, c)
	}
}

// getBubbleTypeForState returns the appropriate bubble type for a given state
func getBubbleTypeForState(state State) BubbleType {
	switch state {
	case StateThinking:
		return BubbleThought
	default:
		return BubbleSpeech
	}
}
