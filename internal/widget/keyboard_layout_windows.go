//go:build windows

package widget

import (
	"fmt"
	"image"
	"sync"
	"unsafe"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
	"golang.org/x/image/font"
)

var (
	getKeyboardLayout        = user32.NewProc("GetKeyboardLayout")
	getForegroundWindow      = user32.NewProc("GetForegroundWindow")
	getWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
)

// languageInfo holds display information for a language
type languageInfo struct {
	iso6391 string // 2-letter code (EN, RU, DE)
	iso6392 string // 3-letter code (ENG, RUS, DEU)
	name    string // Full name (English, Русский, Deutsch)
}

// lcidToLanguage maps Windows LCIDs to language information
var lcidToLanguage = map[uint16]languageInfo{
	0x0409: {"EN", "ENG", "English"},
	0x0809: {"EN", "ENG", "English (UK)"},
	0x0C09: {"EN", "ENG", "English (AU)"},
	0x1009: {"EN", "ENG", "English (CA)"},
	0x0419: {"RU", "RUS", "Русский"},
	0x0407: {"DE", "DEU", "Deutsch"},
	0x0807: {"DE", "DEU", "Deutsch (CH)"},
	0x0C07: {"DE", "DEU", "Deutsch (AT)"},
	0x040C: {"FR", "FRA", "Français"},
	0x080C: {"FR", "FRA", "Français (BE)"},
	0x0C0C: {"FR", "FRA", "Français (CA)"},
	0x100C: {"FR", "FRA", "Français (CH)"},
	0x040A: {"ES", "SPA", "Español"},
	0x080A: {"ES", "SPA", "Español (MX)"},
	0x0C0A: {"ES", "SPA", "Español (ES)"},
	0x0410: {"IT", "ITA", "Italiano"},
	0x0810: {"IT", "ITA", "Italiano (CH)"},
	0x0415: {"PL", "POL", "Polski"},
	0x0416: {"PT", "POR", "Português"},
	0x0816: {"PT", "POR", "Português (PT)"},
	0x0413: {"NL", "NLD", "Nederlands"},
	0x0813: {"NL", "NLD", "Nederlands (BE)"},
	0x0414: {"NO", "NOR", "Norsk"},
	0x041D: {"SV", "SWE", "Svenska"},
	0x040B: {"FI", "FIN", "Suomi"},
	0x0406: {"DA", "DAN", "Dansk"},
	0x0405: {"CS", "CES", "Čeština"},
	0x040E: {"HU", "HUN", "Magyar"},
	0x0418: {"RO", "RON", "Română"},
	0x0424: {"SL", "SLV", "Slovenščina"},
	0x041B: {"SK", "SLK", "Slovenčina"},
	0x0408: {"EL", "ELL", "Ελληνικά"},
	0x041F: {"TR", "TUR", "Türkçe"},
	0x0411: {"JA", "JPN", "日本語"},
	0x0412: {"KO", "KOR", "한국어"},
	0x0804: {"ZH", "ZHO", "中文"},
	0x0404: {"ZH", "ZHO", "中文 (TW)"},
	0x0C04: {"ZH", "ZHO", "中文 (HK)"},
	0x040D: {"HE", "HEB", "עברית"},
	0x0401: {"AR", "ARA", "العربية"},
	0x041E: {"TH", "THA", "ไทย"},
	0x042A: {"VI", "VIE", "Tiếng Việt"},
	0x0421: {"ID", "IND", "Bahasa Indonesia"},
	0x041A: {"HR", "HRV", "Hrvatski"},
	0x0422: {"UK", "UKR", "Українська"},
	0x0423: {"BE", "BEL", "Беларуская"},
	0x042F: {"MK", "MKD", "Македонски"},
	0x0403: {"CA", "CAT", "Català"},
	0x0456: {"GL", "GLG", "Galego"},
	0x042D: {"EU", "EUS", "Euskara"},
}

// KeyboardLayoutWidget displays current keyboard layout
type KeyboardLayoutWidget struct {
	*BaseWidget
	fontSize      int
	horizAlign    string
	vertAlign     string
	padding       int
	displayFormat string // "iso639-1", "iso639-2", "full"
	currentLayout string
	fontFace      font.Face
	lastLCID      uint16 // Cache last LCID to avoid unnecessary updates
	mu            sync.RWMutex
}

