//go:build linux

package hotplug

import (
	"testing"
	"time"
)

func TestStart_SurvivesIdleTimeoutsAndStops(t *testing.T) {
	w, err := Start(0x1038, func() {})
	if err != nil {
		t.Skipf("kernel uevents unavailable here: %v", err)
	}

	// Let the reader go through at least one idle receive timeout.
	time.Sleep(readTimeout + 200*time.Millisecond)

	start := time.Now()
	w.Stop()
	if took := time.Since(start); took > 2*readTimeout {
		t.Errorf("Stop() took %v, want at most about one receive timeout", took)
	}
}
