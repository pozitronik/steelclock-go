//go:build linux

package audiovisualizer

import (
	"bufio"
	"encoding/binary"
	"io"
	"log"
	"math"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// AudioCaptureLinux captures system audio using PipeWire or PulseAudio
type AudioCaptureLinux struct {
	mu           sync.Mutex
	cmd          *exec.Cmd
	stdout       io.ReadCloser
	running      bool
	sampleRate   int
	channels     int
	samplesLeft  []float32
	samplesRight []float32
	maxSamples   int
	lastError    error
	audioTool    string
}

var (
	sharedAudioCapture   *AudioCaptureLinux
	sharedAudioCaptureMu sync.Mutex
)

// GetSharedAudioCaptureLinux returns the shared audio capture instance
func GetSharedAudioCaptureLinux() (*AudioCaptureLinux, error) {
	sharedAudioCaptureMu.Lock()
	defer sharedAudioCaptureMu.Unlock()

	if sharedAudioCapture != nil && sharedAudioCapture.running {
		return sharedAudioCapture, nil
	}

	capture, err := NewAudioCaptureLinux()
	if err != nil {
		return nil, err
	}

	sharedAudioCapture = capture
	return sharedAudioCapture, nil
}

// ReinitializeSharedAudioCaptureLinux reinitializes the shared audio capture
func ReinitializeSharedAudioCaptureLinux() error {
	sharedAudioCaptureMu.Lock()
	defer sharedAudioCaptureMu.Unlock()

	if sharedAudioCapture != nil {
		sharedAudioCapture.Close()
		sharedAudioCapture = nil
	}

	capture, err := NewAudioCaptureLinux()
	if err != nil {
		return err
	}

	sharedAudioCapture = capture
	return nil
}

// NewAudioCaptureLinux creates a new audio capture instance
func NewAudioCaptureLinux() (*AudioCaptureLinux, error) {
	ac := &AudioCaptureLinux{
		sampleRate: 48000,
		channels:   2,
		maxSamples: 16384,
	}

	// Detect available audio tool
	if _, err := exec.LookPath("pw-record"); err == nil {
		ac.audioTool = "pw-record"
	} else if _, err := exec.LookPath("parec"); err == nil {
		ac.audioTool = "parec"
	} else {
		log.Println("[AUDIO-CAPTURE] No audio capture tool found (pw-record or parec)")
		return ac, nil // Return without error - will use demo mode
	}

	if err := ac.start(); err != nil {
		log.Printf("[AUDIO-CAPTURE] Failed to start capture: %v", err)
		return ac, nil // Return without error - will use demo mode
	}

	return ac, nil
}

// findDefaultSinkMonitor finds the default audio sink ID for PipeWire
func findDefaultSinkMonitor() string {
	// Use wpctl to find the default sink ID
	cmd := exec.Command("wpctl", "inspect", "@DEFAULT_AUDIO_SINK@")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	// Parse output to find the object id
	// Format: "id 57, type PipeWire:Interface:Node"
	lines := string(output)
	for _, line := range strings.Split(lines, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "id ") {
			// Extract ID number (format: "id 57,")
			parts := strings.Split(line, ",")
			if len(parts) >= 1 {
				idPart := strings.TrimPrefix(parts[0], "id ")
				idPart = strings.TrimSpace(idPart)
				return idPart
			}
		}
	}
	return ""
}

