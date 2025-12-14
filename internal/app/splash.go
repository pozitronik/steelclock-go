package app

import (
	"image"
	"image/color"
	"strings"
	"time"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/bitmap/glyphs"
	"github.com/pozitronik/steelclock-go/internal/display"
)

// Splash animation constants
const (
	// StartupAnimationDuration is total duration of startup animation
	StartupAnimationDuration = 1500 * time.Millisecond
	// StartupFrameInterval is time between animation frames
	StartupFrameInterval = 30 * time.Millisecond

	// TransitionAnimationDuration is duration of profile transition banner
	TransitionAnimationDuration = 1200 * time.Millisecond
	// TransitionFrameInterval is time between transition frames
	TransitionFrameInterval = 30 * time.Millisecond

	// ExitAnimationDuration is duration of exit message
	ExitAnimationDuration = 800 * time.Millisecond
	// ExitFrameInterval is time between exit frames
	ExitFrameInterval = 30 * time.Millisecond

	// PreviewModeMessageDuration is how long to show preview mode message
	PreviewModeMessageDuration = 500 * time.Millisecond
)

// SplashRenderer handles animated splash screens
type SplashRenderer struct {
	client display.FrameSender
	width  int
	height int
}

// NewSplashRenderer creates a new splash renderer
func NewSplashRenderer(client display.FrameSender, width, height int) *SplashRenderer {
	return &SplashRenderer{
		client: client,
		width:  width,
		height: height,
	}
}

// ShowStartupAnimation displays the startup logo animation
// Returns nil if successful, skips gracefully if client is nil
func (s *SplashRenderer) ShowStartupAnimation() error {
	if s.client == nil {
		return nil
	}

	frames := int(StartupAnimationDuration / StartupFrameInterval)
	ticker := time.NewTicker(StartupFrameInterval)
	defer ticker.Stop()

	for frame := 0; frame < frames; frame++ {
		progress := float64(frame) / float64(frames-1)
		img := s.renderStartupFrame(progress)

		if err := s.sendFrame(img); err != nil {
			return err
		}

		<-ticker.C
	}

	return nil
}

// ShowTransitionBanner displays the profile transition animation
func (s *SplashRenderer) ShowTransitionBanner(profileName string) error {
	if s.client == nil {
		return nil
	}

	frames := int(TransitionAnimationDuration / TransitionFrameInterval)
	ticker := time.NewTicker(TransitionFrameInterval)
	defer ticker.Stop()

	for frame := 0; frame < frames; frame++ {
		progress := float64(frame) / float64(frames-1)
		img := s.renderTransitionFrame(profileName, progress)

		if err := s.sendFrame(img); err != nil {
			return err
		}

		<-ticker.C
	}

	return nil
}

// ShowExitMessage displays the goodbye animation
func (s *SplashRenderer) ShowExitMessage() error {
	if s.client == nil {
		return nil
	}

	frames := int(ExitAnimationDuration / ExitFrameInterval)
	ticker := time.NewTicker(ExitFrameInterval)
	defer ticker.Stop()

	for frame := 0; frame < frames; frame++ {
		progress := float64(frame) / float64(frames-1)
		img := s.renderExitFrame(progress)

		if err := s.sendFrame(img); err != nil {
			return err
		}

		<-ticker.C
	}

	// Send final blank frame
	blank := image.NewGray(image.Rect(0, 0, s.width, s.height))
	return s.sendFrame(blank)
}

// ShowPreviewModeMessage displays "PREVIEW MODE" on the hardware display
// This is shown before switching to preview backend so user knows display is paused
func (s *SplashRenderer) ShowPreviewModeMessage() error {
	if s.client == nil {
		return nil
	}

	img := s.renderPreviewModeFrame()
	if err := s.sendFrame(img); err != nil {
		return err
	}

	// Hold the message briefly
	time.Sleep(PreviewModeMessageDuration)
	return nil
}

