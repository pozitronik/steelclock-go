package widget

import (
	"fmt"
	"image"
	"image/color"
	"regexp"
	"strings"
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

// indicatorState tracks display settings and notify state for a power indicator
type indicatorState struct {
	mode           string        // "always", "never", "notify", "blink", "notify_blink"
	notifyDuration time.Duration // duration to show in notify modes
	notifyUntil    time.Time     // when to stop showing in notify modes
}

// BatteryWidget displays battery level and power status
type BatteryWidget struct {
	*BaseWidget

	// Display settings
	displayMode    string // "battery", "text", "bar", "gauge", "graph"
	showPercentage bool
	orientation    string // "horizontal", "vertical"

	// Power status indicator settings
	chargingState indicatorState
	pluggedState  indicatorState
	economyState  indicatorState

	// Previous status for change detection (notify modes)
	prevCharging bool
	prevPlugged  bool
	prevEconomy  bool

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
	textFormat string // Format string with tokens like {percent}, {status}, etc.

	// Gauge settings
	gaugeColor       uint8
	gaugeNeedleColor uint8
	gaugeShowTicks   bool
	gaugeTicksColor  uint8

	// Bar settings
	barDirection string
	barBorder    bool

	// Graph settings
	fillColor int // -1 = no fill, 0-255 = fill color
	lineColor int // 0-255 = line color

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

	if cfg.Battery != nil {
		if cfg.Battery.ShowPercentage != nil {
			showPercentage = *cfg.Battery.ShowPercentage
		}
	}

	// Power status indicator settings with defaults
	notifyDuration := 60 * time.Second
	chargingMode := "always"
	pluggedMode := "always"
	economyMode := "blink"

	if cfg.PowerStatus != nil {
		if cfg.PowerStatus.NotifyDuration > 0 {
			notifyDuration = time.Duration(cfg.PowerStatus.NotifyDuration) * time.Second
		}
		if cfg.PowerStatus.ShowCharging != "" {
			chargingMode = cfg.PowerStatus.ShowCharging
		}
		if cfg.PowerStatus.ShowPlugged != "" {
			pluggedMode = cfg.PowerStatus.ShowPlugged
		}
		if cfg.PowerStatus.ShowEconomy != "" {
			economyMode = cfg.PowerStatus.ShowEconomy
		}
	}

	chargingState := indicatorState{mode: chargingMode, notifyDuration: notifyDuration}
	pluggedState := indicatorState{mode: pluggedMode, notifyDuration: notifyDuration}
	economyState := indicatorState{mode: economyMode, notifyDuration: notifyDuration}

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

	// Text format with tokens (default: "{percent}%")
	textFormat := "{percent}%"
	if cfg.Text != nil && cfg.Text.Format != "" {
		textFormat = cfg.Text.Format
	}

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
		orientation:       orientation,
		chargingState:     chargingState,
		pluggedState:      pluggedState,
		economyState:      economyState,
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
		textFormat:        textFormat,
		gaugeColor:        uint8(gaugeSettings.ArcColor),
		gaugeNeedleColor:  uint8(gaugeSettings.NeedleColor),
		gaugeShowTicks:    gaugeSettings.ShowTicks,
		gaugeTicksColor:   uint8(gaugeSettings.TicksColor),
		barDirection:      barSettings.Direction,
		barBorder:         barSettings.Border,
		fillColor:         graphSettings.FillColor,
		lineColor:         graphSettings.LineColor,
	}, nil
}

