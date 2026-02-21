package bluetooth

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/bitmap/glyphs"
	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/shared"
	"github.com/pozitronik/steelclock-go/internal/shared/anim"
	"github.com/pozitronik/steelclock-go/internal/shared/render"
	"github.com/pozitronik/steelclock-go/internal/widget"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

func init() {
	widget.Register("bluetooth", func(cfg config.WidgetConfig) (widget.Widget, error) {
		return New(cfg)
	})
}

// API response structures (local, not importing bqc)

type apiResponse struct {
	Adapter         *adapterInfo `json:"adapter,omitempty"` // May be absent from single-device endpoint
	Address         string       `json:"address"`
	Name            string       `json:"name"`
	DisplayName     string       `json:"displayName"`
	Type            string       `json:"type"`
	ConnectionState string       `json:"connectionState"`
	IsConnected     bool         `json:"isConnected"`
	Battery         batteryInfo  `json:"battery"`
}

type adapterInfo struct {
	Available bool `json:"available"`
	Enabled   bool `json:"enabled"`
}

type batteryInfo struct {
	Level     *int `json:"level"`
	Supported bool `json:"supported"`
}

// Widget displays Bluetooth device status from the bqc REST API.
// Each instance tracks a single device by MAC address.
type Widget struct {
	*widget.BaseWidget
	// Configuration
	address             string
	apiURL              string
	format              string
	tokens              []Token
	colorOn             int
	colorOff            int
	lowBatteryThreshold int
	// Display settings
	fontSize   int
	fontName   string
	horizAlign config.HAlign
	vertAlign  config.VAlign
	padding    int
	fontFace   font.Face
	// Icon sets (selected based on widget size)
	iconSet *glyphs.GlyphSet
	// Blink animator for "not found" state
	blink *anim.BlinkAnimator
	// Blink animator for low battery indicator
	batteryBlink *anim.BlinkAnimator
	// HTTP client
	httpClient *http.Client
	// State (mutex-protected)
	mu              sync.RWMutex
	connected       bool
	connectionState string
	deviceType      string
	deviceName      string
	batteryLevel    *int
	batterySupport  bool
	adapterOk       bool // adapter available and enabled
	deviceFound     bool // 404 = false
	apiReachable    bool // connection error = false
}

// New creates a new Bluetooth widget
func New(cfg config.WidgetConfig) (*Widget, error) {
	base := widget.NewBaseWidget(cfg)
	helper := shared.NewConfigHelper(cfg)

	// Extract common settings
	textSettings := helper.GetTextSettings()
	padding := helper.GetPadding()

	// Bluetooth-specific settings
	if cfg.Bluetooth == nil || cfg.Bluetooth.Address == "" {
		return nil, fmt.Errorf("bluetooth widget requires 'address' in bluetooth config")
	}

	btCfg := cfg.Bluetooth

	// Parse format string
	format := btCfg.Format
	tokens := parseBluetoothFormat(format)

	// Colors (on/off like keyboard widget)
	colorOn := 255
	colorOff := 100
	if cfg.Colors != nil {
		if cfg.Colors.On != nil {
			colorOn = *cfg.Colors.On
		}
		if cfg.Colors.Off != nil {
			colorOff = *cfg.Colors.Off
		}
	}

	// Load font
	fontFace, err := bitmap.LoadFont(textSettings.FontName, textSettings.FontSize)
	if err != nil {
		return nil, fmt.Errorf("failed to load font: %w", err)
	}

	// Select icon set based on widget height
	var iconSet *glyphs.GlyphSet
	h := cfg.Position.H
	switch {
	case h >= 16:
		iconSet = glyphs.BluetoothIcons16x16
	case h >= 12:
		iconSet = glyphs.BluetoothIcons12x12
	default:
		iconSet = glyphs.BluetoothIcons8x8
	}

	// Create blink animator for "not found" state (always blink, 500ms)
	blinkAnim := anim.NewBlinkAnimator(anim.BlinkAlways, 500*time.Millisecond)

	return &Widget{
		BaseWidget:          base,
		address:             btCfg.Address,
		apiURL:              btCfg.APIURL,
		format:              format,
		tokens:              tokens,
		colorOn:             colorOn,
		colorOff:            colorOff,
		lowBatteryThreshold: btCfg.LowBatteryThreshold,
		fontSize:            textSettings.FontSize,
		fontName:            textSettings.FontName,
		horizAlign:          textSettings.HorizAlign,
		vertAlign:           textSettings.VertAlign,
		padding:             padding,
		fontFace:            fontFace,
		iconSet:             iconSet,
		blink:               blinkAnim,
		batteryBlink:        anim.NewBlinkAnimator(anim.BlinkAlways, 500*time.Millisecond),
		httpClient:          &http.Client{Timeout: 3 * time.Second},
		apiReachable:        true, // optimistic start
		deviceFound:         true, // optimistic start
		adapterOk:           true, // optimistic start
	}, nil
}