// renderPreviewModeFrame renders the "PREVIEW MODE" static frame
func (s *SplashRenderer) renderPreviewModeFrame() *image.Gray {
	img := image.NewGray(image.Rect(0, 0, s.width, s.height))

	font := glyphs.Font5x7
	text := "PREVIEW MODE"

	textWidth := glyphs.MeasureText(text, font)
	textHeight := font.GlyphHeight

	// Center the text
	textX := (s.width - textWidth) / 2
	textY := (s.height - textHeight) / 2

	// Draw text
	glyphs.DrawText(img, text, textX, textY, font, color.Gray{Y: 255})

	// Draw decorative border lines
	lineY1 := textY - 5
	lineY2 := textY + textHeight + 4
	lineStart := textX - 10
	lineEnd := textX + textWidth + 10

	if lineStart < 2 {
		lineStart = 2
	}
	if lineEnd > s.width-2 {
		lineEnd = s.width - 2
	}

	lineColor := color.Gray{Y: 128}
	for x := lineStart; x < lineEnd; x++ {
		if lineY1 >= 0 && lineY1 < s.height {
			img.Set(x, lineY1, lineColor)
		}
		if lineY2 >= 0 && lineY2 < s.height {
			img.Set(x, lineY2, lineColor)
		}
	}

	return img
}

// renderStartupFrame renders a single frame of the startup animation
// Animation: "STEELCLOCK" text with sweeping reveal from left to right,
// followed by a scanning line effect
func (s *SplashRenderer) renderStartupFrame(progress float64) *image.Gray {
	img := image.NewGray(image.Rect(0, 0, s.width, s.height))

	// Logo text
	logoText := "STEELCLOCK"

	// Get the 5x7 font for the logo
	font := glyphs.Font5x7
	textWidth := glyphs.MeasureText(logoText, font)
	textHeight := font.GlyphHeight

	// Center the text
	textX := (s.width - textWidth) / 2
	textY := (s.height - textHeight) / 2

	// Phase 1 (0-0.6): Reveal text from left to right
	// Phase 2 (0.6-0.8): Scanning line effect
	// Phase 3 (0.8-1.0): Full brightness stabilization

	if progress < 0.6 {
		// Reveal phase - draw text progressively
		revealProgress := progress / 0.6
		revealX := textX + int(float64(textWidth)*revealProgress)

		// Draw revealed portion with full brightness
		s.drawTextClipped(img, logoText, textX, textY, font, textX, revealX)

		// Draw vertical scan line at reveal edge
		if revealX < textX+textWidth {
			scanLineColor := color.Gray{Y: 255}
			for y := textY - 2; y < textY+textHeight+2; y++ {
				if y >= 0 && y < s.height {
					img.Set(revealX, y, scanLineColor)
					if revealX+1 < s.width {
						img.Set(revealX+1, y, color.Gray{Y: 128})
					}
				}
			}
		}
	} else if progress < 0.8 {
		// Scan line effect phase - full text visible with moving highlight
		glyphs.DrawText(img, logoText, textX, textY, font, color.Gray{Y: 200})

		// Draw a brighter scan line moving across
		scanProgress := (progress - 0.6) / 0.2
		scanX := textX + int(float64(textWidth)*scanProgress)

		for y := textY - 2; y < textY+textHeight+2; y++ {
			if y >= 0 && y < s.height {
				for dx := -2; dx <= 2; dx++ {
					px := scanX + dx
					if px >= 0 && px < s.width {
						brightness := uint8(255 - abs(dx)*50)
						current := img.GrayAt(px, y).Y
						if brightness > current {
							img.Set(px, y, color.Gray{Y: brightness})
						}
					}
				}
			}
		}
	} else {
		// Stabilization phase - just show full logo
		glyphs.DrawText(img, logoText, textX, textY, font, color.Gray{Y: 255})
	}

	// Draw decorative lines at top and bottom
	lineY1 := textY - 5
	lineY2 := textY + textHeight + 4

	if progress > 0.3 {
		lineProgress := (progress - 0.3) / 0.7
		lineWidth := int(float64(textWidth+20) * lineProgress)
		lineStartX := (s.width - lineWidth) / 2

		for x := lineStartX; x < lineStartX+lineWidth; x++ {
			if x >= 0 && x < s.width {
				if lineY1 >= 0 && lineY1 < s.height {
					img.Set(x, lineY1, color.Gray{Y: 180})
				}
				if lineY2 >= 0 && lineY2 < s.height {
					img.Set(x, lineY2, color.Gray{Y: 180})
				}
			}
		}
	}

	return img
}