// start begins the audio capture process
func (ac *AudioCaptureLinux) start() error {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	if ac.running {
		return nil
	}

	var cmd *exec.Cmd

	switch ac.audioTool {
	case "pw-record":
		// PipeWire: to capture system audio output (loopback), we need to:
		// 1. Find the default sink's monitor
		// 2. Record from it with raw output (no header)
		sinkID := findDefaultSinkMonitor()
		if sinkID != "" {
			log.Printf("[AUDIO-CAPTURE] Found default sink ID: %s", sinkID)
		}

		// pw-record with --target pointing to the sink captures from its monitor
		// --raw flag outputs raw samples without header
		args := []string{
			"--raw",
			"--rate", "48000",
			"--channels", "2",
			"--format", "f32",
		}
		if sinkID != "" {
			args = append(args, "--target", sinkID)
		}
		args = append(args, "-")

		cmd = exec.Command("pw-record", args...)
	case "parec":
		// PulseAudio: capture from monitor
		cmd = exec.Command("parec",
			"--rate=48000",
			"--channels=2",
			"--format=float32le",
			"--device=@DEFAULT_MONITOR@")
	default:
		return nil
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	ac.cmd = cmd
	ac.stdout = stdout
	ac.running = true
	ac.samplesLeft = make([]float32, 0, ac.maxSamples)
	ac.samplesRight = make([]float32, 0, ac.maxSamples)

	// Start reading audio data in background
	go ac.readLoop()

	log.Printf("[AUDIO-CAPTURE] Started using %s", ac.audioTool)
	return nil
}

// readLoop continuously reads audio data from the capture process
func (ac *AudioCaptureLinux) readLoop() {
	reader := bufio.NewReaderSize(ac.stdout, 32768)
	sampleBuf := make([]byte, 8) // 2 channels * 4 bytes per float32

	for {
		ac.mu.Lock()
		if !ac.running {
			ac.mu.Unlock()
			return
		}
		ac.mu.Unlock()

		// Read one stereo sample (2 x float32 = 8 bytes)
		_, err := io.ReadFull(reader, sampleBuf)
		if err != nil {
			if err != io.EOF {
				ac.mu.Lock()
				ac.lastError = err
				ac.mu.Unlock()
			}
			break
		}

		// Parse float32 samples (little-endian)
		leftSample := float32frombits(binary.LittleEndian.Uint32(sampleBuf[0:4]))
		rightSample := float32frombits(binary.LittleEndian.Uint32(sampleBuf[4:8]))

		ac.mu.Lock()
		ac.samplesLeft = append(ac.samplesLeft, leftSample)
		ac.samplesRight = append(ac.samplesRight, rightSample)

		// Trim to max size
		if len(ac.samplesLeft) > ac.maxSamples {
			ac.samplesLeft = ac.samplesLeft[len(ac.samplesLeft)-ac.maxSamples:]
			ac.samplesRight = ac.samplesRight[len(ac.samplesRight)-ac.maxSamples:]
		}
		ac.mu.Unlock()
	}

	ac.mu.Lock()
	ac.running = false
	ac.mu.Unlock()
}

// float32frombits converts uint32 bits to float32
func float32frombits(b uint32) float32 {
	return math.Float32frombits(b)
}

// ReadSamples returns the current audio samples
func (ac *AudioCaptureLinux) ReadSamples() (left, right []float32, err error) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	if !ac.running || len(ac.samplesLeft) == 0 {
		return nil, nil, nil
	}

	// Return copies of the sample buffers
	left = make([]float32, len(ac.samplesLeft))
	right = make([]float32, len(ac.samplesRight))
	copy(left, ac.samplesLeft)
	copy(right, ac.samplesRight)

	return left, right, nil
}

// GetRecentSamples returns the most recent N samples
func (ac *AudioCaptureLinux) GetRecentSamples(count int) (left, right []float32) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	if len(ac.samplesLeft) == 0 {
		return nil, nil
	}

	if count > len(ac.samplesLeft) {
		count = len(ac.samplesLeft)
	}

	start := len(ac.samplesLeft) - count
	left = make([]float32, count)
	right = make([]float32, count)
	copy(left, ac.samplesLeft[start:])
	copy(right, ac.samplesRight[start:])

	return left, right
}

// IsRunning returns true if audio capture is active
func (ac *AudioCaptureLinux) IsRunning() bool {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	return ac.running
}

// SampleRate returns the capture sample rate
func (ac *AudioCaptureLinux) SampleRate() int {
	return ac.sampleRate
}

// Close stops the audio capture
func (ac *AudioCaptureLinux) Close() {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	if !ac.running {
		return
	}

	ac.running = false

	if ac.cmd != nil && ac.cmd.Process != nil {
		_ = ac.cmd.Process.Kill()
		_ = ac.cmd.Wait()
	}

	if ac.stdout != nil {
		ac.stdout.Close()
	}

	log.Println("[AUDIO-CAPTURE] Stopped")
}

// WaitForSamples waits until at least minSamples are available or timeout
func (ac *AudioCaptureLinux) WaitForSamples(minSamples int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		ac.mu.Lock()
		count := len(ac.samplesLeft)
		ac.mu.Unlock()

		if count >= minSamples {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}
