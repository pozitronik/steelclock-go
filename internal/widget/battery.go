package widget

import (
	"fmt"
	"image"
	"image/color"
	"sync"
	"time"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/bitmap/glyphs"
	"github.com/pozitronik/steelclock-go/internal/config"
	"golang.org/x/image/font"
)

// BatteryStatus represents the current battery state
type BatteryStatus struct {
	Percentage    int  // 0-100
	IsCharging    bool // Currently charging
	IsPluggedIn   bool // AC power connected
	HasBattery    bool // Battery present in system
	IsEconomyMode bool // Power saver/economy mode active
	TimeToEmpty   int  // Minutes remaining (0 if unknown)
	TimeToFull    int  // Minutes to full charge (0 if unknown)
}

// BatteryWidget displays battery level and power status
type BatteryWidget struct {
	*BaseWidget

	// Display settings
	displayMode    string // "icon", "text", "bar", "gauge", "graph"
	showPercentage bool
	showStatus     bool
	orientation    string // "horizontal", "vertical"
	chargingBlink  bool

	// Icon set for status indicators (selected based on widget size)
	iconSet *glyphs.GlyphSet

	// Thresholds
	lowThreshold      int
	criticalThreshold int

	// Colors
	colorNormal     uint8
	colorLow        uint8
	colorCritical   uint8
	colorCharging   uint8
	colorBackground uint8
	colorBorder     uint8

	// Graph mode
	graphHistory int
	history      *RingBuffer[int]

	// Font for text rendering
	fontSize   int
	fontName   string
	horizAlign string
	vertAlign  string
	fontFace   font.Face
	padding    int

	// Gauge settings
	gaugeColor       uint8
	gaugeNeedleColor uint8
	gaugeShowTicks   bool
	gaugeTicksColor  uint8

	// Bar settings
	barDirection string
	barBorder    bool

	// Graph settings
	graphFilled bool
	fillColor   uint8

	// Current state
	currentStatus BatteryStatus
	hasData       bool
	mu            sync.RWMutex
}