// renderTransitionFrame renders a single frame of the profile transition
// Animation: Profile name slides in from right, holds, then fades
func (s *SplashRenderer) renderTransitionFrame(profileName string, progress float64) *image.Gray {
	img := image.NewGray(image.Rect(0, 0, s.width, s.height))

	// Special case: Winamp gets its iconic slogan on two lines
	if strings.EqualFold(profileName, "Winamp") {
		s.renderWinampSlogan(img, progress)
		return img
	}

	font := glyphs.Font5x7

	// Profile name centered
	nameWidth := glyphs.MeasureText(profileName, font)
	nameX := (s.width - nameWidth) / 2
	nameY := (s.height - font.GlyphHeight) / 2

	// Phase 1 (0-0.2): Slide in from right
	// Phase 2 (0.2-0.8): Hold
	// Phase 3 (0.8-1.0): Fade out

	var brightness uint8
	var offsetX int

	if progress < 0.2 {
		// Slide in phase
		slideProgress := progress / 0.2
		offsetX = int(float64(s.width) * (1 - slideProgress))
		brightness = 255
	} else if progress < 0.8 {
		// Hold phase
		offsetX = 0
		brightness = 255
	} else {
		// Fade out phase
		fadeProgress := (progress - 0.8) / 0.2
		offsetX = 0
		brightness = uint8(255 * (1 - fadeProgress))
	}

	// Draw decorative lines above and below the name
	if brightness > 50 {
		lineColor := color.Gray{Y: brightness / 2}
		lineY1 := nameY - 4
		lineY2 := nameY + font.GlyphHeight + 3

		// Calculate line width based on text width
		lineStart := nameX - 10
		lineEnd := nameX + nameWidth + 10
		if lineStart < 5 {
			lineStart = 5
		}
		if lineEnd > s.width-5 {
			lineEnd = s.width - 5
		}

		for x := lineStart; x < lineEnd; x++ {
			if x+offsetX >= 0 && x+offsetX < s.width {
				if lineY1 >= 0 && lineY1 < s.height {
					img.Set(x+offsetX, lineY1, lineColor)
				}
				if lineY2 >= 0 && lineY2 < s.height {
					img.Set(x+offsetX, lineY2, lineColor)
				}
			}
		}
	}

	// Draw profile name
	nameColor := color.Gray{Y: brightness}
	drawX := nameX + offsetX
	for _, ch := range profileName {
		glyph := glyphs.GetGlyph(font, ch)
		if glyph != nil {
			glyphs.DrawGlyph(img, glyph, drawX, nameY, nameColor)
			drawX += glyph.Width + 1
		}
	}

	return img
}

// renderWinampSlogan renders Winamp's iconic slogan on two lines
// "It really whips" / "the llama's ass!"
func (s *SplashRenderer) renderWinampSlogan(img *image.Gray, progress float64) {
	font := glyphs.Font5x7
	line1 := "It really whips"
	line2 := "the llama's ass!"

	lineSpacing := 2
	totalHeight := font.GlyphHeight*2 + lineSpacing

	// Center both lines vertically
	startY := (s.height - totalHeight) / 2

	line1Width := glyphs.MeasureText(line1, font)
	line2Width := glyphs.MeasureText(line2, font)
	line1X := (s.width - line1Width) / 2
	line2X := (s.width - line2Width) / 2
	line1Y := startY
	line2Y := startY + font.GlyphHeight + lineSpacing

	// Phase 1 (0-0.2): Slide in from right
	// Phase 2 (0.2-0.8): Hold
	// Phase 3 (0.8-1.0): Fade out

	var brightness uint8
	var offsetX int

	if progress < 0.2 {
		slideProgress := progress / 0.2
		offsetX = int(float64(s.width) * (1 - slideProgress))
		brightness = 255
	} else if progress < 0.8 {
		offsetX = 0
		brightness = 255
	} else {
		fadeProgress := (progress - 0.8) / 0.2
		offsetX = 0
		brightness = uint8(255 * (1 - fadeProgress))
	}

	// Draw decorative lines
	if brightness > 50 {
		lineColor := color.Gray{Y: brightness / 2}
		decorY1 := line1Y - 4
		decorY2 := line2Y + font.GlyphHeight + 3

		// Use wider of the two lines for decoration width
		maxWidth := line1Width
		if line2Width > maxWidth {
			maxWidth = line2Width
		}
		lineStart := (s.width-maxWidth)/2 - 10
		lineEnd := (s.width+maxWidth)/2 + 10
		if lineStart < 5 {
			lineStart = 5
		}
		if lineEnd > s.width-5 {
			lineEnd = s.width - 5
		}

		for x := lineStart; x < lineEnd; x++ {
			if x+offsetX >= 0 && x+offsetX < s.width {
				if decorY1 >= 0 && decorY1 < s.height {
					img.Set(x+offsetX, decorY1, lineColor)
				}
				if decorY2 >= 0 && decorY2 < s.height {
					img.Set(x+offsetX, decorY2, lineColor)
				}
			}
		}
	}

	// Draw both lines
	textColor := color.Gray{Y: brightness}

	// Line 1
	drawX := line1X + offsetX
	for _, ch := range line1 {
		glyph := glyphs.GetGlyph(font, ch)
		if glyph != nil {
			glyphs.DrawGlyph(img, glyph, drawX, line1Y, textColor)
			drawX += glyph.Width + 1
		}
	}

	// Line 2
	drawX = line2X + offsetX
	for _, ch := range line2 {
		glyph := glyphs.GetGlyph(font, ch)
		if glyph != nil {
			glyphs.DrawGlyph(img, glyph, drawX, line2Y, textColor)
			drawX += glyph.Width + 1
		}
	}
}

