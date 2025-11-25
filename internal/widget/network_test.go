package widget

import (
	"testing"

	"github.com/pozitronik/steelclock-go/internal/config"
)

// TestNewNetworkWidget tests successful network widget creation
func TestNewNetworkWidget(t *testing.T) {
	iface := "eth0"
	cfg := config.WidgetConfig{
		Type:    "network",
		ID:      "test_network",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode:         "text",
		Interface:    &iface,
		MaxSpeedMbps: 1000,
		Text: &config.TextConfig{
			Size: 10,
		},
		Colors: &config.ColorsConfig{
			Rx: config.IntPtr(255),
			Tx: config.IntPtr(200),
		},
		Graph: &config.GraphConfig{
			History: 30,
		},
	}

	widget, err := NewNetworkWidget(cfg)
	if err != nil {
		t.Fatalf("NewNetworkWidget() error = %v", err)
	}

	if widget == nil {
		t.Fatal("NewNetworkWidget() returned nil")
	}

	if widget.Name() != "test_network" {
		t.Errorf("Name() = %s, want test_network", widget.Name())
	}
}

// TestNewNetworkWidget_Defaults tests default values
func TestNewNetworkWidget_Defaults(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "network",
		ID:      "test_network_defaults",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		// Interface intentionally nil to test all interfaces mode
	}

	widget, err := NewNetworkWidget(cfg)
	if err != nil {
		t.Fatalf("NewNetworkWidget() error = %v", err)
	}

	// Check defaults
	if widget.displayMode != "text" {
		t.Errorf("default displayMode = %s, want text", widget.displayMode)
	}

	if widget.rxColor != 255 {
		t.Errorf("default rxColor = %d, want 255", widget.rxColor)
	}

	if widget.txColor != 255 {
		t.Errorf("default txColor = %d, want 255", widget.txColor)
	}

	if widget.maxSpeedMbps != -1 {
		t.Errorf("default maxSpeedMbps = %f, want -1 (auto)", widget.maxSpeedMbps)
	}
}

// TestNetworkWidget_Update tests network stat collection
func TestNetworkWidget_Update(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "network",
		ID:      "test_network_update",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode: "text",
	}

	widget, err := NewNetworkWidget(cfg)
	if err != nil {
		t.Fatalf("NewNetworkWidget() error = %v", err)
	}

	// First update initializes baseline
	err = widget.Update()
	if err != nil {
		t.Errorf("Update() error = %v", err)
	}

	// Second update calculates deltas
	err = widget.Update()
	if err != nil {
		t.Errorf("Update() error = %v", err)
	}

	// Verify stats were collected (values may be 0 in test environment)
	widget.mu.RLock()
	hasStats := widget.currentRxMbps >= 0 && widget.currentTxMbps >= 0
	widget.mu.RUnlock()

	if !hasStats {
		t.Error("Update() did not collect network stats")
	}
}

// TestNetworkWidget_RenderText tests text mode rendering
func TestNetworkWidget_RenderText(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "network",
		ID:      "test_network_text",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode: "text",
		Text: &config.TextConfig{
			Size: 10,
		},
	}

	widget, err := NewNetworkWidget(cfg)
	if err != nil {
		t.Fatalf("NewNetworkWidget() error = %v", err)
	}

	// Update to collect stats
	_ = widget.Update()

	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}

	if img == nil {
		t.Fatal("Render() returned nil image")
	}
}

// TestNetworkWidget_RenderBarHorizontal tests horizontal bar rendering
func TestNetworkWidget_RenderBarHorizontal(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "network",
		ID:      "test_network_bar_h",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode:         "bar_horizontal",
		MaxSpeedMbps: 100,
		Colors: &config.ColorsConfig{
			Rx: config.IntPtr(255),
			Tx: config.IntPtr(200),
		},
	}

	widget, err := NewNetworkWidget(cfg)
	if err != nil {
		t.Fatalf("NewNetworkWidget() error = %v", err)
	}

	_ = widget.Update()

	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}

	if img == nil {
		t.Fatal("Render() returned nil image")
	}
}

// TestNetworkWidget_RenderBarVertical tests vertical bar rendering
func TestNetworkWidget_RenderBarVertical(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "network",
		ID:      "test_network_bar_v",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode: "bar_vertical",
		Colors: &config.ColorsConfig{
			Rx: config.IntPtr(255),
			Tx: config.IntPtr(200),
		},
	}

	widget, err := NewNetworkWidget(cfg)
	if err != nil {
		t.Fatalf("NewNetworkWidget() error = %v", err)
	}

	_ = widget.Update()

	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}

	if img == nil {
		t.Fatal("Render() returned nil image")
	}
}

// TestNetworkWidget_RenderGraph tests graph mode rendering
func TestNetworkWidget_RenderGraph(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "network",
		ID:      "test_network_graph",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode: "graph",
		Colors: &config.ColorsConfig{
			Rx: config.IntPtr(255),
			Tx: config.IntPtr(200),
		},
		Graph: &config.GraphConfig{
			History: 30,
		},
	}

	widget, err := NewNetworkWidget(cfg)
	if err != nil {
		t.Fatalf("NewNetworkWidget() error = %v", err)
	}

	// Build history
	for i := 0; i < 5; i++ {
		_ = widget.Update()
	}

	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}

	if img == nil {
		t.Fatal("Render() returned nil image")
	}
}

