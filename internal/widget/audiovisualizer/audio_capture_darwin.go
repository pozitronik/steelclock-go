//go:build darwin

package audiovisualizer

import (
	"log"
	"sync"

	"github.com/pozitronik/steelclock-go/internal/coreaudio"
)

// AudioCaptureDarwin captures system audio on macOS using CoreAudio aggregate device tap
type AudioCaptureDarwin struct {
	tap *coreaudio.AudioTap
}

var (
	sharedAudioCapture   *AudioCaptureDarwin
	sharedAudioCaptureMu sync.Mutex
)

// GetSharedAudioCaptureDarwin returns the shared audio capture instance
func GetSharedAudioCaptureDarwin() (*AudioCaptureDarwin, error) {
	sharedAudioCaptureMu.Lock()
	defer sharedAudioCaptureMu.Unlock()

	if sharedAudioCapture != nil && sharedAudioCapture.IsRunning() {
		return sharedAudioCapture, nil
	}

	capture, err := NewAudioCaptureDarwin()
	if err != nil {
		return nil, err
	}

	sharedAudioCapture = capture
	return sharedAudioCapture, nil
}

// ReinitializeSharedAudioCaptureDarwin reinitializes the shared audio capture
func ReinitializeSharedAudioCaptureDarwin() error {
	sharedAudioCaptureMu.Lock()
	defer sharedAudioCaptureMu.Unlock()

	if sharedAudioCapture != nil {
		sharedAudioCapture.Close()
		sharedAudioCapture = nil
	}

	capture, err := NewAudioCaptureDarwin()
	if err != nil {
		return err
	}

	sharedAudioCapture = capture
	return nil
}

// NewAudioCaptureDarwin creates a new audio capture instance using CoreAudio tap
func NewAudioCaptureDarwin() (*AudioCaptureDarwin, error) {
	tap, err := coreaudio.NewAudioTap()
	if err != nil {
		log.Printf("[AUDIO-CAPTURE-DARWIN] Failed to create audio tap: %v", err)
		return &AudioCaptureDarwin{}, nil // Return without error - will use demo mode
	}

	log.Printf("[AUDIO-CAPTURE-DARWIN] Audio capture started via CoreAudio tap")
	return &AudioCaptureDarwin{tap: tap}, nil
}

// ReadSamples returns available audio samples
func (ac *AudioCaptureDarwin) ReadSamples() (left, right []float32, err error) {
	if ac.tap == nil {
		return nil, nil, nil
	}
	return ac.tap.ReadSamples()
}

// GetRecentSamples returns the most recent N samples
func (ac *AudioCaptureDarwin) GetRecentSamples(count int) (left, right []float32) {
	if ac.tap == nil {
		return nil, nil
	}
	return ac.tap.GetRecentSamples(count)
}

// IsRunning returns true if audio capture is active
func (ac *AudioCaptureDarwin) IsRunning() bool {
	if ac.tap == nil {
		return false
	}
	return ac.tap.IsRunning()
}

// SampleRate returns the capture sample rate
func (ac *AudioCaptureDarwin) SampleRate() int {
	if ac.tap == nil {
		return 48000 // default
	}
	return ac.tap.SampleRate()
}

// Close stops the audio capture
func (ac *AudioCaptureDarwin) Close() {
	if ac.tap != nil {
		ac.tap.Close()
		ac.tap = nil
	}
	log.Println("[AUDIO-CAPTURE-DARWIN] Stopped")
}