// Update reads current battery status
func (w *BatteryWidget) Update() error {
	status, err := getBatteryStatus()
	if err != nil {
		return err
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	// Detect status changes for notify modes
	now := time.Now()

	// Charging status changed to active
	if status.IsCharging && !w.prevCharging {
		if w.chargingState.mode == "notify" || w.chargingState.mode == "notify_blink" {
			w.chargingState.notifyUntil = now.Add(w.chargingState.notifyDuration)
		}
	}
	w.prevCharging = status.IsCharging

	// Plugged status changed to active
	if status.IsPluggedIn && !w.prevPlugged {
		if w.pluggedState.mode == "notify" || w.pluggedState.mode == "notify_blink" {
			w.pluggedState.notifyUntil = now.Add(w.pluggedState.notifyDuration)
		}
	}
	w.prevPlugged = status.IsPluggedIn

	// Economy status changed to active
	if status.IsEconomyMode && !w.prevEconomy {
		if w.economyState.mode == "notify" || w.economyState.mode == "notify_blink" {
			w.economyState.notifyUntil = now.Add(w.economyState.notifyDuration)
		}
	}
	w.prevEconomy = status.IsEconomyMode

	w.currentStatus = status
	w.hasData = true
	// Add to history for graph mode
	w.history.Push(status.Percentage)

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
	default: // "battery" - progressbar in battery shape
		w.renderBattery(img, status)
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

// shouldShowIndicator returns whether the given indicator should be visible
func (w *BatteryWidget) shouldShowIndicator(state *indicatorState, isActive bool) bool {
	if !isActive {
		return false
	}
	switch state.mode {
	case "always", "blink":
		return true
	case "never":
		return false
	case "notify", "notify_blink":
		return time.Now().Before(state.notifyUntil)
	default:
		return true // fallback to always
	}
}

// shouldBlinkIndicator returns whether the indicator should blink (be hidden this frame)
func (w *BatteryWidget) shouldBlinkIndicator(state *indicatorState) bool {
	if state.mode == "blink" || state.mode == "notify_blink" {
		return time.Now().Second()%2 != 0
	}
	return false
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

// getVisibleStatusIcon returns the icon name and its indicator state if visible
// Priority: charging > economy > ac_power (only show one icon)
// Returns empty string and nil if no indicator should be shown
func (w *BatteryWidget) getVisibleStatusIcon(status BatteryStatus) (string, *indicatorState) {
	// Check in priority order: charging > economy > plugged
	if w.shouldShowIndicator(&w.chargingState, status.IsCharging) {
		return "charging", &w.chargingState
	}
	if w.shouldShowIndicator(&w.economyState, status.IsEconomyMode) {
		return "economy", &w.economyState
	}
	if w.shouldShowIndicator(&w.pluggedState, status.IsPluggedIn) {
		return "ac_power", &w.pluggedState
	}
	return "", nil
}

// drawStatusIcon draws a status icon with 1px black border for visibility
func (w *BatteryWidget) drawStatusIcon(img *image.Gray, x, y int, status BatteryStatus) {
	iconName, state := w.getVisibleStatusIcon(status)
	if iconName == "" || w.iconSet == nil || state == nil {
		return
	}

	// Apply blink effect if indicator should blink
	if w.shouldBlinkIndicator(state) {
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

// formatMinutes formats a duration in minutes to a human-readable string
func formatMinutes(minutes int) string {
	if minutes <= 0 {
		return "-"
	}
	hours := minutes / 60
	mins := minutes % 60
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	return fmt.Sprintf("%dm", mins)
}

// expandToken expands a single token to its value
func (w *BatteryWidget) expandToken(token string, status BatteryStatus) string {
	switch token {
	// Percentage
	case "percent", "pct":
		return fmt.Sprintf("%d", status.Percentage)

	// Status text (respects power_status visibility and blink)
	case "status":
		if w.shouldShowIndicator(&w.chargingState, status.IsCharging) && !w.shouldBlinkIndicator(&w.chargingState) {
			return "CHG"
		}
		if w.shouldShowIndicator(&w.economyState, status.IsEconomyMode) && !w.shouldBlinkIndicator(&w.economyState) {
			return "ECO"
		}
		if w.shouldShowIndicator(&w.pluggedState, status.IsPluggedIn) && !w.shouldBlinkIndicator(&w.pluggedState) {
			return "AC"
		}
		return ""

	case "status_full":
		if w.shouldShowIndicator(&w.chargingState, status.IsCharging) && !w.shouldBlinkIndicator(&w.chargingState) {
			return "Charging"
		}
		if w.shouldShowIndicator(&w.economyState, status.IsEconomyMode) && !w.shouldBlinkIndicator(&w.economyState) {
			return "Economy"
		}
		if w.shouldShowIndicator(&w.pluggedState, status.IsPluggedIn) && !w.shouldBlinkIndicator(&w.pluggedState) {
			return "AC Power"
		}
		return ""

	// Time remaining
	case "time":
		// Smart: time to full if charging, time to empty otherwise
		if status.IsCharging && status.TimeToFull > 0 {
			return formatMinutes(status.TimeToFull)
		}
		return formatMinutes(status.TimeToEmpty)

	case "time_left":
		return formatMinutes(status.TimeToEmpty)

	case "time_to_full":
		return formatMinutes(status.TimeToFull)

	case "time_left_min":
		if status.TimeToEmpty > 0 {
			return fmt.Sprintf("%d", status.TimeToEmpty)
		}
		return "-"

	// Battery level state
	case "level":
		if status.Percentage <= w.criticalThreshold {
			return "critical"
		}
		if status.Percentage <= w.lowThreshold {
			return "low"
		}
		return "normal"

	// Boolean indicators (always show if status is active, ignoring visibility settings)
	case "charging":
		if status.IsCharging {
			return "CHG"
		}
		return ""

	case "plugged":
		if status.IsPluggedIn {
			return "AC"
		}
		return ""

	case "economy":
		if status.IsEconomyMode {
			return "ECO"
		}
		return ""

	default:
		// Unknown token - return as-is with braces
		return "{" + token + "}"
	}
}

// expandFormat expands all tokens in the format string
func (w *BatteryWidget) expandFormat(format string, status BatteryStatus) string {
	// Regex to match {token}
	re := regexp.MustCompile(`\{([a-zA-Z_][a-zA-Z0-9_]*)\}`)

	result := re.ReplaceAllStringFunc(format, func(match string) string {
		// Extract token name (remove braces)
		token := match[1 : len(match)-1]
		return w.expandToken(token, status)
	})

	// Clean up multiple spaces that may result from empty tokens
	result = regexp.MustCompile(`\s+`).ReplaceAllString(result, " ")
	result = strings.TrimSpace(result)

	return result
}

// renderText renders battery as text using format tokens
func (w *BatteryWidget) renderText(img *image.Gray, status BatteryStatus) {
	text := w.expandFormat(w.textFormat, status)
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

	// Draw status icon (charging, economy, or AC) in top-left corner
	w.drawStatusIcon(img, w.padding+2, w.padding+2, status)
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

	// Draw status icon (charging, economy, or AC) in top-left corner
	w.drawStatusIcon(img, w.padding+2, w.padding+2, status)
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

	// Use existing DrawGraph function (values are 0-100 for percentage)
	bitmap.DrawGraph(img, graphX, graphY, graphW, graphH, floatHistory, w.graphHistory, w.fillColor, w.lineColor)

	// Draw current percentage
	if w.showPercentage {
		text := fmt.Sprintf("%d%%", status.Percentage)
		bitmap.SmartDrawAlignedText(img, text, w.fontFace, w.fontName, "right", "top", w.padding+2)
	}

	// Draw status icon (charging, economy, or AC) in top-left corner
	w.drawStatusIcon(img, w.padding+2, w.padding+2, status)
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

	// Ensure minimum usable size but don't exceed widget bounds
	if batteryW < 8 {
		batteryW = 8
	}
	if batteryH < 6 {
		batteryH = 6
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
	w.drawStatusIcon(img, batteryX+2, batteryY+2, status)
}

// renderBatteryVertical draws vertical battery progressbar
func (w *BatteryWidget) renderBatteryVertical(img *image.Gray, status BatteryStatus, pos config.PositionConfig) {
	// Battery dimensions - leave room for the positive terminal nub at top
	nubH := 4
	batteryX := w.padding
	batteryY := w.padding + nubH + 1
	batteryW := pos.W - w.padding*2
	batteryH := pos.H - w.padding*2 - nubH - 1

	// Ensure minimum usable size but don't exceed widget bounds
	if batteryW < 6 {
		batteryW = 6
	}
	if batteryH < 8 {
		batteryH = 8
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

	// Draw status icon in top-left corner (inside battery body)
	w.drawStatusIcon(img, batteryX+2, batteryY+2, status)
}

// Stop cleans up resources
func (w *BatteryWidget) Stop() {
	// Nothing to clean up
}