// renderExitFrame renders a single frame of the exit animation
// Animation: "BYE!" text fades out with a wave effect
func (s *SplashRenderer) renderExitFrame(progress float64) *image.Gray {
	img := image.NewGray(image.Rect(0, 0, s.width, s.height))

	exitText := "BYE!"
	font := glyphs.Font5x7

	textWidth := glyphs.MeasureText(exitText, font)
	textHeight := font.GlyphHeight
	textX := (s.width - textWidth) / 2
	textY := (s.height - textHeight) / 2

	// Phase 1 (0-0.3): Full display
	// Phase 2 (0.3-1.0): Wave fade out from left to right

	if progress < 0.3 {
		// Full display
		glyphs.DrawText(img, exitText, textX, textY, font, color.Gray{Y: 255})
	} else {
		// Wave fade out
		fadeProgress := (progress - 0.3) / 0.7
		waveX := int(float64(textWidth+20) * fadeProgress)

		currentX := textX
		for _, ch := range exitText {
			glyph := glyphs.GetGlyph(font, ch)
			if glyph == nil {
				continue
			}

			// Calculate brightness based on distance from wave front
			charCenterX := currentX + glyph.Width/2
			distFromWave := charCenterX - (textX + waveX)

			var brightness uint8
			if distFromWave < -10 {
				brightness = 0
			} else if distFromWave < 0 {
				brightness = uint8(255 * float64(distFromWave+10) / 10)
			} else {
				brightness = 255
			}

			if brightness > 0 {
				glyphs.DrawGlyph(img, glyph, currentX, textY, color.Gray{Y: brightness})
			}

			currentX += glyph.Width + 1
		}
	}

	// Draw a small wave/hand animation at the end
	if progress > 0.4 && progress < 0.9 {
		waveProgress := (progress - 0.4) / 0.5
		waveOffset := int(3 * waveProgress)

		// Simple hand wave (just a few pixels moving)
		handX := textX + textWidth + 10
		handY := textY + textHeight/2

		// Draw a small "hand" that waves
		for dy := -2; dy <= 2; dy++ {
			yOffset := 0
			if dy == -2 || dy == 2 {
				yOffset = waveOffset % 2
			}
			if handX < s.width && handY+dy+yOffset >= 0 && handY+dy+yOffset < s.height {
				brightness := uint8(200 - int(progress*200))
				img.Set(handX, handY+dy+yOffset, color.Gray{Y: brightness})
			}
		}
	}

	return img
}

// drawTextClipped draws text with horizontal clipping (for reveal animation)
func (s *SplashRenderer) drawTextClipped(img *image.Gray, text string, x, y int, font *glyphs.GlyphSet, clipLeft, clipRight int) {
	currentX := x
	for _, ch := range text {
		glyph := glyphs.GetGlyph(font, ch)
		if glyph == nil {
			continue
		}

		// Draw glyph with clipping
		for row := 0; row < glyph.Height && row < len(glyph.Data); row++ {
			for col := 0; col < glyph.Width && col < len(glyph.Data[row]); col++ {
				px := currentX + col
				py := y + row

				// Clip horizontally
				if px < clipLeft || px >= clipRight {
					continue
				}

				if glyph.Data[row][col] {
					img.Set(px, py, color.Gray{Y: 255})
				}
			}
		}

		currentX += glyph.Width + 1
	}
}

// sendFrame sends an image frame to the display
func (s *SplashRenderer) sendFrame(img *image.Gray) error {
	// Convert to bitmap data
	bitmapData, err := bitmap.ImageToBytes(img, s.width, s.height, nil)
	if err != nil {
		return err
	}

	// Send to display
	return s.client.SendScreenData(EventName, bitmapData)
}

// abs returns absolute value of an integer
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
