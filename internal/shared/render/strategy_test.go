package render

import (
	"image"
	"testing"

	"github.com/pozitronik/steelclock-go/internal/bitmap"
	"github.com/pozitronik/steelclock-go/internal/config"
)

func createStrategyTestRenderer() *MetricRenderer {
	return NewMetricRenderer(
		BarConfig{Direction: config.DirectionHorizontal, Border: true, Color: 255},
		GraphConfig{FillColor: 128, LineColor: 255, HistoryLen: 30},
		GaugeConfig{ArcColor: 200, NeedleColor: 255, ShowTicks: false},
		TextConfig{HorizAlign: config.AlignCenter, VertAlign: config.AlignMiddle, Padding: 0},
	)
}

func createStrategyTestDualRenderer() *DualMetricRenderer {
	return NewDualMetricRenderer(
		DualBarConfig{Direction: config.DirectionHorizontal, Border: true, PrimaryColor: 255, SecondaryColor: 128},
		DualGraphConfig{HistoryLen: 30, PrimaryFill: 200, PrimaryLine: 255, SecondaryFill: 100, SecondaryLine: 200},
		DualGaugeConfig{PrimaryArcColor: 200, PrimaryNeedleColor: 255, SecondaryArcColor: 150, SecondaryNeedleColor: 200},
		TextConfig{HorizAlign: config.AlignCenter, VertAlign: config.AlignMiddle, Padding: 0},
	)
}

func TestGetMetricStrategy(t *testing.T) {
	tests := []struct {
		mode         DisplayMode
		expectedType string
	}{
		{DisplayModeText, "*shared.TextDisplayStrategy"},
		{DisplayModeBar, "*shared.BarDisplayStrategy"},
		{DisplayModeGraph, "*shared.GraphDisplayStrategy"},
		{DisplayModeGauge, "*shared.GaugeDisplayStrategy"},
		{DisplayMode("unknown"), "*shared.TextDisplayStrategy"}, // fallback
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			strategy := GetMetricStrategy(tt.mode)
			if strategy == nil {
				t.Fatal("GetMetricStrategy returned nil")
			}
			// Verify singleton behavior
			strategy2 := GetMetricStrategy(tt.mode)
			if strategy != strategy2 {
				t.Error("GetMetricStrategy should return singleton instances")
			}
		})
	}
}

func TestGetDualMetricStrategy(t *testing.T) {
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
			strategy := GetDualMetricStrategy(tt.mode)
			if strategy == nil {
				t.Fatal("GetDualMetricStrategy returned nil")
			}
			// Verify singleton behavior
			strategy2 := GetDualMetricStrategy(tt.mode)
			if strategy != strategy2 {
				t.Error("GetDualMetricStrategy should return singleton instances")
			}
		})
	}
}

func TestTextDisplayStrategy_Render(t *testing.T) {
	img := bitmap.NewGrayscaleImage(64, 32, 0)
	renderer := createStrategyTestRenderer()
	strategy := &TextDisplayStrategy{}

	t.Run("with custom format", func(t *testing.T) {
		data := MetricData{
			Value:      75.5,
			TextFormat: "%.1f%%",
		}
		// Should not panic
		strategy.Render(img, data, renderer)
	})

	t.Run("with empty format uses default", func(t *testing.T) {
		data := MetricData{
			Value:      50.0,
			TextFormat: "",
		}
		// Should not panic and use "%.0f" format
		strategy.Render(img, data, renderer)
	})
}

func TestBarDisplayStrategy_Render(t *testing.T) {
	img := bitmap.NewGrayscaleImage(64, 32, 0)
	renderer := createStrategyTestRenderer()
	strategy := &BarDisplayStrategy{}

	data := MetricData{
		Value:       65.0,
		ContentArea: image.Rect(2, 2, 62, 30),
	}
	// Should not panic
	strategy.Render(img, data, renderer)
}

