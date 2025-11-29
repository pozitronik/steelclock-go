//go:build linux

package widget

import (
	"fmt"
	"image"
	"os/exec"
	"strings"
	"sync"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
	"golang.org/x/image/font"
)

// languageInfo holds display information for a language
type languageInfo struct {
	iso6391 string // 2-letter code (EN, RU, DE)
	iso6392 string // 3-letter code (ENG, RUS, DEU)
	name    string // Full name (English, Русский, Deutsch)
}

// xkbToLanguage maps XKB layout codes to language information
var xkbToLanguage = map[string]languageInfo{
	"us":    {"EN", "ENG", "English"},
	"gb":    {"EN", "ENG", "English (UK)"},
	"ru":    {"RU", "RUS", "Русский"},
	"de":    {"DE", "DEU", "Deutsch"},
	"fr":    {"FR", "FRA", "Français"},
	"es":    {"ES", "SPA", "Español"},
	"it":    {"IT", "ITA", "Italiano"},
	"pt":    {"PT", "POR", "Português"},
	"pl":    {"PL", "POL", "Polski"},
	"nl":    {"NL", "NLD", "Nederlands"},
	"no":    {"NO", "NOR", "Norsk"},
	"se":    {"SV", "SWE", "Svenska"},
	"fi":    {"FI", "FIN", "Suomi"},
	"dk":    {"DA", "DAN", "Dansk"},
	"cz":    {"CS", "CES", "Čeština"},
	"hu":    {"HU", "HUN", "Magyar"},
	"ro":    {"RO", "RON", "Română"},
	"si":    {"SL", "SLV", "Slovenščina"},
	"sk":    {"SK", "SLK", "Slovenčina"},
	"gr":    {"EL", "ELL", "Ελληνικά"},
	"tr":    {"TR", "TUR", "Türkçe"},
	"jp":    {"JA", "JPN", "日本語"},
	"kr":    {"KO", "KOR", "한국어"},
	"cn":    {"ZH", "ZHO", "中文"},
	"tw":    {"ZH", "ZHO", "中文 (TW)"},
	"il":    {"HE", "HEB", "עברית"},
	"ara":   {"AR", "ARA", "العربية"},
	"th":    {"TH", "THA", "ไทย"},
	"vn":    {"VI", "VIE", "Tiếng Việt"},
	"id":    {"ID", "IND", "Bahasa Indonesia"},
	"hr":    {"HR", "HRV", "Hrvatski"},
	"ua":    {"UK", "UKR", "Українська"},
	"by":    {"BE", "BEL", "Беларуская"},
	"mk":    {"MK", "MKD", "Македонски"},
	"latam": {"ES", "SPA", "Español (LA)"},
	"br":    {"PT", "POR", "Português (BR)"},
	"ca":    {"CA", "CAT", "Català"},
	"ch":    {"DE", "DEU", "Deutsch (CH)"},
	"at":    {"DE", "DEU", "Deutsch (AT)"},
	"be":    {"NL", "NLD", "Nederlands (BE)"},
	"ie":    {"EN", "ENG", "English (IE)"},
	"in":    {"HI", "HIN", "हिन्दी"},
	"bg":    {"BG", "BUL", "Български"},
	"rs":    {"SR", "SRP", "Српски"},
	"ee":    {"ET", "EST", "Eesti"},
	"lt":    {"LT", "LIT", "Lietuvių"},
	"lv":    {"LV", "LAV", "Latviešu"},
	"is":    {"IS", "ISL", "Íslenska"},
	"mt":    {"MT", "MLT", "Malti"},
	"al":    {"SQ", "SQI", "Shqip"},
	"me":    {"SR", "SRP", "Crnogorski"},
	"ba":    {"BS", "BOS", "Bosanski"},
	"am":    {"HY", "HYE", "Հայերdelays"},
	"ge":    {"KA", "KAT", "ქართული"},
	"az":    {"AZ", "AZE", "Azərbaycanca"},
	"kz":    {"KK", "KAZ", "Қазақша"},
	"kg":    {"KY", "KIR", "Кыргызча"},
	"tj":    {"TG", "TGK", "Тоҷикӣ"},
	"tm":    {"TK", "TUK", "Türkmen"},
	"uz":    {"UZ", "UZB", "O'zbek"},
	"mn":    {"MN", "MON", "Монгол"},
	"np":    {"NE", "NEP", "नेपाली"},
	"bd":    {"BN", "BEN", "বাংলা"},
	"lk":    {"SI", "SIN", "සිංහල"},
	"mm":    {"MY", "MYA", "မြန်မာ"},
	"kh":    {"KM", "KHM", "ខ្មែរ"},
	"la":    {"LO", "LAO", "ລາວ"},
	"my":    {"MS", "MSA", "Bahasa Melayu"},
	"ph":    {"TL", "TGL", "Tagalog"},
}