// NewBatteryWidget creates a new battery widget
func NewBatteryWidget(cfg config.WidgetConfig) (*BatteryWidget, error) {
	base := NewBaseWidget(cfg)
	helper := NewConfigHelper(cfg)

	// Display mode from widget-level Mode (like CPU widget)
	displayMode := "icon"
	if cfg.Mode != "" {
		displayMode = cfg.Mode
	}

	// Boolean settings with defaults
	showPercentage := true
	showStatus := true
	chargingBlink := false // Default false - less distracting

	if cfg.Battery != nil {
		if cfg.Battery.ShowPercentage != nil {
			showPercentage = *cfg.Battery.ShowPercentage
		}
		if cfg.Battery.ShowStatus != nil {
			showStatus = *cfg.Battery.ShowStatus
		}
		if cfg.Battery.ChargingBlink != nil {
			chargingBlink = *cfg.Battery.ChargingBlink
		}
	}

	// Orientation from battery config
	orientation := "horizontal"
	if cfg.Battery != nil && cfg.Battery.Orientation != "" {
		orientation = cfg.Battery.Orientation
	}

	// Thresholds
	lowThreshold := 20
	criticalThreshold := 10
	if cfg.Battery != nil {
		if cfg.Battery.LowThreshold > 0 {
			lowThreshold = cfg.Battery.LowThreshold
		}
		if cfg.Battery.CriticalThreshold > 0 {
			criticalThreshold = cfg.Battery.CriticalThreshold
		}
	}

	// Colors - use pointers to allow 0 (black)
	colorNormal := uint8(255)
	colorLow := uint8(200)
	colorCritical := uint8(150)
	colorCharging := uint8(255)
	colorBackground := uint8(0)
	colorBorder := uint8(255)

	if cfg.Battery != nil && cfg.Battery.Colors != nil {
		if cfg.Battery.Colors.Normal != nil {
			colorNormal = uint8(*cfg.Battery.Colors.Normal)
		}
		if cfg.Battery.Colors.Low != nil {
			colorLow = uint8(*cfg.Battery.Colors.Low)
		}
		if cfg.Battery.Colors.Critical != nil {
			colorCritical = uint8(*cfg.Battery.Colors.Critical)
		}
		if cfg.Battery.Colors.Charging != nil {
			colorCharging = uint8(*cfg.Battery.Colors.Charging)
		}
		if cfg.Battery.Colors.Background != nil {
			colorBackground = uint8(*cfg.Battery.Colors.Background)
		}
		if cfg.Battery.Colors.Border != nil {
			colorBorder = uint8(*cfg.Battery.Colors.Border)
		}
	}

	// Graph history from graph config (shared)
	graphHistory := 60
	if cfg.Graph != nil && cfg.Graph.History > 0 {
		graphHistory = cfg.Graph.History
	}

	// Get common settings from helper
	textSettings := helper.GetTextSettings()
	padding := helper.GetPadding()
	barSettings := helper.GetBarSettings()
	graphSettings := helper.GetGraphSettings()
	gaugeSettings := helper.GetGaugeSettings()
	fillColor := helper.GetFillColorForMode(displayMode)

	// Load font for text mode
	fontFace, err := helper.LoadFontForTextMode(displayMode)
	if err != nil {
		return nil, fmt.Errorf("failed to load font: %w", err)
	}

	// Select icon set based on widget dimensions
	// For vertical orientation, width is the limiting factor; for horizontal, height is
	iconDimension := cfg.Position.H
	if orientation == "vertical" {
		iconDimension = cfg.Position.W
	}
	iconSet := selectBatteryIconSet(iconDimension)

	return &BatteryWidget{
		BaseWidget:        base,
		displayMode:       displayMode,
		showPercentage:    showPercentage,
		showStatus:        showStatus,
		orientation:       orientation,
		chargingBlink:     chargingBlink,
		iconSet:           iconSet,
		lowThreshold:      lowThreshold,
		criticalThreshold: criticalThreshold,
		colorNormal:       colorNormal,
		colorLow:          colorLow,
		colorCritical:     colorCritical,
		colorCharging:     colorCharging,
		colorBackground:   colorBackground,
		colorBorder:       colorBorder,
		graphHistory:      graphHistory,
		history:           NewRingBuffer[int](graphHistory),
		fontSize:          textSettings.FontSize,
		fontName:          textSettings.FontName,
		horizAlign:        textSettings.HorizAlign,
		vertAlign:         textSettings.VertAlign,
		fontFace:          fontFace,
		padding:           padding,
		gaugeColor:        uint8(gaugeSettings.ArcColor),
		gaugeNeedleColor:  uint8(gaugeSettings.NeedleColor),
		gaugeShowTicks:    gaugeSettings.ShowTicks,
		gaugeTicksColor:   uint8(gaugeSettings.TicksColor),
		barDirection:      barSettings.Direction,
		barBorder:         barSettings.Border,
		graphFilled:       graphSettings.Filled,
		fillColor:         uint8(fillColor),
	}, nil
}

// Update reads current battery status
func (w *BatteryWidget) Update() error {
	status, err := getBatteryStatus()
	if err != nil {
		return err
	}

	w.mu.Lock()
	w.currentStatus = status
	w.hasData = true
	// Add to history for graph mode
	w.history.Push(status.Percentage)
	w.mu.Unlock()

	return nil
}

// Render renders the battery widget
func (w *BatteryWidget) Render() (image.Image, error) {
	pos := w.GetPosition()
	style := w.GetStyle()

	img := bitmap.NewGrayscaleImage(pos.W, pos.H, w.GetRenderBackgroundColor())

	if style.Border >= 0 {
		bitmap.DrawBorder(img, uint8(style.Border))
	}

	w.mu.RLock()
	status := w.currentStatus
	hasData := w.hasData
	w.mu.RUnlock()

	if !hasData {
		bitmap.SmartDrawAlignedText(img, "...", w.fontFace, w.fontName, "center", "center", w.padding)
		return img, nil
	}

	if !status.HasBattery {
		bitmap.SmartDrawAlignedText(img, "No Battery", w.fontFace, w.fontName, "center", "center", w.padding)
		return img, nil
	}

	switch w.displayMode {
	case "text":
		w.renderText(img, status)
	case "bar":
		w.renderBar(img, status)
	case "gauge":
		w.renderGauge(img, status)
	case "graph":
		w.renderGraph(img, status)
	case "battery":
		w.renderBattery(img, status) // Progressbar in battery shape
	default: // "icon" - compact tray-style icon
		w.renderIcon(img, status)
	}

	return img, nil
}