// Update fetches device status from the bqc API
func (w *Widget) Update() error {
	url := fmt.Sprintf("http://%s/api/devices/%s", w.apiURL, w.address)

	resp, err := w.httpClient.Get(url)
	if err != nil {
		w.mu.Lock()
		w.apiReachable = false
		w.mu.Unlock()
		log.Printf("bluetooth: API unreachable at %s: %v", url, err)
		return nil // Not a fatal error - widget shows "off" state
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		w.mu.Lock()
		w.apiReachable = false
		w.mu.Unlock()
		return nil
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	w.apiReachable = true

	switch resp.StatusCode {
	case http.StatusOK:
		var device apiResponse
		if err := json.Unmarshal(body, &device); err != nil {
			log.Printf("bluetooth: failed to parse response: %v", err)
			return nil
		}

		w.deviceFound = true
		// If adapter info is present, use it; otherwise assume adapter is ok
		// (the API responded with device data, so adapter must be working)
		if device.Adapter != nil {
			w.adapterOk = device.Adapter.Available && device.Adapter.Enabled
		} else {
			w.adapterOk = true
		}
		w.connected = device.IsConnected
		w.connectionState = device.ConnectionState
		w.deviceType = device.Type

		// Use displayName with fallback to name
		if device.DisplayName != "" {
			w.deviceName = device.DisplayName
		} else {
			w.deviceName = device.Name
		}

		w.batteryLevel = device.Battery.Level
		w.batterySupport = device.Battery.Supported

	case http.StatusNotFound:
		w.deviceFound = false

	default:
		log.Printf("bluetooth: unexpected status %d from API", resp.StatusCode)
	}

	// Update blink animators
	w.blink.Update(0)
	w.batteryBlink.Update(0)

	return nil
}

// Render creates an image of the Bluetooth device status
func (w *Widget) Render() (image.Image, error) {
	img := w.CreateCanvas()
	w.ApplyBorder(img)
	content := w.GetContentArea()

	w.mu.RLock()
	apiReachable := w.apiReachable
	adapterOk := w.adapterOk
	deviceFound := w.deviceFound
	connected := w.connected
	connState := w.connectionState
	devType := w.deviceType
	devName := w.deviceName
	battLevel := w.batteryLevel
	battSupported := w.batterySupport
	blinkVisible := w.blink.ShouldRender()
	batteryBlinkVisible := w.batteryBlink.ShouldRender()
	w.mu.RUnlock()

	// Determine visual state
	isTransient := connState == "Connecting" || connState == "Disconnecting"

	// Determine low-battery blink
	batteryIsLow := w.lowBatteryThreshold > 0 && connected && battSupported &&
		battLevel != nil && *battLevel <= w.lowBatteryThreshold
	blinkTargetIdx := -1
	if batteryIsLow {
		blinkTargetIdx = findBlinkTarget(w.tokens)
	}

	// Determine icon name and color
	var iconName string
	var iconColor int

	switch {
	case !apiReachable || !adapterOk:
		iconName = "bt_off"
		iconColor = w.colorOff
	case !deviceFound:
		iconName = "bt_unknown"
		iconColor = w.colorOn
	case isTransient:
		iconName = deviceTypeToIcon(devType)
		iconColor = w.colorOn
	case connected:
		iconName = deviceTypeToIcon(devType)
		iconColor = w.colorOn
	default: // disconnected
		iconName = deviceTypeToIcon(devType)
		iconColor = w.colorOff
	}

	// State for token text resolution
	state := &tokenState{
		apiReachable:  apiReachable,
		adapterOk:     adapterOk,
		deviceFound:   deviceFound,
		connected:     connected,
		connState:     connState,
		devName:       devName,
		battLevel:     battLevel,
		battSupported: battSupported,
		isTransient:   isTransient,
		iconName:      iconName,
		iconColor:     iconColor,
	}

	// === Pass 1: Measure total width of all tokens ===
	totalWidth := 0
	widths := make([]int, len(w.tokens))
	for i := range w.tokens {
		widths[i] = w.measureBluetoothToken(&w.tokens[i], state)
		totalWidth += widths[i]
	}

	// === Calculate starting X based on horizontal alignment ===
	var startX int
	switch w.horizAlign {
	case config.AlignLeft:
		startX = content.X
	case config.AlignRight:
		startX = content.X + content.Width - totalWidth
		if startX < content.X {
			startX = content.X
		}
	default: // center
		startX = content.X + (content.Width-totalWidth)/2
		if startX < content.X {
			startX = content.X
		}
	}

	// === Pass 2: Render each token ===
	currentX := startX
	for i := range w.tokens {
		t := &w.tokens[i]

		// Determine if this token should be hidden due to blink
		skipDraw := false
		if i == blinkTargetIdx && !batteryBlinkVisible {
			skipDraw = true
		}
		// Icon blink for "not found" state
		if t.Type == TokenIcon && !deviceFound && !blinkVisible {
			skipDraw = true
		}

		if !skipDraw {
			w.renderBluetoothToken(img, t, currentX, content.Y, content.Height, state)
		}

		// Always advance X to reserve space (prevent layout shift)
		currentX += widths[i]
	}

	return img, nil
}

// tokenState holds snapshot of device state for token rendering
type tokenState struct {
	apiReachable  bool
	adapterOk     bool
	deviceFound   bool
	connected     bool
	connState     string
	devName       string
	battLevel     *int
	battSupported bool
	isTransient   bool
	iconName      string
	iconColor     int
}

// measureBluetoothToken returns the pixel width of a single token
func (w *Widget) measureBluetoothToken(t *Token, state *tokenState) int {
	switch t.Type {
	case TokenLiteral:
		width, _ := bitmap.SmartMeasureText(t.Literal, w.fontFace, w.fontName)
		return width
	case TokenIcon:
		icon := glyphs.GetIcon(w.iconSet, state.iconName)
		if icon != nil {
			return icon.Width + 2 // +2 gap after icon
		}
		return 0
	case TokenText:
		text := w.resolveTextToken(t, state)
		if text == "" {
			return 0
		}
		width, _ := bitmap.SmartMeasureText(text, w.fontFace, w.fontName)
		return width
	case TokenShape:
		return w.measureShapeToken(t)
	}
	return 0
}

// resolveTextToken returns the string value for a text token
func (w *Widget) resolveTextToken(t *Token, state *tokenState) string {
	if !state.apiReachable || !state.adapterOk || !state.deviceFound {
		return ""
	}
	switch t.Name {
	case "name":
		return state.devName
	case "level":
		if state.connected && state.battSupported && state.battLevel != nil {
			return fmt.Sprintf("%d%%", *state.battLevel)
		}
		return ""
	case "state":
		return state.connState
	}
	return ""
}

// measureShapeToken returns the pixel width of a shape token.
// The size parameter N always specifies the horizontal width,
// regardless of orientation. Vertical height comes from the content area.
func (w *Widget) measureShapeToken(t *Token) int {
	size := w.parseShapeSize(t)
	if size <= 0 {
		return 0
	}
	return size
}

// parseShapeSize extracts the pixel size from the token parameter
func (w *Widget) parseShapeSize(t *Token) int {
	if t.Param == "" {
		return 0
	}
	var size int
	_, _ = fmt.Sscanf(t.Param, "%d", &size)
	return size
}

// renderBluetoothToken draws a single token at (x, y) within the given height
func (w *Widget) renderBluetoothToken(img *image.Gray, t *Token, x, y, height int, state *tokenState) {
	switch t.Type {
	case TokenLiteral:
		textColor := w.colorOn
		if !state.connected && state.deviceFound && state.apiReachable && state.adapterOk {
			textColor = w.colorOff
		}
		w.drawTextAligned(img, t.Literal, x, y, height, uint8(textColor))
	case TokenIcon:
		w.renderIconToken(img, x, y, height, state)
	case TokenText:
		w.renderTextToken(img, t, x, y, height, state)
	case TokenShape:
		w.renderShapeToken(img, t, x, y, height, state)
	}
}

// renderIconToken draws the device type icon
func (w *Widget) renderIconToken(img *image.Gray, x, y, height int, state *tokenState) {
	icon := glyphs.GetIcon(w.iconSet, state.iconName)
	if icon == nil || state.iconColor < 0 {
		return
	}

	var iconY int
	switch w.vertAlign {
	case config.AlignTop:
		iconY = y
	case config.AlignBottom:
		iconY = y + height - icon.Height
	default: // center
		iconY = y + (height-icon.Height)/2
	}
	glyphs.DrawGlyph(img, icon, x, iconY, color.Gray{Y: uint8(state.iconColor)})
}

// renderTextToken draws a text token (name, level, state)
func (w *Widget) renderTextToken(img *image.Gray, t *Token, x, y, height int, state *tokenState) {
	text := w.resolveTextToken(t, state)
	if text == "" {
		return
	}
	c := w.colorOn
	if !state.connected {
		c = w.colorOff
	}
	w.drawTextAligned(img, text, x, y, height, uint8(c))
}

// renderShapeToken draws a battery or bar shape
func (w *Widget) renderShapeToken(img *image.Gray, t *Token, x, y, height int, state *tokenState) {
	if !state.connected || !state.battSupported || state.battLevel == nil || !state.deviceFound {
		return
	}
	level := *state.battLevel
	size := w.parseShapeSize(t)
	if size <= 0 {
		return
	}

	switch t.Name {
	case "battery", "battery_h":
		w.drawBatteryShape(img, x, y, size, height, level, "horizontal")
	case "battery_v":
		w.drawBatteryShape(img, x, y, size, height, level, "vertical")
	case "bar", "bar_h":
		w.drawBarShape(img, x, y, size, height, level, false)
	case "bar_v":
		w.drawBarShape(img, x, y, size, height, level, true)
	}
}

// drawBatteryShape draws a battery outline+fill shape
func (w *Widget) drawBatteryShape(img *image.Gray, x, y, shapeW, availH, level int, orientation string) {
	if shapeW < 4 || availH < 4 {
		return
	}

	// For horizontal, cap height to maintain reasonable aspect ratio
	battH := availH
	battW := shapeW
	if orientation == "horizontal" {
		if battH > battW {
			battH = battW / 2
			if battH < 6 {
				battH = min(availH, 6)
			}
		}
	}

	battY := y + (availH-battH)/2
	render.DrawBatteryShape(img, x, battY, battW, battH, render.BatteryShapeConfig{
		Orientation: orientation,
		Percentage:  level,
		FillColor:   uint8(w.colorOn),
		BorderColor: uint8(w.colorOn),
	})
}

// drawBarShape draws a simple filled bar
func (w *Widget) drawBarShape(img *image.Gray, x, y, barW, availH, level int, vertical bool) {
	if barW < 3 || availH < 3 {
		return
	}

	if vertical {
		barH := availH - 4
		if barH < 3 {
			barH = 3
		}
		barY := y + (availH-barH)/2
		fillHeight := barH * level / 100

		bitmap.DrawRectangle(img, x, barY, barW, barH, uint8(w.colorOn))
		if fillHeight > 2 {
			startY := barY + barH - 1 - fillHeight + 1
			for py := startY; py < barY+barH-1; py++ {
				for px := x + 1; px < x+barW-1; px++ {
					img.SetGray(px, py, color.Gray{Y: uint8(w.colorOn)})
				}
			}
		}
	} else {
		barH := availH - 4
		if barH < 3 {
			barH = 3
		}
		barY := y + (availH-barH)/2
		fillWidth := barW * level / 100

		bitmap.DrawRectangle(img, x, barY, barW, barH, uint8(w.colorOn))
		if fillWidth > 2 {
			for py := barY + 1; py < barY+barH-1; py++ {
				for px := x + 1; px < x+1+fillWidth-2; px++ {
					img.SetGray(px, py, color.Gray{Y: uint8(w.colorOn)})
				}
			}
		}
	}
}

// drawTextAligned draws text at the given position with color and vertical alignment
func (w *Widget) drawTextAligned(img *image.Gray, text string, x, y, height int, c uint8) {
	metrics := w.fontFace.Metrics()
	textHeight := (metrics.Ascent + metrics.Descent).Ceil()
	ascent := metrics.Ascent.Ceil()

	var baseY int
	switch w.vertAlign {
	case config.AlignTop:
		baseY = y + ascent
	case config.AlignBottom:
		baseY = y + height - textHeight + ascent
	default: // center
		baseY = y + (height-textHeight)/2 + ascent
	}

	drawer := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(color.Gray{Y: c}),
		Face: w.fontFace,
		Dot: fixed.Point26_6{
			X: fixed.I(x),
			Y: fixed.I(baseY),
		},
	}
	drawer.DrawString(text)
}

// deviceTypeToIcon maps bqc API device type strings to icon names
func deviceTypeToIcon(deviceType string) string {
	switch deviceType {
	case "AudioOutput", "Headset":
		return "bt_headphones"
	case "AudioInput":
		return "bt_microphone"
	case "Keyboard":
		return "bt_keyboard"
	case "Mouse":
		return "bt_mouse"
	case "Gamepad":
		return "bt_gamepad"
	case "Computer":
		return "bt_computer"
	case "Phone":
		return "bt_phone"
	default: // "Unknown", "HID", or any other type
		return "bt_generic"
	}
}
