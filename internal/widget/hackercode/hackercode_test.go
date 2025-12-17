package hackercode

import (
	"strings"
	"testing"

	"github.com/pozitronik/steelclock-go/internal/config"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.WidgetConfig
		wantErr bool
	}{
		{
			name: "default configuration",
			cfg: config.WidgetConfig{
				Type:     "hacker_code",
				Position: config.PositionConfig{W: 128, H: 40},
			},
			wantErr: false,
		},
		{
			name: "c style",
			cfg: config.WidgetConfig{
				Type:     "hacker_code",
				Position: config.PositionConfig{W: 128, H: 40},
				HackerCode: &config.HackerCodeConfig{
					Style: "c",
				},
			},
			wantErr: false,
		},
		{
			name: "asm style",
			cfg: config.WidgetConfig{
				Type:     "hacker_code",
				Position: config.PositionConfig{W: 128, H: 40},
				HackerCode: &config.HackerCodeConfig{
					Style: "asm",
				},
			},
			wantErr: false,
		},
		{
			name: "mixed style",
			cfg: config.WidgetConfig{
				Type:     "hacker_code",
				Position: config.PositionConfig{W: 128, H: 40},
				HackerCode: &config.HackerCodeConfig{
					Style: "mixed",
				},
			},
			wantErr: false,
		},
		{
			name: "custom typing speed (integer)",
			cfg: config.WidgetConfig{
				Type:     "hacker_code",
				Position: config.PositionConfig{W: 128, H: 40},
				HackerCode: &config.HackerCodeConfig{
					TypingSpeed: &config.IntOrRange{Min: 100, Max: 100},
				},
			},
			wantErr: false,
		},
		{
			name: "custom typing speed (range)",
			cfg: config.WidgetConfig{
				Type:     "hacker_code",
				Position: config.PositionConfig{W: 128, H: 40},
				HackerCode: &config.HackerCodeConfig{
					TypingSpeed: &config.IntOrRange{Min: 20, Max: 100},
				},
			},
			wantErr: false,
		},
		{
			name: "small font via text settings",
			cfg: config.WidgetConfig{
				Type:     "hacker_code",
				Position: config.PositionConfig{W: 128, H: 40},
				Text: &config.TextConfig{
					Font: "3x5",
				},
			},
			wantErr: false,
		},
		{
			name: "large font via text settings",
			cfg: config.WidgetConfig{
				Type:     "hacker_code",
				Position: config.PositionConfig{W: 128, H: 40},
				Text: &config.TextConfig{
					Font: "5x7",
				},
			},
			wantErr: false,
		},
		{
			name: "cursor disabled",
			cfg: config.WidgetConfig{
				Type:     "hacker_code",
				Position: config.PositionConfig{W: 128, H: 40},
				HackerCode: &config.HackerCodeConfig{
					ShowCursor: boolPtr(false),
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, err := New(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && w == nil {
				t.Error("New() returned nil widget without error")
			}
		})
	}
}

func TestWidget_Update(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:     "hacker_code",
		Position: config.PositionConfig{W: 128, H: 40},
		HackerCode: &config.HackerCodeConfig{
			TypingSpeed: &config.IntOrRange{Min: 1000, Max: 1000}, // Fast typing for test
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Update should not error
	if err := w.Update(); err != nil {
		t.Errorf("Update() error = %v", err)
	}

	// Multiple updates should not error
	for i := 0; i < 10; i++ {
		if err := w.Update(); err != nil {
			t.Errorf("Update() iteration %d error = %v", i, err)
		}
	}
}

func TestWidget_Render(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:     "hacker_code",
		Position: config.PositionConfig{W: 128, H: 40},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Render should not error
	img, err := w.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}

	if img == nil {
		t.Error("Render() returned nil image")
	}

	// Check image dimensions
	bounds := img.Bounds()
	if bounds.Dx() != 128 || bounds.Dy() != 40 {
		t.Errorf("Render() image size = %dx%d, want 128x40", bounds.Dx(), bounds.Dy())
	}
}

func TestCGenerator(t *testing.T) {
	gen := NewCGenerator(12345)

	// Generate several lines and check they're not empty
	for i := 0; i < 50; i++ {
		line := gen.NextLine()
		if line == "" {
			t.Errorf("NextLine() returned empty string at iteration %d", i)
		}
	}
}

func TestCGenerator_OutputFormat(t *testing.T) {
	gen := NewCGenerator(42)

	// Generate lines and verify they look like C code
	hasFunction := false
	hasVariable := false
	hasControl := false

	for i := 0; i < 100; i++ {
		line := gen.NextLine()

		if strings.Contains(line, "(") && strings.Contains(line, ")") && strings.Contains(line, "{") {
			hasFunction = true
		}
		if strings.Contains(line, "=") && strings.Contains(line, ";") {
			hasVariable = true
		}
		if strings.HasPrefix(strings.TrimSpace(line), "if ") ||
			strings.HasPrefix(strings.TrimSpace(line), "for ") ||
			strings.HasPrefix(strings.TrimSpace(line), "while ") {
			hasControl = true
		}
	}

	if !hasFunction {
		t.Error("CGenerator did not produce any function definitions")
	}
	if !hasVariable {
		t.Error("CGenerator did not produce any variable assignments")
	}
	if !hasControl {
		t.Error("CGenerator did not produce any control flow statements")
	}
}

func TestCGenerator_Reset(t *testing.T) {
	gen := NewCGenerator(12345)

	// Generate some lines
	for i := 0; i < 10; i++ {
		gen.NextLine()
	}

	// Reset
	gen.Reset()

	// Verify state is reset
	if gen.inFunction {
		t.Error("Reset() did not clear inFunction")
	}
	if gen.indentLevel != 0 {
		t.Errorf("Reset() did not clear indentLevel, got %d", gen.indentLevel)
	}
	if gen.lineCount != 0 {
		t.Errorf("Reset() did not clear lineCount, got %d", gen.lineCount)
	}
}

func TestAsmGenerator(t *testing.T) {
	gen := NewAsmGenerator(12345)

	// Generate several lines and check they're not empty
	for i := 0; i < 50; i++ {
		line := gen.NextLine()
		if line == "" {
			t.Errorf("NextLine() returned empty string at iteration %d", i)
		}
	}
}

func TestAsmGenerator_OutputFormat(t *testing.T) {
	gen := NewAsmGenerator(42)

	// Generate lines and verify they look like assembly
	hasMov := false
	hasJump := false
	hasStack := false
	hasLabel := false

	for i := 0; i < 100; i++ {
		line := gen.NextLine()

		if strings.HasPrefix(line, "MOV ") {
			hasMov = true
		}
		if strings.HasPrefix(line, "J") || strings.HasPrefix(line, "CALL ") {
			hasJump = true
		}
		if strings.HasPrefix(line, "PUSH ") || strings.HasPrefix(line, "POP ") {
			hasStack = true
		}
		if strings.HasSuffix(line, ":") {
			hasLabel = true
		}
	}

	if !hasMov {
		t.Error("AsmGenerator did not produce any MOV instructions")
	}
	if !hasJump {
		t.Error("AsmGenerator did not produce any jump/call instructions")
	}
	if !hasStack {
		t.Error("AsmGenerator did not produce any stack operations")
	}
	if !hasLabel {
		t.Error("AsmGenerator did not produce any labels")
	}
}

func TestAsmGenerator_Reset(t *testing.T) {
	gen := NewAsmGenerator(12345)

	// Generate some lines
	for i := 0; i < 10; i++ {
		gen.NextLine()
	}

	// Reset
	gen.Reset()

	// Verify state is reset
	if gen.lineCount != 0 {
		t.Errorf("Reset() did not clear lineCount, got %d", gen.lineCount)
	}
}

func TestMixedStyle(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:     "hacker_code",
		Position: config.PositionConfig{W: 128, H: 40},
		HackerCode: &config.HackerCodeConfig{
			Style: "mixed",
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Update many times to trigger potential style switches
	for i := 0; i < 100; i++ {
		if err := w.Update(); err != nil {
			t.Errorf("Update() error = %v", err)
		}
	}

	// Render should still work
	img, err := w.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}
	if img == nil {
		t.Error("Render() returned nil image")
	}
}

func TestWidget_Name(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:     "hacker_code",
		Position: config.PositionConfig{W: 128, H: 40},
	}
	cfg.ID = "test_hacker_code" // ID is auto-generated in real use, set manually for test

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	name := w.Name()
	if name != "test_hacker_code" {
		t.Errorf("Name() = %q, want %q", name, "test_hacker_code")
	}
}