// KeyboardLayoutWidget displays current keyboard layout
type KeyboardLayoutWidget struct {
	*BaseWidget
	fontSize      int
	fontName      string
	horizAlign    string
	vertAlign     string
	padding       int
	displayFormat string // "iso639-1", "iso639-2", "full"
	currentLayout string
	fontFace      font.Face
	lastXkbLayout string // Cache last XKB layout to avoid unnecessary updates
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

	fontFace, err := bitmap.LoadFont(fontName, fontSize)
	if err != nil {
		return nil, fmt.Errorf("failed to load font: %w", err)
	}

	widget := &KeyboardLayoutWidget{
		BaseWidget:    base,
		fontSize:      fontSize,
		fontName:      fontName,
		horizAlign:    horizAlign,
		vertAlign:     vertAlign,
		padding:       padding,
		displayFormat: displayFormat,
		fontFace:      fontFace,
	}

	// Get initial layout
	layout := getCurrentKeyboardLayout()
	widget.lastXkbLayout = layout
	widget.currentLayout = widget.formatLayout(layout)

	return widget, nil
}

// Update updates the keyboard layout state
func (w *KeyboardLayoutWidget) Update() error {
	layout := getCurrentKeyboardLayout()

	// Only update if layout actually changed
	w.mu.Lock()
	if layout != w.lastXkbLayout {
		w.lastXkbLayout = layout
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

	if style.Border >= 0 {
		bitmap.DrawBorder(img, uint8(style.Border))
	}

	// Draw layout text (thread-safe read)
	w.mu.RLock()
	layout := w.currentLayout
	w.mu.RUnlock()

	bitmap.SmartDrawAlignedText(img, layout, w.fontFace, w.fontName, w.horizAlign, w.vertAlign, w.padding)

	return img, nil
}

// getCurrentKeyboardLayout gets the current keyboard layout using setxkbmap
func getCurrentKeyboardLayout() string {
	// Try setxkbmap first (works with X11)
	out, err := exec.Command("setxkbmap", "-query").Output()
	if err == nil {
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "layout:") {
				layout := strings.TrimSpace(strings.TrimPrefix(line, "layout:"))
				// Handle multiple layouts (e.g., "us,ru") - return first one
				// TODO: detect active layout group
				if idx := strings.Index(layout, ","); idx != -1 {
					layout = layout[:idx]
				}
				return layout
			}
		}
	}

	// Fallback: try reading from /etc/default/keyboard
	out, err = exec.Command("cat", "/etc/default/keyboard").Output()
	if err == nil {
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "XKBLAYOUT=") {
				layout := strings.Trim(strings.TrimPrefix(line, "XKBLAYOUT="), "\"'")
				if idx := strings.Index(layout, ","); idx != -1 {
					layout = layout[:idx]
				}
				return layout
			}
		}
	}

	return "unknown"
}

// formatLayout formats the layout according to display format
func (w *KeyboardLayoutWidget) formatLayout(xkbLayout string) string {
	info, ok := xkbToLanguage[xkbLayout]
	if !ok {
		// Unknown layout - show raw XKB code in uppercase
		return strings.ToUpper(xkbLayout)
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