// getColorForLevel returns the appropriate color based on battery level
func (w *BatteryWidget) getColorForLevel(percentage int) uint8 {
	if percentage <= w.criticalThreshold {
		return w.colorCritical
	}
	if percentage <= w.lowThreshold {
		return w.colorLow
	}
	return w.colorNormal
}

// selectBatteryIconSet returns the appropriate icon set based on widget height
func selectBatteryIconSet(height int) *glyphs.GlyphSet {
	if height >= 22 {
		return glyphs.BatteryIcons16x16
	} else if height >= 14 {
		return glyphs.BatteryIcons12x12
	}
	return glyphs.BatteryIcons8x8
}

// getStatusIconName returns the icon name to display based on battery status
// Priority: charging > economy > ac_power (only show one icon)
func (w *BatteryWidget) getStatusIconName(status BatteryStatus) string {
	if status.IsCharging {
		return "charging"
	}
	if status.IsEconomyMode {
		return "economy"
	}
	if status.IsPluggedIn {
		return "ac_power"
	}
	return ""
}

// drawStatusIcon draws a status icon with 1px black border for visibility
func (w *BatteryWidget) drawStatusIcon(img *image.Gray, x, y int, status BatteryStatus) {
	iconName := w.getStatusIconName(status)
	if iconName == "" || w.iconSet == nil {
		return
	}

	// Apply blink effect for charging indicator
	if iconName == "charging" && w.chargingBlink && time.Now().Second()%2 != 0 {
		return
	}

	icon := glyphs.GetIcon(w.iconSet, iconName)
	if icon == nil {
		return
	}

	black := color.Gray{Y: 0}
	white := color.Gray{Y: 255}

	// Draw black border (1px offset in all directions)
	glyphs.DrawGlyph(img, icon, x-1, y, black)
	glyphs.DrawGlyph(img, icon, x+1, y, black)
	glyphs.DrawGlyph(img, icon, x, y-1, black)
	glyphs.DrawGlyph(img, icon, x, y+1, black)
	// Diagonal borders for better visibility
	glyphs.DrawGlyph(img, icon, x-1, y-1, black)
	glyphs.DrawGlyph(img, icon, x+1, y-1, black)
	glyphs.DrawGlyph(img, icon, x-1, y+1, black)
	glyphs.DrawGlyph(img, icon, x+1, y+1, black)
	// Draw white icon on top
	glyphs.DrawGlyph(img, icon, x, y, white)
}

// getIconSize returns the width and height of the current icon set
func (w *BatteryWidget) getIconSize() (int, int) {
	if w.iconSet == nil {
		return 8, 8
	}
	return w.iconSet.GlyphWidth, w.iconSet.GlyphHeight
}

// drawFilledRect draws a filled rectangle
func (w *BatteryWidget) drawFilledRect(img *image.Gray, x, y, width, height int, c color.Gray) {
	bounds := img.Bounds()
	for py := y; py < y+height; py++ {
		if py < bounds.Min.Y || py >= bounds.Max.Y {
			continue
		}
		for px := x; px < x+width; px++ {
			if px >= bounds.Min.X && px < bounds.Max.X {
				img.SetGray(px, py, c)
			}
		}
	}
}