func TestWidget_FontSelection(t *testing.T) {
	tests := []struct {
		name       string
		fontName   string
		wantWidth  int
		wantHeight int
	}{
		{
			name:       "explicit 3x5 font",
			fontName:   "3x5",
			wantWidth:  4, // 3 + 1 spacing
			wantHeight: 6, // 5 + 1 spacing
		},
		{
			name:       "explicit 5x7 font",
			fontName:   "5x7",
			wantWidth:  6, // 5 + 1 spacing
			wantHeight: 8, // 7 + 1 spacing
		},
		{
			name:       "pixel3x5 alias",
			fontName:   "pixel3x5",
			wantWidth:  4,
			wantHeight: 6,
		},
		{
			name:       "pixel5x7 alias",
			fontName:   "pixel5x7",
			wantWidth:  6,
			wantHeight: 8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.WidgetConfig{
				Type:     "hacker_code",
				Position: config.PositionConfig{W: 128, H: 40},
				Text: &config.TextConfig{
					Font: tt.fontName,
				},
			}
			w, err := New(cfg)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}
			if w.charWidth != tt.wantWidth {
				t.Errorf("charWidth = %d, want %d", w.charWidth, tt.wantWidth)
			}
			if w.charHeight != tt.wantHeight {
				t.Errorf("charHeight = %d, want %d", w.charHeight, tt.wantHeight)
			}
		})
	}
}

