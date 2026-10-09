package fps

import (
	"fmt"
	"testing"

	"github.com/pozitronik/steelclock-go/internal/config"
)

// mockReader is a test implementation of Reader.
type mockReader struct {
	fpsValue  float64
	procName  string
	returnErr error
	closed    bool
}

func (m *mockReader) GetFPS() (float64, string, error) {
	if m.returnErr != nil {
		return 0, "", m.returnErr
	}
	return m.fpsValue, m.procName, nil
}

func (m *mockReader) Close() { m.closed = true }

func newTestWidget(t *testing.T) *Widget {
	t.Helper()
	cfg := config.WidgetConfig{
		Type:    "fps",
		ID:      "test_fps",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 20,
		},
		Mode: "text",
	}
	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return w
}

func TestNew_DefaultTextFormat(t *testing.T) {
	w := newTestWidget(t)
	if w.textFormat != "%.0f FPS" {
		t.Errorf("textFormat = %q, want %q", w.textFormat, "%.0f FPS")
	}
}

func TestNew_ReadsTextFormatFromConfig(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "fps",
		ID:      "test_fps_format",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 20,
		},
		Mode: "text",
		Text: &config.TextConfig{Format: "%.1f fps"},
	}
	w, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if w.textFormat != "%.1f fps" {
		t.Errorf("textFormat = %q, want %q", w.textFormat, "%.1f fps")
	}
}

func TestWidget_Update_WithMockReader(t *testing.T) {
	w := newTestWidget(t)
	w.reader = &mockReader{fpsValue: 144.0, procName: "game.exe"}

	if err := w.Update(); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	w.mu.RLock()
	val := w.currentValue
	hasData := w.hasData
	w.mu.RUnlock()

	if !hasData {
		t.Error("hasData = false, want true after Update()")
	}
	if val != 144.0 {
		t.Errorf("currentValue = %v, want 144.0", val)
	}
}

func TestWidget_Update_NegativeClamped(t *testing.T) {
	w := newTestWidget(t)
	w.reader = &mockReader{fpsValue: -5.0}

	if err := w.Update(); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	w.mu.RLock()
	val := w.currentValue
	w.mu.RUnlock()

	if val != 0 {
		t.Errorf("currentValue = %v, want 0 (clamped)", val)
	}
}

func TestWidget_Update_ReaderError(t *testing.T) {
	w := newTestWidget(t)
	w.reader = &mockReader{returnErr: fmt.Errorf("simulated RTSS read error")}

	if err := w.Update(); err == nil {
		t.Error("Update() should return error when reader fails")
	}
}

// TestWidget_Update_ReaderError_DropsReader verifies that a failing reader
// (e.g. RTSS having released its shared memory segment) is dropped so the
// reconnect path in Update() can reopen it, rather than being retried
// forever with the same broken mapping.
func TestWidget_Update_ReaderError_DropsReader(t *testing.T) {
	w := newTestWidget(t)
	mock := &mockReader{returnErr: fmt.Errorf("simulated RTSS segment gone")}
	w.reader = mock

	if err := w.Update(); err == nil {
		t.Fatal("Update() should return error when reader fails")
	}

	if !mock.closed {
		t.Error("Update() should Close() a reader that failed")
	}
	if w.reader != nil {
		t.Error("Update() should drop the reader after an error so it reconnects")
	}
	w.mu.RLock()
	hasData := w.hasData
	w.mu.RUnlock()
	if hasData {
		t.Error("hasData should be reset to false after a reader error")
	}
}

func TestNew_RejectsNonTextMode(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "fps",
		ID:      "test_fps_bar",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 20,
		},
		Mode: "bar",
	}

	if _, err := New(cfg); err == nil {
		t.Error("New() should reject non-text display modes (bar/graph/gauge scale as 0-100%, FPS is not)")
	}
}

func TestWidget_Render_NoReader_ShowsNA(t *testing.T) {
	w := newTestWidget(t)
	w.reader = nil

	img, err := w.Render()
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if img == nil {
		t.Error("Render() returned nil image, want an \"FPS N/A\" placeholder image")
	}
}

func TestWidget_Render_BeforeUpdate_NoPanic(t *testing.T) {
	w := newTestWidget(t)
	w.reader = &mockReader{fpsValue: 60.0}

	if _, err := w.Render(); err != nil {
		t.Errorf("Render() before Update() error = %v", err)
	}
}

func TestWidget_Stop_ClosesReader(t *testing.T) {
	w := newTestWidget(t)
	mock := &mockReader{fpsValue: 60.0}
	w.reader = mock

	w.Stop()

	if !mock.closed {
		t.Error("Stop() did not close the reader")
	}
}