// renderText renders battery as text
func (w *BatteryWidget) renderText(img *image.Gray, status BatteryStatus) {
	text := fmt.Sprintf("%d%%", status.Percentage)
	if w.showStatus && status.IsCharging {
		text += " CHG"
	} else if w.showStatus && status.IsPluggedIn {
		text += " AC"
	}
	bitmap.SmartDrawAlignedText(img, text, w.fontFace, w.fontName, w.horizAlign, w.vertAlign, w.padding)
}

// renderBar renders battery as a progress bar
func (w *BatteryWidget) renderBar(img *image.Gray, status BatteryStatus) {
	pos := w.GetPosition()
	fillColor := w.getColorForLevel(status.Percentage)

	barX := w.padding
	barY := w.padding
	barW := pos.W - 2*w.padding
	barH := pos.H - 2*w.padding

	// Draw border if enabled
	if w.barBorder {
		bitmap.DrawRectangle(img, barX, barY, barW, barH, w.colorBorder)
		barX++
		barY++
		barW -= 2
		barH -= 2
	}

	// Calculate fill
	fillAmount := float64(status.Percentage) / 100.0

	if w.orientation == "vertical" || w.barDirection == "up" || w.barDirection == "down" {
		fillH := int(float64(barH) * fillAmount)
		if w.barDirection == "down" {
			w.drawFilledRect(img, barX, barY, barW, fillH, color.Gray{Y: fillColor})
		} else {
			w.drawFilledRect(img, barX, barY+barH-fillH, barW, fillH, color.Gray{Y: fillColor})
		}
	} else {
		fillW := int(float64(barW) * fillAmount)
		if w.barDirection == "right" {
			w.drawFilledRect(img, barX+barW-fillW, barY, fillW, barH, color.Gray{Y: fillColor})
		} else {
			w.drawFilledRect(img, barX, barY, fillW, barH, color.Gray{Y: fillColor})
		}
	}

	// Draw percentage text if enabled
	if w.showPercentage {
		text := fmt.Sprintf("%d%%", status.Percentage)
		bitmap.SmartDrawAlignedText(img, text, w.fontFace, w.fontName, "center", "center", 0)
	}

	// Draw status icon (charging, economy, or AC)
	if w.showStatus {
		iconW, _ := w.getIconSize()
		w.drawStatusIcon(img, pos.W-iconW-2, 2, status)
	}
}

// renderGauge renders battery as a semicircular gauge
func (w *BatteryWidget) renderGauge(img *image.Gray, status BatteryStatus) {
	pos := w.GetPosition()

	// Use the existing DrawGauge function
	bitmap.DrawGauge(img, 0, 0, pos.W, pos.H, float64(status.Percentage),
		w.gaugeColor, w.gaugeNeedleColor, w.gaugeShowTicks, w.gaugeTicksColor)

	// Draw percentage text
	if w.showPercentage {
		text := fmt.Sprintf("%d%%", status.Percentage)
		bitmap.SmartDrawAlignedText(img, text, w.fontFace, w.fontName, "center", "top", w.padding)
	}

	// Draw status icon (charging, economy, or AC)
	if w.showStatus {
		iconW, _ := w.getIconSize()
		w.drawStatusIcon(img, pos.W-iconW-2, 2, status)
	}
}

// renderGraph renders battery history as a graph
func (w *BatteryWidget) renderGraph(img *image.Gray, status BatteryStatus) {
	pos := w.GetPosition()

	// Get history data and convert to float64
	w.mu.RLock()
	historyData := w.history.ToSlice()
	w.mu.RUnlock()

	// Convert int slice to float64 slice for DrawGraph
	floatHistory := make([]float64, len(historyData))
	for i, v := range historyData {
		floatHistory[i] = float64(v)
	}

	graphX := w.padding
	graphY := w.padding
	graphW := pos.W - 2*w.padding
	graphH := pos.H - 2*w.padding

	// Draw border
	bitmap.DrawRectangle(img, graphX, graphY, graphW, graphH, w.colorBorder)

	// Use existing DrawGraph function (values are 0-100 for percentage)
	bitmap.DrawGraph(img, graphX+1, graphY+1, graphW-2, graphH-2, floatHistory, w.graphHistory, w.fillColor, w.graphFilled)

	// Draw current percentage
	if w.showPercentage {
		text := fmt.Sprintf("%d%%", status.Percentage)
		bitmap.SmartDrawAlignedText(img, text, w.fontFace, w.fontName, "right", "top", w.padding+2)
	}

	// Draw status icon (charging, economy, or AC) in top-left
	if w.showStatus {
		w.drawStatusIcon(img, w.padding+2, w.padding+2, status)
	}
}

