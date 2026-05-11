//go:build darwin

package volumemeter

import (
	"log"
	"sync"

	"github.com/pozitronik/steelclock-go/internal/coreaudio"
)

// DarwinMeterReader reads audio meter data on macOS using CoreAudio aggregate device tap
type DarwinMeterReader struct {
	mu  sync.Mutex
	tap *coreaudio.AudioTap
}

// NewDarwinMeterReader creates a new macOS meter reader
func NewDarwinMeterReader() (*DarwinMeterReader, error) {
	tap, err := coreaudio.NewAudioTap()
	if err != nil {
		log.Printf("[METER-DARWIN] Failed to create audio tap: %v", err)
		// Return reader without tap - will return silence
		return &DarwinMeterReader{}, nil
	}

	log.Printf("[METER-DARWIN] Using CoreAudio aggregate device tap")
	return &DarwinMeterReader{tap: tap}, nil
}

// GetMeterData reads current audio meter values from the tap's peak levels
func (r *DarwinMeterReader) GetMeterData(clippingThreshold, silenceThreshold float64) (*MeterData, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.tap == nil || !r.tap.IsRunning() {
		return &MeterData{
			Peak:         0,
			ChannelPeaks: []float64{0, 0},
			ChannelCount: 2,
			IsClipping:   false,
			HasAudio:     false,
		}, nil
	}

	left, right := r.tap.GetPeakLevels()
	peak := left
	if right > peak {
		peak = right
	}

	return &MeterData{
		Peak:         peak,
		ChannelPeaks: []float64{left, right},
		ChannelCount: 2,
		IsClipping:   peak >= clippingThreshold,
		HasAudio:     peak > silenceThreshold,
	}, nil
}

// Close releases the audio tap
func (r *DarwinMeterReader) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.tap != nil {
		r.tap.Close()
		r.tap = nil
	}
}

// Reinitialize recreates the audio tap
func (r *DarwinMeterReader) Reinitialize() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.tap != nil {
		r.tap.Close()
	}

	tap, err := coreaudio.NewAudioTap()
	if err != nil {
		r.tap = nil
		return err
	}

	r.tap = tap
	log.Printf("[METER-DARWIN] Reinitialized with CoreAudio tap")
	return nil
}

// NeedsReinitialize returns true if the tap is no longer running
func (r *DarwinMeterReader) NeedsReinitialize() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.tap != nil && !r.tap.IsRunning()
}

// newMeterReader creates a platform-specific meter reader (macOS implementation)
func newMeterReader() (Reader, error) {
	return NewDarwinMeterReader()
}