func TestWidget_TypingSpeedRange(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:     "hacker_code",
		Position: config.PositionConfig{W: 128, H: 40},
		HackerCode: &config.HackerCodeConfig{
			TypingSpeed: &config.IntOrRange{Min: 10, Max: 500},
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Verify min and max are set correctly
	if w.typingSpeedMin != 10 {
		t.Errorf("typingSpeedMin = %d, want 10", w.typingSpeedMin)
	}
	if w.typingSpeedMax != 500 {
		t.Errorf("typingSpeedMax = %d, want 500", w.typingSpeedMax)
	}

	// Verify current speed is within range
	if w.currentSpeed < 10 || w.currentSpeed > 500 {
		t.Errorf("currentSpeed = %d, want between 10 and 500", w.currentSpeed)
	}

	// Verify pickTypingSpeed returns values within range
	for i := 0; i < 100; i++ {
		speed := w.pickTypingSpeed()
		if speed < 10 || speed > 500 {
			t.Errorf("pickTypingSpeed() = %d, want between 10 and 500", speed)
		}
	}
}

func TestWidget_TypingSpeedFixed(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:     "hacker_code",
		Position: config.PositionConfig{W: 128, H: 40},
		HackerCode: &config.HackerCodeConfig{
			TypingSpeed: &config.IntOrRange{Min: 100, Max: 100}, // Fixed speed
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Verify both min and max are set to the same value
	if w.typingSpeedMin != 100 || w.typingSpeedMax != 100 {
		t.Errorf("typingSpeed = {%d, %d}, want {100, 100}", w.typingSpeedMin, w.typingSpeedMax)
	}

	// Verify pickTypingSpeed always returns the fixed value
	for i := 0; i < 10; i++ {
		speed := w.pickTypingSpeed()
		if speed != 100 {
			t.Errorf("pickTypingSpeed() = %d, want 100", speed)
		}
	}
}

func TestWidget_TypingSpeedSwapped(t *testing.T) {
	// Test that min > max gets swapped correctly
	cfg := config.WidgetConfig{
		Type:     "hacker_code",
		Position: config.PositionConfig{W: 128, H: 40},
		HackerCode: &config.HackerCodeConfig{
			TypingSpeed: &config.IntOrRange{Min: 500, Max: 10}, // Inverted range
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Verify min and max are swapped
	if w.typingSpeedMin != 10 || w.typingSpeedMax != 500 {
		t.Errorf("typingSpeed = {%d, %d}, want {10, 500}", w.typingSpeedMin, w.typingSpeedMax)
	}
}

func TestWidget_WrapLine(t *testing.T) {
	// Create widget with known dimensions
	cfg := config.WidgetConfig{
		Type:     "hacker_code",
		Position: config.PositionConfig{W: 30, H: 40}, // Small width to force wrapping
		Text: &config.TextConfig{
			Font: "5x7", // 6 chars width per char (5+1 spacing)
		},
	}

	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// With width 30 and charWidth 6, we get 5 chars per line
	if w.maxCharsPerLine != 5 {
		t.Errorf("maxCharsPerLine = %d, want 5", w.maxCharsPerLine)
	}

	tests := []struct {
		name     string
		line     string
		wantSegs int
	}{
		{"empty line", "", 1},
		{"short line", "ABC", 1},
		{"exact fit", "ABCDE", 1},
		{"needs wrap", "ABCDEFGH", 2},
		{"long line", "ABCDEFGHIJKLMNO", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			segments := w.wrapLine(tt.line)
			if len(segments) != tt.wantSegs {
				t.Errorf("wrapLine(%q) returned %d segments, want %d: %v", tt.line, len(segments), tt.wantSegs, segments)
			}
		})
	}
}

func boolPtr(b bool) *bool {
	return &b
}
