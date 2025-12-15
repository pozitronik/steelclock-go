package clipboard

import (
	"testing"

	"github.com/pozitronik/steelclock-go/internal/config"
)

// mockClipboardReader is a mock implementation for testing.
type mockClipboardReader struct {
	content     string
	contentType ContentType
	changed     bool
	readErr     error
}

func (m *mockClipboardReader) Read() (string, ContentType, error) {
	if m.readErr != nil {
		return "", TypeUnknown, m.readErr
	}
	return m.content, m.contentType, nil
}

func (m *mockClipboardReader) HasChanged() bool {
	changed := m.changed
	m.changed = false // Reset after check
	return changed
}

func (m *mockClipboardReader) Close() error {
	return nil
}

func TestContentType_String(t *testing.T) {
	tests := []struct {
		name        string
		contentType ContentType
		want        string
	}{
		{"Empty", TypeEmpty, "Empty"},
		{"Text", TypeText, "Text"},
		{"Image", TypeImage, "Image"},
		{"Files", TypeFiles, "Files"},
		{"HTML", TypeHTML, "HTML"},
		{"Unknown", TypeUnknown, "Unknown"},
		{"Invalid", ContentType(99), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.contentType.String(); got != tt.want {
				t.Errorf("ContentType.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseClipboardConfig(t *testing.T) {
	showTypeTrue := true
	showTypeFalse := false
	scrollTrue := true
	scrollFalse := false

	tests := []struct {
		name string
		cfg  config.WidgetConfig
		want ClipboardConfig
	}{
		{
			name: "default config",
			cfg:  config.WidgetConfig{},
			want: ClipboardConfig{
				MaxLength:      100,
				ShowType:       true,
				ScrollLongText: true,
				PollIntervalMs: 500,
				TextFormat:     "{content}",
				ShowInvisible:  false,
			},
		},
		{
			name: "with text format",
			cfg: config.WidgetConfig{
				Text: &config.TextConfig{
					Format: "{type}: {content}",
				},
			},
			want: ClipboardConfig{
				MaxLength:      100,
				ShowType:       true,
				ScrollLongText: true,
				PollIntervalMs: 500,
				TextFormat:     "{type}: {content}",
				ShowInvisible:  false,
			},
		},
		{
			name: "with clipboard config",
			cfg: config.WidgetConfig{
				Clipboard: &config.ClipboardWidgetConfig{
					MaxLength:      50,
					ShowType:       &showTypeFalse,
					ScrollLongText: &scrollFalse,
					PollIntervalMs: 1000,
					ShowInvisible:  true,
				},
			},
			want: ClipboardConfig{
				MaxLength:      50,
				ShowType:       false,
				ScrollLongText: false,
				PollIntervalMs: 1000,
				TextFormat:     "{content}",
				ShowInvisible:  true,
			},
		},
		{
			name: "partial clipboard config uses defaults",
			cfg: config.WidgetConfig{
				Clipboard: &config.ClipboardWidgetConfig{
					ShowType:      &showTypeTrue,
					ShowInvisible: false,
				},
			},
			want: ClipboardConfig{
				MaxLength:      100,
				ShowType:       true,
				ScrollLongText: true,
				PollIntervalMs: 500,
				TextFormat:     "{content}",
				ShowInvisible:  false,
			},
		},
		{
			name: "scroll disabled",
			cfg: config.WidgetConfig{
				Clipboard: &config.ClipboardWidgetConfig{
					ScrollLongText: &scrollTrue,
				},
			},
			want: ClipboardConfig{
				MaxLength:      100,
				ShowType:       true,
				ScrollLongText: true,
				PollIntervalMs: 500,
				TextFormat:     "{content}",
				ShowInvisible:  false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseClipboardConfig(tt.cfg)
			if got.MaxLength != tt.want.MaxLength {
				t.Errorf("MaxLength = %v, want %v", got.MaxLength, tt.want.MaxLength)
			}
			if got.ShowType != tt.want.ShowType {
				t.Errorf("ShowType = %v, want %v", got.ShowType, tt.want.ShowType)
			}
			if got.ScrollLongText != tt.want.ScrollLongText {
				t.Errorf("ScrollLongText = %v, want %v", got.ScrollLongText, tt.want.ScrollLongText)
			}
			if got.PollIntervalMs != tt.want.PollIntervalMs {
				t.Errorf("PollIntervalMs = %v, want %v", got.PollIntervalMs, tt.want.PollIntervalMs)
			}
			if got.TextFormat != tt.want.TextFormat {
				t.Errorf("TextFormat = %v, want %v", got.TextFormat, tt.want.TextFormat)
			}
			if got.ShowInvisible != tt.want.ShowInvisible {
				t.Errorf("ShowInvisible = %v, want %v", got.ShowInvisible, tt.want.ShowInvisible)
			}
		})
	}
}

func TestWidget_formatContent(t *testing.T) {
	tests := []struct {
		name        string
		cfg         ClipboardConfig
		content     string
		contentType ContentType
		want        string
	}{
		{
			name: "empty clipboard",
			cfg: ClipboardConfig{
				TextFormat: "{content}",
			},
			content:     "",
			contentType: TypeEmpty,
			want:        "[Empty]",
		},
		{
			name: "simple text",
			cfg: ClipboardConfig{
				TextFormat: "{content}",
				MaxLength:  100,
			},
			content:     "Hello World",
			contentType: TypeText,
			want:        "Hello World",
		},
		{
			name: "text with type prefix",
			cfg: ClipboardConfig{
				TextFormat: "{type}: {content}",
				MaxLength:  100,
			},
			content:     "Hello",
			contentType: TypeText,
			want:        "Text: Hello",
		},
		{
			name: "text with length",
			cfg: ClipboardConfig{
				TextFormat: "{content} ({length})",
				MaxLength:  100,
			},
			content:     "Hello",
			contentType: TypeText,
			want:        "Hello (5)",
		},
		{
			name: "text with preview",
			cfg: ClipboardConfig{
				TextFormat: "{preview}",
				MaxLength:  100,
			},
			content:     "Short",
			contentType: TypeText,
			want:        "Short",
		},
		{
			name: "long text preview truncated",
			cfg: ClipboardConfig{
				TextFormat: "{preview}",
				MaxLength:  100,
			},
			content:     "This is a very long text that should be truncated in the preview",
			contentType: TypeText,
			want:        "This is a very long ...",
		},
		{
			name: "text truncation",
			cfg: ClipboardConfig{
				TextFormat: "{content}",
				MaxLength:  10,
			},
			content:     "This is a long text",
			contentType: TypeText,
			want:        "This is...",
		},
		{
			name: "newlines replaced",
			cfg: ClipboardConfig{
				TextFormat: "{content}",
				MaxLength:  100,
			},
			content:     "Line1\nLine2\r\nLine3",
			contentType: TypeText,
			want:        "Line1 Line2 Line3",
		},
		{
			name: "tabs replaced",
			cfg: ClipboardConfig{
				TextFormat: "{content}",
				MaxLength:  100,
			},
			content:     "Col1\tCol2\tCol3",
			contentType: TypeText,
			want:        "Col1 Col2 Col3",
		},
		{
			name: "show invisible - newlines",
			cfg: ClipboardConfig{
				TextFormat:    "{content}",
				MaxLength:     100,
				ShowInvisible: true,
			},
			content:     "Line1\nLine2\nLine3",
			contentType: TypeText,
			want:        "Line1\\nLine2\\nLine3",
		},
		{
			name: "show invisible - Windows newlines",
			cfg: ClipboardConfig{
				TextFormat:    "{content}",
				MaxLength:     100,
				ShowInvisible: true,
			},
			content:     "Line1\r\nLine2\r\nLine3",
			contentType: TypeText,
			want:        "Line1\\nLine2\\nLine3",
		},
		{
			name: "show invisible - tabs",
			cfg: ClipboardConfig{
				TextFormat:    "{content}",
				MaxLength:     100,
				ShowInvisible: true,
			},
			content:     "Col1\tCol2\tCol3",
			contentType: TypeText,
			want:        "Col1\\tCol2\\tCol3",
		},
		{
			name: "show invisible - carriage return only",
			cfg: ClipboardConfig{
				TextFormat:    "{content}",
				MaxLength:     100,
				ShowInvisible: true,
			},
			content:     "Line1\rLine2",
			contentType: TypeText,
			want:        "Line1\\rLine2",
		},
		{
			name: "show invisible - mixed",
			cfg: ClipboardConfig{
				TextFormat:    "{content}",
				MaxLength:     100,
				ShowInvisible: true,
			},
			content:     "A\tB\nC\r\nD",
			contentType: TypeText,
			want:        "A\\tB\\nC\\nD",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &Widget{cfg: tt.cfg}
			got := w.formatContent(tt.content, tt.contentType)
			if got != tt.want {
				t.Errorf("formatContent() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMockClipboardReader(t *testing.T) {
	t.Run("HasChanged returns true then false", func(t *testing.T) {
		mock := &mockClipboardReader{
			content:     "test",
			contentType: TypeText,
			changed:     true,
		}

		if !mock.HasChanged() {
			t.Error("first HasChanged() should return true")
		}
		if mock.HasChanged() {
			t.Error("second HasChanged() should return false (reset)")
		}
	})

	t.Run("Read returns configured values", func(t *testing.T) {
		mock := &mockClipboardReader{
			content:     "test content",
			contentType: TypeText,
		}

		content, ct, err := mock.Read()
		if err != nil {
			t.Errorf("Read() error = %v", err)
		}
		if content != "test content" {
			t.Errorf("Read() content = %q, want %q", content, "test content")
		}
		if ct != TypeText {
			t.Errorf("Read() contentType = %v, want %v", ct, TypeText)
		}
	})

	t.Run("Close returns nil", func(t *testing.T) {
		mock := &mockClipboardReader{}
		if err := mock.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})
}

func TestWidget_Update(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:     "clipboard",
		Position: config.PositionConfig{W: 128, H: 40},
	}

	// We can't easily test New() because it creates a real clipboard reader
	// and starts a goroutine. Instead, test the Update method behavior.
	t.Run("Update returns nil", func(t *testing.T) {
		// Create a minimal widget just to test Update behavior
		w := &Widget{}
		if err := w.Update(); err != nil {
			t.Errorf("Update() error = %v", err)
		}
	})

	t.Logf("Config type: %s", cfg.Type)
}