func TestGraphDisplayStrategy_Render(t *testing.T) {
	img := bitmap.NewGrayscaleImage(64, 32, 0)
	renderer := createStrategyTestRenderer()
	strategy := &GraphDisplayStrategy{}

	data := MetricData{
		History:     []float64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100},
		ContentArea: image.Rect(2, 2, 62, 30),
	}
	// Should not panic
	strategy.Render(img, data, renderer)
}

func TestGaugeDisplayStrategy_Render(t *testing.T) {
	img := bitmap.NewGrayscaleImage(64, 40, 0)
	renderer := createStrategyTestRenderer()
	strategy := &GaugeDisplayStrategy{}

	data := MetricData{
		Value:     80.0,
		GaugeArea: image.Rect(0, 0, 64, 40),
	}
	// Should not panic
	strategy.Render(img, data, renderer)
}

func TestDualTextDisplayStrategy_Render(t *testing.T) {
	img := bitmap.NewGrayscaleImage(64, 32, 0)
	renderer := createStrategyTestDualRenderer()
	strategy := &DualTextDisplayStrategy{}

	t.Run("with custom format", func(t *testing.T) {
		data := DualMetricData{
			PrimaryValue:   100.5,
			SecondaryValue: 50.2,
			TextFormat:     "D:%.0f U:%.0f",
		}
		// Should not panic
		strategy.Render(img, data, renderer)
	})

	t.Run("with empty format uses default", func(t *testing.T) {
		data := DualMetricData{
			PrimaryValue:   75.0,
			SecondaryValue: 25.0,
			TextFormat:     "",
		}
		// Should not panic and use "%.0f/%.0f" format
		strategy.Render(img, data, renderer)
	})
}

func TestDualBarDisplayStrategy_Render(t *testing.T) {
	img := bitmap.NewGrayscaleImage(64, 32, 0)
	renderer := createStrategyTestDualRenderer()
	strategy := &DualBarDisplayStrategy{}

	data := DualMetricData{
		PrimaryValue:   70.0,
		SecondaryValue: 30.0,
		ContentArea:    image.Rect(2, 2, 62, 30),
	}
	// Should not panic
	strategy.Render(img, data, renderer)
}

func TestDualGraphDisplayStrategy_Render(t *testing.T) {
	img := bitmap.NewGrayscaleImage(64, 32, 0)
	renderer := createStrategyTestDualRenderer()
	strategy := &DualGraphDisplayStrategy{}

	data := DualMetricData{
		PrimaryHistory:   []float64{10, 30, 50, 70, 90},
		SecondaryHistory: []float64{20, 40, 60, 80, 100},
		ContentArea:      image.Rect(2, 2, 62, 30),
	}
	// Should not panic
	strategy.Render(img, data, renderer)
}

func TestDualGaugeDisplayStrategy_Render(t *testing.T) {
	img := bitmap.NewGrayscaleImage(64, 40, 0)
	renderer := createStrategyTestDualRenderer()
	strategy := &DualGaugeDisplayStrategy{}

	data := DualMetricData{
		PrimaryValue:   80.0,
		SecondaryValue: 60.0,
		GaugeArea:      image.Rect(0, 0, 64, 40),
		WidgetWidth:    64,
		WidgetHeight:   40,
	}
	// Should not panic
	strategy.Render(img, data, renderer)
}

func TestStrategyStatelessness(t *testing.T) {
	// Strategies should be stateless and safely reusable
	img1 := bitmap.NewGrayscaleImage(64, 32, 0)
	img2 := bitmap.NewGrayscaleImage(64, 32, 0)
	renderer := createStrategyTestRenderer()

	strategy := GetMetricStrategy(DisplayModeBar)

	data1 := MetricData{Value: 25.0, ContentArea: image.Rect(0, 0, 64, 32)}
	data2 := MetricData{Value: 75.0, ContentArea: image.Rect(0, 0, 64, 32)}

	// Rendering different data with same strategy should work
	strategy.Render(img1, data1, renderer)
	strategy.Render(img2, data2, renderer)

	// Both renders should succeed without interference
}