// renderIcon renders a compact tray-style battery icon (like Windows system tray)
func (w *BatteryWidget) renderIcon(img *image.Gray, status BatteryStatus) {
	pos := w.GetPosition()

	// Compact icon: small battery shape with optional percentage text next to it
	// Center the icon in the widget area
	iconW := 16
	iconH := 10
	nubW := 2

	if w.orientation == "vertical" {
		iconW = 10
		iconH = 16
	}

	// Center position
	startX := (pos.W - iconW - nubW) / 2
	startY := (pos.H - iconH) / 2

	if w.orientation == "vertical" {
		w.drawCompactBatteryVertical(img, startX, startY, iconW, iconH, status)
	} else {
		w.drawCompactBatteryHorizontal(img, startX, startY, iconW, iconH, status)
	}
}

// drawCompactBatteryHorizontal draws a small horizontal battery icon
func (w *BatteryWidget) drawCompactBatteryHorizontal(img *image.Gray, x, y, width, height int, status BatteryStatus) {
	nubW := 2
	nubH := height / 2

	// Battery body outline
	bitmap.DrawRectangle(img, x, y, width, height, w.colorBorder)

	// Positive terminal nub
	nubX := x + width
	nubY := y + (height-nubH)/2
	w.drawFilledRect(img, nubX, nubY, nubW, nubH, color.Gray{Y: w.colorBorder})

	// Fill level
	fillMargin := 1
	fillX := x + fillMargin
	fillY := y + fillMargin
	fillMaxW := width - 2*fillMargin
	fillH := height - 2*fillMargin
	fillW := int(float64(fillMaxW) * float64(status.Percentage) / 100.0)

	fillColor := w.getColorForLevel(status.Percentage)
	if fillW > 0 {
		w.drawFilledRect(img, fillX, fillY, fillW, fillH, color.Gray{Y: fillColor})
	}

	// Draw status icon inside the battery shape
	if w.showStatus {
		iconW, iconH := w.getIconSize()
		iconX := x + (width-iconW)/2
		iconY := y + (height-iconH)/2
		w.drawStatusIcon(img, iconX, iconY, status)
	}
}

// drawCompactBatteryVertical draws a small vertical battery icon
func (w *BatteryWidget) drawCompactBatteryVertical(img *image.Gray, x, y, width, height int, status BatteryStatus) {
	nubW := width / 2
	nubH := 2

	// Positive terminal nub at top
	nubX := x + (width-nubW)/2
	nubY := y
	w.drawFilledRect(img, nubX, nubY, nubW, nubH, color.Gray{Y: w.colorBorder})

	// Battery body outline below nub
	bodyY := y + nubH
	bodyH := height - nubH
	bitmap.DrawRectangle(img, x, bodyY, width, bodyH, w.colorBorder)

	// Fill level (from bottom)
	fillMargin := 1
	fillX := x + fillMargin
	fillW := width - 2*fillMargin
	fillMaxH := bodyH - 2*fillMargin
	fillH := int(float64(fillMaxH) * float64(status.Percentage) / 100.0)
	fillY := bodyY + fillMargin + fillMaxH - fillH

	fillColor := w.getColorForLevel(status.Percentage)
	if fillH > 0 {
		w.drawFilledRect(img, fillX, fillY, fillW, fillH, color.Gray{Y: fillColor})
	}

	// Draw status icon inside the battery shape
	if w.showStatus {
		iconW, iconH := w.getIconSize()
		iconX := x + (width-iconW)/2
		iconY := bodyY + (bodyH-iconH)/2
		w.drawStatusIcon(img, iconX, iconY, status)
	}
}

