package render

import (
	"image"
	"testing"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
)

func TestTransposeHistory(t *testing.T) {
	t.Run("normal 3x2", func(t *testing.T) {
		// 3 time samples, 2 items each
		input := [][]float64{
			{10, 20},
			{30, 40},
			{50, 60},
		}
		got := TransposeHistory(input)
		if len(got) != 2 {
			t.Fatalf("got %d items, want 2", len(got))
		}
		// Item 0: [10, 30, 50]
		want0 := []float64{10, 30, 50}
		for i, v := range got[0] {
			if v != want0[i] {
				t.Errorf("got[0][%d] = %.0f, want %.0f", i, v, want0[i])
			}
		}
		// Item 1: [20, 40, 60]
		want1 := []float64{20, 40, 60}
		for i, v := range got[1] {
			if v != want1[i] {
				t.Errorf("got[1][%d] = %.0f, want %.0f", i, v, want1[i])
			}
		}
	})

	t.Run("single time sample returns nil", func(t *testing.T) {
		got := TransposeHistory([][]float64{{10, 20}})
		if got != nil {
			t.Errorf("expected nil for single sample, got %v", got)
		}
	})

	t.Run("empty input returns nil", func(t *testing.T) {
		got := TransposeHistory([][]float64{})
		if got != nil {
			t.Errorf("expected nil for empty input, got %v", got)
		}
	})

	t.Run("nil input returns nil", func(t *testing.T) {
		got := TransposeHistory(nil)
		if got != nil {
			t.Errorf("expected nil for nil input, got %v", got)
		}
	})

	t.Run("ragged rows", func(t *testing.T) {
		// Second row has fewer items - should not panic, missing values stay 0
		input := [][]float64{
			{10, 20, 30},
			{40, 50},
		}
		got := TransposeHistory(input)
		if len(got) != 3 {
			t.Fatalf("got %d items, want 3", len(got))
		}
		// Item 2: [30, 0] (second row had no index 2)
		if got[2][0] != 30 {
			t.Errorf("got[2][0] = %.0f, want 30", got[2][0])
		}
		if got[2][1] != 0 {
			t.Errorf("got[2][1] = %.0f, want 0 (missing)", got[2][1])
		}
	})
}

func TestGetGridMetricStrategy(t *testing.T) {
	tests := []struct {
		mode DisplayMode
	}{
		{DisplayModeText},
		{DisplayModeBar},
		{DisplayModeGraph},
		{DisplayModeGauge},
		{DisplayMode("unknown")}, // fallback
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			strategy := GetGridMetricStrategy(tt.mode)
			if strategy == nil {
				t.Fatal("GetGridMetricStrategy returned nil")
			}
			// Verify singleton behavior
			strategy2 := GetGridMetricStrategy(tt.mode)
			if strategy != strategy2 {
				t.Error("GetGridMetricStrategy should return singleton instances")
			}
		})
	}
}

func TestCalculateGridLayout(t *testing.T) {
	tests := []struct {
		name        string
		numCells    int
		totalWidth  int
		totalHeight int
		margin      int
		wantCols    int
		wantRows    int
	}{
		{"empty", 0, 100, 100, 2, 0, 0},
		{"single cell", 1, 100, 100, 2, 1, 1},
		{"4 cells", 4, 100, 100, 2, 2, 2},
		{"8 cells", 8, 100, 100, 2, 3, 3},
		{"9 cells", 9, 100, 100, 2, 3, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cols, rows, _, _ := calculateGridLayout(tt.numCells, tt.totalWidth, tt.totalHeight, tt.margin)
			if cols != tt.wantCols {
				t.Errorf("cols = %d, want %d", cols, tt.wantCols)
			}
			if rows != tt.wantRows {
				t.Errorf("rows = %d, want %d", rows, tt.wantRows)
			}
		})
	}
}

func createGridTestRenderer() *MetricRenderer {
	return NewMetricRenderer(
		BarConfig{Direction: config.DirectionHorizontal, Border: true, Color: 255},
		GraphConfig{FillColor: 128, LineColor: 255, HistoryLen: 30},
		GaugeConfig{ArcColor: 200, NeedleColor: 255, ShowTicks: false},
		TextConfig{HorizAlign: config.AlignCenter, VertAlign: config.AlignMiddle, Padding: 0},
	)
}

