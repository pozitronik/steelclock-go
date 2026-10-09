// Package fps displays the current framerate of the foreground game/app,
// read from RTSS (RivaTuner Statistics Server) shared memory. RTSS is the
// framerate-capture engine behind MSI Afterburner and is already used by
// many on-screen-display tools as a stable source of per-process FPS,
// avoiding the need to implement our own DirectX/OpenGL/Vulkan present hooks.
package fps

import (
	"fmt"
	"image"
	"log"
	"sync"
	"time"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
	"github.com/pozitronik/steelclock-go/internal/shared"
	"github.com/pozitronik/steelclock-go/internal/shared/render"
	"github.com/pozitronik/steelclock-go/internal/shared/util"
	"github.com/pozitronik/steelclock-go/internal/widget"
)

func init() {
	widget.Register("fps", func(cfg config.WidgetConfig) (widget.Widget, error) {
		return New(cfg)
	})
}

// Reader is the interface for reading the current framerate.
type Reader interface {
	// GetFPS returns the current framerate and process name of the tracked
	// foreground application. Returns (0, "", nil) when nothing is currently
	// being measured — not an error condition.
	GetFPS() (fps float64, processName string, err error)
	// Close releases resources.
	Close()
}

// reconnectInterval limits how often a failed/absent reader is retried, so a
// closed game or a not-yet-running RTSS doesn't cause a syscall every tick.
const reconnectInterval = 3 * time.Second

// Widget displays the current framerate
type Widget struct {
	*widget.BaseWidget
	displayMode render.DisplayMode
	strategy    render.MetricDisplayStrategy
	Renderer    *render.MetricRenderer
	textFormat  string

	reader        Reader
	lastReconnect time.Time

	currentValue float64
	history      *util.RingBuffer[float64]
	hasData      bool
	mu           sync.RWMutex
}

// New creates a new FPS widget
func New(cfg config.WidgetConfig) (*Widget, error) {
	base := widget.NewBaseWidget(cfg)
	helper := shared.NewConfigHelper(cfg)

	mr, err := helper.BuildMetricRenderer()
	if err != nil {
		return nil, err
	}
	// Bar/graph/gauge scale values as 0-100 percent, which FPS is not.
	if mr.DisplayMode != render.DisplayModeText {
		return nil, fmt.Errorf("fps widget supports only text mode, got %q", mr.DisplayMode)
	}

	textFormat := "%.0f FPS"
	if cfg.Text != nil && cfg.Text.Format != "" {
		textFormat = cfg.Text.Format
	}

	w := &Widget{
		BaseWidget:  base,
		displayMode: mr.DisplayMode,
		strategy:    mr.Strategy,
		Renderer:    mr.Renderer,
		textFormat:  textFormat,
		history:     util.NewRingBuffer[float64](mr.HistoryLen),
	}

	// Best-effort initial connection; Update() retries periodically if RTSS
	// isn't running yet (it may start later, e.g. when a game launches).
	if reader, readerErr := newReader(); readerErr == nil {
		w.reader = reader
	} else {
		log.Printf("[FPS] %v", readerErr)
	}

	return w, nil
}

// Update reads the current framerate
func (w *Widget) Update() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.reader == nil {
		if time.Since(w.lastReconnect) < reconnectInterval {
			return nil
		}
		w.lastReconnect = time.Now()
		reader, err := newReader()
		if err != nil {
			return nil // Render shows "FPS N/A"; no point logging every retry
		}
		w.reader = reader
	}

	value, _, err := w.reader.GetFPS()
	if err != nil {
		// RTSS released or replaced its segment: drop the mapping so the
		// reconnect path above reopens it once RTSS is back.
		w.reader.Close()
		w.reader = nil
		w.hasData = false
		return err
	}

	if value < 0 {
		value = 0
	}

	w.currentValue = value
	w.hasData = true
	if w.displayMode == render.DisplayModeGraph {
		w.history.Push(value)
	}

	return nil
}

// Render creates an image of the FPS widget
func (w *Widget) Render() (image.Image, error) {
	img := w.CreateCanvas()
	w.ApplyBorder(img)

	content := w.GetContentArea()
	pos := w.GetPosition()

	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.reader == nil {
		bitmap.DrawAlignedInternalText(img, "FPS N/A", nil, "center", "center", 0)
		return img, nil
	}

	if !w.hasData {
		return img, nil
	}

	w.strategy.Render(img, render.MetricData{
		Value:       w.currentValue,
		History:     w.history.ToSlice(),
		TextFormat:  w.textFormat,
		ContentArea: image.Rect(content.X, content.Y, content.X+content.Width, content.Y+content.Height),
		GaugeArea:   image.Rect(0, 0, pos.W, pos.H),
	}, w.Renderer)

	return img, nil
}

// Stop releases resources
func (w *Widget) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.reader != nil {
		w.reader.Close()
		w.reader = nil
	}
}