// renderBattery renders battery as a large progressbar in battery shape
func (w *BatteryWidget) renderBattery(img *image.Gray, status BatteryStatus) {
	pos := w.GetPosition()

	if w.orientation == "vertical" {
		w.renderBatteryVertical(img, status, pos)
	} else {
		w.renderBatteryHorizontal(img, status, pos)
	}
}

// renderBatteryHorizontal draws horizontal battery progressbar
func (w *BatteryWidget) renderBatteryHorizontal(img *image.Gray, status BatteryStatus, pos config.PositionConfig) {
	// Battery dimensions - leave room for the positive terminal nub
	nubW := 4
	batteryX := w.padding
	batteryY := w.padding
	batteryW := pos.W - w.padding*2 - nubW - 1
	batteryH := pos.H - w.padding*2

	if batteryW < 20 {
		batteryW = 20
	}
	if batteryH < 10 {
		batteryH = 10
	}

	// Draw battery body outline
	bitmap.DrawRectangle(img, batteryX, batteryY, batteryW, batteryH, w.colorBorder)

	// Draw positive terminal (nub on the right)
	nubH := batteryH / 3
	if nubH < 4 {
		nubH = 4
	}
	nubX := batteryX + batteryW
	nubY := batteryY + (batteryH-nubH)/2
	w.drawFilledRect(img, nubX, nubY, nubW, nubH, color.Gray{Y: w.colorBorder})

	// Calculate fill dimensions (inside the battery body)
	fillMargin := 2
	fillX := batteryX + fillMargin
	fillY := batteryY + fillMargin
	fillMaxW := batteryW - 2*fillMargin
	fillH := batteryH - 2*fillMargin
	fillW := int(float64(fillMaxW) * float64(status.Percentage) / 100.0)

	fillColor := w.getColorForLevel(status.Percentage)
	if fillW > 0 {
		w.drawFilledRect(img, fillX, fillY, fillW, fillH, color.Gray{Y: fillColor})
	}

	// Draw status icon in top-left corner
	if w.showStatus {
		w.drawStatusIcon(img, batteryX+2, batteryY+2, status)
	}
}

// renderBatteryVertical draws vertical battery progressbar
func (w *BatteryWidget) renderBatteryVertical(img *image.Gray, status BatteryStatus, pos config.PositionConfig) {
	// Battery dimensions - leave room for the positive terminal nub at top
	nubH := 4
	batteryX := w.padding
	batteryY := w.padding + nubH + 1
	batteryW := pos.W - w.padding*2
	batteryH := pos.H - w.padding*2 - nubH - 1

	if batteryW < 10 {
		batteryW = 10
	}
	if batteryH < 20 {
		batteryH = 20
	}

	// Draw battery body outline
	bitmap.DrawRectangle(img, batteryX, batteryY, batteryW, batteryH, w.colorBorder)

	// Draw positive terminal (nub at top)
	nubW := batteryW / 3
	if nubW < 4 {
		nubW = 4
	}
	nubX := batteryX + (batteryW-nubW)/2
	nubY := w.padding
	w.drawFilledRect(img, nubX, nubY, nubW, nubH, color.Gray{Y: w.colorBorder})

	// Calculate fill dimensions
	fillMargin := 2
	fillX := batteryX + fillMargin
	fillMaxH := batteryH - 2*fillMargin
	fillW := batteryW - 2*fillMargin
	fillH := int(float64(fillMaxH) * float64(status.Percentage) / 100.0)
	fillY := batteryY + fillMargin + fillMaxH - fillH

	fillColor := w.getColorForLevel(status.Percentage)
	if fillH > 0 {
		w.drawFilledRect(img, fillX, fillY, fillW, fillH, color.Gray{Y: fillColor})
	}

	// Draw status icon at bottom-center
	if w.showStatus {
		w.drawStatusIcon(img, batteryX+2, batteryY+2, status)
	}
}

// Stop cleans up resources
func (w *BatteryWidget) Stop() {
	// Nothing to clean up
}
