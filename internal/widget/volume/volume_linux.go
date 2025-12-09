//go:build linux

package volume

import (
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// LinuxReader reads system volume using available Linux audio tools
// Supports: wpctl (PipeWire), pactl (PulseAudio), amixer (ALSA)
type LinuxReader struct {
	mu         sync.Mutex
	audioTool  string // "wpctl", "pactl", or "amixer"
	lastVolume float64
	lastMuted  bool
}

// NewLinuxReader creates a new Linux volume reader
func NewLinuxReader() (*LinuxReader, error) {
	reader := &LinuxReader{}

	// Detect which audio tool is available
	// Priority: wpctl (PipeWire) > pactl (PulseAudio) > amixer (ALSA)
	if _, err := exec.LookPath("wpctl"); err == nil {
		// Verify wpctl works
		if out, err := exec.Command("wpctl", "get-volume", "@DEFAULT_AUDIO_SINK@").Output(); err == nil && len(out) > 0 {
			reader.audioTool = "wpctl"
			log.Printf("[VOLUME-LINUX] Using wpctl (PipeWire)")
			return reader, nil
		}
	}

	if _, err := exec.LookPath("pactl"); err == nil {
		// Verify pactl works
		if out, err := exec.Command("pactl", "get-sink-volume", "@DEFAULT_SINK@").Output(); err == nil && len(out) > 0 {
			reader.audioTool = "pactl"
			log.Printf("[VOLUME-LINUX] Using pactl (PulseAudio)")
			return reader, nil
		}
	}

	if _, err := exec.LookPath("amixer"); err == nil {
		// Verify amixer works
		if out, err := exec.Command("amixer", "get", "Master").Output(); err == nil && len(out) > 0 {
			reader.audioTool = "amixer"
			log.Printf("[VOLUME-LINUX] Using amixer (ALSA)")
			return reader, nil
		}
	}

	return nil, fmt.Errorf("no audio tool available (tried wpctl, pactl, amixer)")
}

// GetVolume reads the current master volume level (0-100) and mute status
func (r *LinuxReader) GetVolume() (volume float64, muted bool, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	switch r.audioTool {
	case "wpctl":
		return r.getVolumeWpctl()
	case "pactl":
		return r.getVolumePactl()
	case "amixer":
		return r.getVolumeAmixer()
	default:
		return 0, false, fmt.Errorf("no audio tool configured")
	}
}

// getVolumeWpctl reads volume using wpctl (PipeWire)
func (r *LinuxReader) getVolumeWpctl() (float64, bool, error) {
	// wpctl get-volume @DEFAULT_AUDIO_SINK@
	// Output: "Volume: 0.40" or "Volume: 0.40 [MUTED]"
	out, err := exec.Command("wpctl", "get-volume", "@DEFAULT_AUDIO_SINK@").Output()
	if err != nil {
		return r.lastVolume, r.lastMuted, fmt.Errorf("wpctl failed: %w", err)
	}

	output := strings.TrimSpace(string(out))

	// Check for muted
	muted := strings.Contains(output, "[MUTED]")

	// Parse volume: "Volume: 0.40" -> 40%
	re := regexp.MustCompile(`Volume:\s*([0-9.]+)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) < 2 {
		return r.lastVolume, r.lastMuted, fmt.Errorf("failed to parse wpctl output: %s", output)
	}

	vol, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return r.lastVolume, r.lastMuted, fmt.Errorf("failed to parse volume: %w", err)
	}

	// Convert from 0-1 to 0-100
	volume := vol * 100.0
	r.lastVolume = volume
	r.lastMuted = muted

	return volume, muted, nil
}

// getVolumePactl reads volume using pactl (PulseAudio)
func (r *LinuxReader) getVolumePactl() (float64, bool, error) {
	// pactl get-sink-volume @DEFAULT_SINK@
	// Output: "Volume: front-left: 26214 /  40% / -23.81 dB,   front-right: 26214 /  40% / -23.81 dB"
	out, err := exec.Command("pactl", "get-sink-volume", "@DEFAULT_SINK@").Output()
	if err != nil {
		return r.lastVolume, r.lastMuted, fmt.Errorf("pactl get-sink-volume failed: %w", err)
	}

	// Parse volume percentage
	re := regexp.MustCompile(`(\d+)%`)
	matches := re.FindStringSubmatch(string(out))
	if len(matches) < 2 {
		return r.lastVolume, r.lastMuted, fmt.Errorf("failed to parse pactl volume output")
	}

	vol, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return r.lastVolume, r.lastMuted, fmt.Errorf("failed to parse volume: %w", err)
	}

	// Check mute status
	muteOut, err := exec.Command("pactl", "get-sink-mute", "@DEFAULT_SINK@").Output()
	muted := false
	if err == nil {
		muted = strings.Contains(string(muteOut), "yes")
	}

	r.lastVolume = vol
	r.lastMuted = muted

	return vol, muted, nil
}

// getVolumeAmixer reads volume using amixer (ALSA)
func (r *LinuxReader) getVolumeAmixer() (float64, bool, error) {
	// amixer get Master
	// Output includes: "  Front Left: Playback 26214 [40%] [on]"
	out, err := exec.Command("amixer", "get", "Master").Output()
	if err != nil {
		return r.lastVolume, r.lastMuted, fmt.Errorf("amixer failed: %w", err)
	}

	output := string(out)

	// Parse volume percentage: [40%]
	reVol := regexp.MustCompile(`\[(\d+)%\]`)
	volMatches := reVol.FindStringSubmatch(output)
	if len(volMatches) < 2 {
		return r.lastVolume, r.lastMuted, fmt.Errorf("failed to parse amixer volume output")
	}

	vol, err := strconv.ParseFloat(volMatches[1], 64)
	if err != nil {
		return r.lastVolume, r.lastMuted, fmt.Errorf("failed to parse volume: %w", err)
	}

	// Check mute status: [on] or [off]
	muted := strings.Contains(output, "[off]")

	r.lastVolume = vol
	r.lastMuted = muted

	return vol, muted, nil
}

// Close releases resources (no-op for Linux)
func (r *LinuxReader) Close() {
	// No cleanup needed - we use command-line tools
}

// Reinitialize re-detects the audio tool
func (r *LinuxReader) Reinitialize() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Re-detect audio tool
	if _, err := exec.LookPath("wpctl"); err == nil {
		if out, err := exec.Command("wpctl", "get-volume", "@DEFAULT_AUDIO_SINK@").Output(); err == nil && len(out) > 0 {
			r.audioTool = "wpctl"
			log.Printf("[VOLUME-LINUX] Reinitialized with wpctl (PipeWire)")
			return nil
		}
	}

	if _, err := exec.LookPath("pactl"); err == nil {
		if out, err := exec.Command("pactl", "get-sink-volume", "@DEFAULT_SINK@").Output(); err == nil && len(out) > 0 {
			r.audioTool = "pactl"
			log.Printf("[VOLUME-LINUX] Reinitialized with pactl (PulseAudio)")
			return nil
		}
	}

	if _, err := exec.LookPath("amixer"); err == nil {
		if out, err := exec.Command("amixer", "get", "Master").Output(); err == nil && len(out) > 0 {
			r.audioTool = "amixer"
			log.Printf("[VOLUME-LINUX] Reinitialized with amixer (ALSA)")
			return nil
		}
	}

	return fmt.Errorf("no audio tool available after reinitialization")
}

// NeedsReinitialize returns false - Linux reader doesn't need reinitialization
func (r *LinuxReader) NeedsReinitialize() bool {
	return false
}

// newVolumeReader creates a platform-specific volume reader (Linux implementation)
func newVolumeReader() (Reader, error) {
	return NewLinuxReader()
}