// TestNetworkWidget_AutoScale tests auto-scaling when maxSpeedMbps is -1
func TestNetworkWidget_AutoScale(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "network",
		ID:      "test_network_autoscale",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode:         "bar_horizontal",
		MaxSpeedMbps: -1, // Auto-scale
	}

	widget, err := NewNetworkWidget(cfg)
	if err != nil {
		t.Fatalf("NewNetworkWidget() error = %v", err)
	}

	_ = widget.Update()

	// Should render without error even with auto-scale
	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() with auto-scale error = %v", err)
	}

	if img == nil {
		t.Fatal("Render() returned nil image")
	}
}

// TestNetworkWidget_SpecificInterface tests monitoring specific interface
func TestNetworkWidget_SpecificInterface(t *testing.T) {
	// Use a likely non-existent interface to test error handling
	iface := "nonexistent_interface_xyz123"
	cfg := config.WidgetConfig{
		Type:    "network",
		ID:      "test_network_specific",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode:      "text",
		Interface: &iface,
	}

	widget, err := NewNetworkWidget(cfg)
	if err != nil {
		t.Fatalf("NewNetworkWidget() error = %v", err)
	}

	// Update should work even if interface doesn't exist (returns 0 stats)
	err = widget.Update()
	if err != nil {
		t.Errorf("Update() with nonexistent interface should not error, got %v", err)
	}
}

// TestNetworkWidget_RenderGauge tests dual gauge mode rendering
func TestNetworkWidget_RenderGauge(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "network",
		ID:      "test_network_gauge",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode:         "gauge",
		MaxSpeedMbps: 100,
		Colors: &config.ColorsConfig{
			Rx:       config.IntPtr(255),
			Tx:       config.IntPtr(200),
			RxNeedle: config.IntPtr(255),
			TxNeedle: config.IntPtr(180),
		},
	}

	widget, err := NewNetworkWidget(cfg)
	if err != nil {
		t.Fatalf("NewNetworkWidget() error = %v", err)
	}

	_ = widget.Update()

	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}

	if img == nil {
		t.Fatal("Render() returned nil image")
	}
}

// TestNetworkWidget_GaugeDefaults tests gauge mode with default needle colors
func TestNetworkWidget_GaugeDefaults(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "network",
		ID:      "test_network_gauge_defaults",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 64, H: 40,
		},
		Mode:         "gauge",
		MaxSpeedMbps: 100,
		// Don't specify needle colors to test defaults
	}

	widget, err := NewNetworkWidget(cfg)
	if err != nil {
		t.Fatalf("NewNetworkWidget() error = %v", err)
	}

	// Verify defaults
	if widget.rxNeedleColor != 255 {
		t.Errorf("default rxNeedleColor = %d, want 255", widget.rxNeedleColor)
	}

	if widget.txNeedleColor != 200 {
		t.Errorf("default txNeedleColor = %d, want 200", widget.txNeedleColor)
	}

	_ = widget.Update()

	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() error = %v", err)
	}

	if img == nil {
		t.Fatal("Render() returned nil image")
	}
}

// TestNetworkWidget_GaugeAutoScale tests gauge mode with auto-scaling
func TestNetworkWidget_GaugeAutoScale(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "network",
		ID:      "test_network_gauge_autoscale",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 64, H: 40,
		},
		Mode:         "gauge",
		MaxSpeedMbps: -1, // Auto-scale
		Colors: &config.ColorsConfig{
			Rx: config.IntPtr(255),
			Tx: config.IntPtr(200),
		},
	}

	widget, err := NewNetworkWidget(cfg)
	if err != nil {
		t.Fatalf("NewNetworkWidget() error = %v", err)
	}

	_ = widget.Update()

	// Should render without error even with auto-scale
	img, err := widget.Render()
	if err != nil {
		t.Errorf("Render() with auto-scale error = %v", err)
	}

	if img == nil {
		t.Fatal("Render() returned nil image")
	}
}

// TestNetworkWidget_ConcurrentAccess tests thread safety
func TestNetworkWidget_ConcurrentAccess(t *testing.T) {
	cfg := config.WidgetConfig{
		Type:    "network",
		ID:      "test_network_concurrent",
		Enabled: config.BoolPtr(true),
		Position: config.PositionConfig{
			X: 0, Y: 0, W: 128, H: 40,
		},
		Mode: "text",
	}

	widget, err := NewNetworkWidget(cfg)
	if err != nil {
		t.Fatalf("NewNetworkWidget() error = %v", err)
	}

	done := make(chan bool)

	// Concurrent updates
	go func() {
		for i := 0; i < 10; i++ {
			_ = widget.Update()
		}
		done <- true
	}()

	// Concurrent renders
	go func() {
		for i := 0; i < 10; i++ {
			_, _ = widget.Render()
		}
		done <- true
	}()

	<-done
	<-done
	// Should not panic or race
}
