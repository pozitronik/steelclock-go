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
	showIcon            bool
	showBattery         bool
	batteryMode         string // "none", "icon", "text", "bar"
	showName            bool
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

	// Show flags (default: icon=true, name=false)
	showIcon := true
	if btCfg.ShowIcon != nil {
		showIcon = *btCfg.ShowIcon
	}
	showName := false
	if btCfg.ShowName != nil {
		showName = *btCfg.ShowName
	}

	// Battery mode: "none" disables battery display
	showBattery := btCfg.BatteryMode != "none"

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
		showIcon:            showIcon,
		showBattery:         showBattery,
		batteryMode:         btCfg.BatteryMode,
		showName:            showName,
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

	// Determine low-battery blink target:
	// If battery is shown, blink the battery indicator.
	// Otherwise, blink the icon (if shown) or the name.
	batteryIsLow := w.lowBatteryThreshold > 0 && connected && battSupported &&
		battLevel != nil && *battLevel <= w.lowBatteryThreshold
	hasBatteryDisplay := w.showBattery && connected && battSupported && battLevel != nil && deviceFound
	blinkBattery := batteryIsLow && hasBatteryDisplay
	blinkIcon := batteryIsLow && !hasBatteryDisplay && w.showIcon
	blinkName := batteryIsLow && !hasBatteryDisplay && !w.showIcon && w.showName

	// Determine icon name and color
	var iconName string
	var iconColor int

	switch {
	case !apiReachable || !adapterOk:
		iconName = "bt_off"
		iconColor = w.colorOff
	case !deviceFound:
		iconName = "bt_unknown"
		iconColor = w.colorOn // "?" is an attention indicator, not a "dimmed" state
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

	// Layout elements horizontally: [Icon] [Name] [Battery/Ellipsis]
	currentX := content.X

	// Draw icon (always reserve space to prevent layout shift during blink)
	if w.showIcon {
		icon := glyphs.GetIcon(w.iconSet, iconName)
		if icon != nil {
			skipDraw := (!deviceFound && !blinkVisible) || (blinkIcon && !batteryBlinkVisible)
			if !skipDraw && iconColor >= 0 {
				iconY := content.Y + (content.Height-icon.Height)/2
				glyphs.DrawGlyph(img, icon, currentX, iconY, color.Gray{Y: uint8(iconColor)})
			}
			currentX += icon.Width + 2
		}
	}

	// Draw ellipsis for transient states
	if isTransient {
		ellipsis := glyphs.GetIcon(w.iconSet, "bt_ellipsis")
		if ellipsis != nil {
			ellY := content.Y + (content.Height-ellipsis.Height)/2
			glyphs.DrawGlyph(img, ellipsis, currentX, ellY, color.Gray{Y: uint8(w.colorOn)})
			currentX += ellipsis.Width + 2
		}
	}

	// Draw device name (always reserve space to prevent layout shift during blink)
	if w.showName && devName != "" && deviceFound && apiReachable && adapterOk {
		nameColor := w.colorOff
		if connected {
			nameColor = w.colorOn
		}
		skipDraw := blinkName && !batteryBlinkVisible
		if !skipDraw && nameColor >= 0 {
			w.drawText(img, devName, currentX, content.Y, content.Height, uint8(nameColor))
		}
		// Always advance position to reserve space
		drawer := &font.Drawer{Face: w.fontFace}
		advance := drawer.MeasureString(devName)
		currentX += advance.Ceil() + 2
	}

	// Draw battery indicator
	if hasBatteryDisplay {
		if !blinkBattery || batteryBlinkVisible {
			remainingW := content.X + content.Width - currentX
			w.drawBattery(img, currentX, content.Y, remainingW, content.Height, *battLevel)
		}
	}

	return img, nil
}

// drawText draws text at the given position, vertically centered
func (w *Widget) drawText(img *image.Gray, text string, x, y, height int, c uint8) {
	metrics := w.fontFace.Metrics()
	textHeight := (metrics.Ascent + metrics.Descent).Ceil()
	baseY := y + (height-textHeight)/2 + metrics.Ascent.Ceil()

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

// drawBattery draws a battery level indicator at the given position.
// availW is the remaining horizontal space available for the battery.
func (w *Widget) drawBattery(img *image.Gray, x, y, availW, availH, level int) {
	if availW < 4 || availH < 4 {
		return
	}

	switch w.batteryMode {
	case "text":
		text := fmt.Sprintf("%d%%", level)
		w.drawText(img, text, x, y, availH, uint8(w.colorOn))
	case "bar":
		barWidth := availW
		barHeight := availH - 4
		if barHeight < 3 {
			barHeight = 3
		}
		barY := y + (availH-barHeight)/2
		fillWidth := barWidth * level / 100

		bitmap.DrawRectangle(img, x, barY, barWidth, barHeight, uint8(w.colorOn))
		if fillWidth > 2 {
			for py := barY + 1; py < barY+barHeight-1; py++ {
				for px := x + 1; px < x+1+fillWidth-2; px++ {
					img.SetGray(px, py, color.Gray{Y: uint8(w.colorOn)})
				}
			}
		}
	default: // "icon" - battery outline shape fitted to remaining space
		// Use available space, keeping ~2:1 aspect ratio capped by availW
		battH := availH
		battW := battH * 2
		if battW > availW {
			battW = availW
			battH = battW / 2
			if battH < 6 {
				battH = min(availH, 6)
			}
		}
		battY := y + (availH-battH)/2
		render.DrawBatteryShape(img, x, battY, battW, battH, render.BatteryShapeConfig{
			Orientation: "horizontal",
			Percentage:  level,
			FillColor:   uint8(w.colorOn),
			BorderColor: uint8(w.colorOn),
		})
	}
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