func TestGridTextDisplayStrategy_Render(t *testing.T) {
	img := bitmap.NewGrayscaleImage(64, 40, 0)
	renderer := createGridTestRenderer()
	strategy := &GridTextDisplayStrategy{}

	data := GridMetricData{
		Values:      []float64{25.0, 50.0, 75.0, 100.0},
		Position:    config.PositionConfig{X: 0, Y: 0, W: 64, H: 40},
		CoreBorder:  true,
		CoreMargin:  2,
		BorderColor: 255,
	}
	// Should not panic
	strategy.Render(img, data, renderer)
}

func TestGridTextDisplayStrategy_EmptyValues(t *testing.T) {
	img := bitmap.NewGrayscaleImage(64, 40, 0)
	renderer := createGridTestRenderer()
	strategy := &GridTextDisplayStrategy{}

	data := GridMetricData{
		Values:   []float64{},
		Position: config.PositionConfig{X: 0, Y: 0, W: 64, H: 40},
	}
	// Should not panic with empty values
	strategy.Render(img, data, renderer)
}

func TestGridBarDisplayStrategy_Render(t *testing.T) {
	img := bitmap.NewGrayscaleImage(64, 40, 0)
	renderer := createGridTestRenderer()
	strategy := &GridBarDisplayStrategy{}

	t.Run("horizontal bars", func(t *testing.T) {
		data := GridMetricData{
			Values:      []float64{25.0, 50.0, 75.0, 100.0},
			ContentArea: image.Rect(2, 2, 62, 38),
			CoreMargin:  2,
		}
		// Should not panic
		strategy.Render(img, data, renderer)
	})

	t.Run("vertical bars", func(t *testing.T) {
		vertRenderer := NewMetricRenderer(
			BarConfig{Direction: config.DirectionVertical, Border: true, Color: 255},
			GraphConfig{FillColor: 128, LineColor: 255, HistoryLen: 30},
			GaugeConfig{ArcColor: 200, NeedleColor: 255, ShowTicks: false},
			TextConfig{HorizAlign: config.AlignCenter, VertAlign: config.AlignMiddle, Padding: 0},
		)
		data := GridMetricData{
			Values:      []float64{25.0, 50.0, 75.0, 100.0},
			ContentArea: image.Rect(2, 2, 62, 38),
			CoreMargin:  2,
		}
		// Should not panic
		strategy.Render(img, data, vertRenderer)
	})
}

func TestGridGraphDisplayStrategy_Render(t *testing.T) {
	img := bitmap.NewGrayscaleImage(64, 40, 0)
	renderer := createGridTestRenderer()
	strategy := &GridGraphDisplayStrategy{}

	t.Run("with sufficient history", func(t *testing.T) {
		// History format: [cell][time]
		data := GridMetricData{
			History: [][]float64{
				{10, 20, 30, 40, 50},
				{15, 25, 35, 45, 55},
				{20, 30, 40, 50, 60},
				{25, 35, 45, 55, 65},
			},
			ContentArea: image.Rect(2, 2, 62, 38),
			CoreBorder:  true,
			CoreMargin:  2,
			BorderColor: 200,
		}
		// Should not panic
		strategy.Render(img, data, renderer)
	})

	t.Run("with insufficient history", func(t *testing.T) {
		data := GridMetricData{
			History:     [][]float64{{10}}, // Only one point, need at least 2
			ContentArea: image.Rect(2, 2, 62, 38),
		}
		// Should not panic, just return early
		strategy.Render(img, data, renderer)
	})

	t.Run("with empty history", func(t *testing.T) {
		data := GridMetricData{
			History:     [][]float64{},
			ContentArea: image.Rect(2, 2, 62, 38),
		}
		// Should not panic, just return early
		strategy.Render(img, data, renderer)
	})
}

func TestGridGaugeDisplayStrategy_Render(t *testing.T) {
	img := bitmap.NewGrayscaleImage(64, 40, 0)
	renderer := createGridTestRenderer()
	strategy := &GridGaugeDisplayStrategy{}

	data := GridMetricData{
		Values:      []float64{25.0, 50.0, 75.0, 100.0},
		Position:    config.PositionConfig{X: 0, Y: 0, W: 64, H: 40},
		CoreBorder:  true,
		CoreMargin:  2,
		BorderColor: 200,
	}
	// Should not panic
	strategy.Render(img, data, renderer)
}

func TestGridGaugeDisplayStrategy_EmptyValues(t *testing.T) {
	img := bitmap.NewGrayscaleImage(64, 40, 0)
	renderer := createGridTestRenderer()
	strategy := &GridGaugeDisplayStrategy{}

	data := GridMetricData{
		Values:   []float64{},
		Position: config.PositionConfig{X: 0, Y: 0, W: 64, H: 40},
	}
	// Should not panic with empty values
	strategy.Render(img, data, renderer)
}
