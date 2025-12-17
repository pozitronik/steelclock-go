package screenmirror

import (
	"image"
	"image/color"
	imgdraw "image/draw"

	"golang.org/x/image/draw"
)

// ScaleMode defines how the captured image is scaled to fit the widget.
type ScaleMode string

const (
	// ScaleModeFit preserves aspect ratio, adds letterboxing if needed.
	ScaleModeFit ScaleMode = "fit"
	// ScaleModeStretch stretches to fill, may distort aspect ratio.
	ScaleModeStretch ScaleMode = "stretch"
	// ScaleModeCrop preserves aspect ratio, crops edges to fill.
	ScaleModeCrop ScaleMode = "crop"
)

// ScaleImage scales the source image to fit the target dimensions using the specified mode.
// Returns a new grayscale image suitable for the OLED display.
func ScaleImage(src image.Image, targetWidth, targetHeight int, mode ScaleMode) *image.Gray {
	if src == nil {
		return image.NewGray(image.Rect(0, 0, targetWidth, targetHeight))
	}

	srcBounds := src.Bounds()
	srcWidth := srcBounds.Dx()
	srcHeight := srcBounds.Dy()

	if srcWidth == 0 || srcHeight == 0 {
		return image.NewGray(image.Rect(0, 0, targetWidth, targetHeight))
	}

	// Calculate destination rectangle based on scale mode
	var dstRect image.Rectangle
	var srcRect image.Rectangle

	switch mode {
	case ScaleModeStretch:
		// Stretch to fill entire target
		dstRect = image.Rect(0, 0, targetWidth, targetHeight)
		srcRect = srcBounds

	case ScaleModeCrop:
		// Fill target, crop source to maintain aspect ratio
		dstRect = image.Rect(0, 0, targetWidth, targetHeight)
		srcRect = calculateCropRect(srcWidth, srcHeight, targetWidth, targetHeight)

	case ScaleModeFit:
		fallthrough
	default:
		// Fit within target, maintain aspect ratio (letterbox)
		dstRect = calculateFitRect(srcWidth, srcHeight, targetWidth, targetHeight)
		srcRect = srcBounds
	}

	// Create intermediate RGBA image for scaling
	scaled := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))

	// Fill with black background (for letterboxing)
	imgdraw.Draw(scaled, scaled.Bounds(), image.NewUniform(color.Black), image.Point{}, imgdraw.Src)

	// Scale using high-quality interpolation
	scaler := draw.CatmullRom // Smooth interpolation
	scaler.Scale(scaled, dstRect, src, srcRect, draw.Over, nil)

	// Convert to grayscale
	gray := image.NewGray(image.Rect(0, 0, targetWidth, targetHeight))
	for y := 0; y < targetHeight; y++ {
		for x := 0; x < targetWidth; x++ {
			r, g, b, _ := scaled.At(x, y).RGBA()
			// Convert to grayscale using luminance formula
			lum := uint8((299*r + 587*g + 114*b) / 1000 >> 8)
			gray.SetGray(x, y, color.Gray{Y: lum})
		}
	}

	return gray
}

// calculateFitRect calculates the destination rectangle for fit mode (letterbox).
func calculateFitRect(srcWidth, srcHeight, targetWidth, targetHeight int) image.Rectangle {
	srcAspect := float64(srcWidth) / float64(srcHeight)
	targetAspect := float64(targetWidth) / float64(targetHeight)

	var dstWidth, dstHeight int

	if srcAspect > targetAspect {
		// Source is wider - fit to width, letterbox top/bottom
		dstWidth = targetWidth
		dstHeight = int(float64(targetWidth) / srcAspect)
	} else {
		// Source is taller - fit to height, letterbox left/right
		dstHeight = targetHeight
		dstWidth = int(float64(targetHeight) * srcAspect)
	}

	// Center in target
	x := (targetWidth - dstWidth) / 2
	y := (targetHeight - dstHeight) / 2

	return image.Rect(x, y, x+dstWidth, y+dstHeight)
}

// calculateCropRect calculates the source rectangle for crop mode (fill and crop).
func calculateCropRect(srcWidth, srcHeight, targetWidth, targetHeight int) image.Rectangle {
	srcAspect := float64(srcWidth) / float64(srcHeight)
	targetAspect := float64(targetWidth) / float64(targetHeight)

	var cropWidth, cropHeight int

	if srcAspect > targetAspect {
		// Source is wider - crop horizontally
		cropHeight = srcHeight
		cropWidth = int(float64(srcHeight) * targetAspect)
	} else {
		// Source is taller - crop vertically
		cropWidth = srcWidth
		cropHeight = int(float64(srcWidth) / targetAspect)
	}

	// Center crop region
	x := (srcWidth - cropWidth) / 2
	y := (srcHeight - cropHeight) / 2

	return image.Rect(x, y, x+cropWidth, y+cropHeight)
}