// NewKeyboardLayoutWidget creates a new keyboard layout widget
func NewKeyboardLayoutWidget(cfg config.WidgetConfig) (*KeyboardLayoutWidget, error) {
	base := NewBaseWidget(cfg)

	// Extract text settings
	fontSize := 10
	fontName := ""
	horizAlign := "center"
	vertAlign := "center"
	padding := 0

	if cfg.Text != nil {
		if cfg.Text.Size > 0 {
			fontSize = cfg.Text.Size
		}
		fontName = cfg.Text.Font
		if cfg.Text.Align != nil {
			if cfg.Text.Align.H != "" {
				horizAlign = cfg.Text.Align.H
			}
			if cfg.Text.Align.V != "" {
				vertAlign = cfg.Text.Align.V
			}
		}
	}

	// Extract padding from style
	if cfg.Style != nil {
		padding = cfg.Style.Padding
	}

	// Display format from config
	displayFormat := cfg.Format
	if displayFormat == "" {
		displayFormat = "iso639-1"
	}

	// Validate display format
	if displayFormat != "iso639-1" && displayFormat != "iso639-2" && displayFormat != "full" {
		return nil, fmt.Errorf("invalid format: %s (must be iso639-1, iso639-2, or full)", displayFormat)
	}

	// Load font
	fontFace, err := bitmap.LoadFont(fontName, fontSize)
	if err != nil {
		return nil, fmt.Errorf("failed to load font: %w", err)
	}

	widget := &KeyboardLayoutWidget{
		BaseWidget:    base,
		fontSize:      fontSize,
		horizAlign:    horizAlign,
		vertAlign:     vertAlign,
		padding:       padding,
		displayFormat: displayFormat,
		fontFace:      fontFace,
	}

	// Get initial layout
	layout := getCurrentKeyboardLayout()
	widget.lastLCID = layout
	widget.currentLayout = widget.formatLayout(layout)

	return widget, nil
}

// Update checks for keyboard layout changes
func (w *KeyboardLayoutWidget) Update() error {
	layout := getCurrentKeyboardLayout()

	// Only update if layout actually changed
	w.mu.Lock()
	if layout != w.lastLCID {
		w.lastLCID = layout
		w.currentLayout = w.formatLayout(layout)
	}
	w.mu.Unlock()

	return nil
}

// Render creates an image of the keyboard layout widget
func (w *KeyboardLayoutWidget) Render() (image.Image, error) {
	pos := w.GetPosition()
	style := w.GetStyle()

	img := bitmap.NewGrayscaleImage(pos.W, pos.H, w.GetRenderBackgroundColor())

	if style.Border {
		bitmap.DrawBorder(img, uint8(style.BorderColor))
	}

	// Draw layout text (thread-safe read)
	w.mu.RLock()
	layout := w.currentLayout
	w.mu.RUnlock()

	bitmap.DrawAlignedText(img, layout, w.fontFace, w.horizAlign, w.vertAlign, w.padding)

	return img, nil
}

// getCurrentKeyboardLayout gets the current keyboard layout LCID for the foreground window
func getCurrentKeyboardLayout() uint16 {
	// Get the foreground window handle
	hwnd, _, _ := getForegroundWindow.Call()
	if hwnd == 0 {
		// No foreground window - fall back to current thread
		ret, _, _ := getKeyboardLayout.Call(0)
		return uint16(ret & 0xFFFF)
	}

	// Get the thread ID for the foreground window
	var processID uint32
	threadID, _, _ := getWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&processID)))
	if threadID == 0 {
		// Failed to get thread ID - fall back to current thread
		ret, _, _ := getKeyboardLayout.Call(0)
		return uint16(ret & 0xFFFF)
	}

	// Get the keyboard layout for that thread
	ret, _, _ := getKeyboardLayout.Call(threadID)
	// Lower word contains the LCID
	lcid := uint16(ret & 0xFFFF)
	return lcid
}

// formatLayout formats the layout according to display format
func (w *KeyboardLayoutWidget) formatLayout(lcid uint16) string {
	info, ok := lcidToLanguage[lcid]
	if !ok {
		// Unknown layout - show LCID in hex
		return fmt.Sprintf("0x%04X", lcid)
	}

	switch w.displayFormat {
	case "iso639-2":
		return info.iso6392
	case "full":
		return info.name
	default: // "iso639-1"
		return info.iso6391
	}
}
