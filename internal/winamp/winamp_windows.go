//go:build windows

package winamp

import (
	"fmt"
	"path/filepath"
	"syscall"
	"unsafe"
)

// Winamp IPC message constants
const (
	WM_USER   = 0x0400
	WM_WA_IPC = WM_USER

	// IPC commands - see wa_ipc.h
	IPC_GETVERSION       = 0   // Returns Winamp version (0x5xyy for 5.xy)
	IPC_ISPLAYING        = 104 // Returns playback status: 0=stopped, 1=playing, 3=paused
	IPC_GETOUTPUTTIME    = 105 // wParam=0: position in ms, wParam=1: track length in seconds
	IPC_GETLISTLENGTH    = 124 // Returns number of tracks in playlist
	IPC_GETLISTPOS       = 125 // Returns current playlist position (0-indexed)
	IPC_GETINFO          = 126 // wParam: 0=samplerate, 1=bitrate, 2=channels
	IPC_GETPLAYLISTFILE  = 211 // Returns pointer to filename (plugin-only)
	IPC_GETPLAYLISTTITLE = 212 // Returns pointer to title (plugin-only)
	IPC_GET_SHUFFLE      = 250 // Returns shuffle state (1=enabled)
	IPC_GET_REPEAT       = 251 // Returns repeat state (1=enabled)
)

// Windows API functions
var (
	user32           = syscall.NewLazyDLL("user32.dll")
	procFindWindowW  = user32.NewProc("FindWindowW")
	procSendMessageW = user32.NewProc("SendMessageW")
)

// windowsClient implements the Winamp client for Windows
type windowsClient struct{}

func newPlatformClient() Client {
	return &windowsClient{}
}

// findWinampWindow locates the Winamp main window
func findWinampWindow() uintptr {
	className, _ := syscall.UTF16PtrFromString("Winamp v1.x")
	hwnd, _, _ := procFindWindowW.Call(
		uintptr(unsafe.Pointer(className)),
		0,
	)
	return hwnd
}

// sendMessage sends a WM_WA_IPC message to Winamp
func sendMessage(hwnd uintptr, wParam, lParam uintptr) uintptr {
	ret, _, _ := procSendMessageW.Call(
		hwnd,
		WM_WA_IPC,
		wParam,
		lParam,
	)
	return ret
}

// IsRunning returns true if Winamp is running
func (c *windowsClient) IsRunning() bool {
	return findWinampWindow() != 0
}

// GetStatus returns the current playback status
func (c *windowsClient) GetStatus() PlaybackStatus {
	hwnd := findWinampWindow()
	if hwnd == 0 {
		return StatusStopped
	}

	status := sendMessage(hwnd, 0, IPC_ISPLAYING)
	return PlaybackStatus(status)
}

// GetCurrentTitle returns the title of the currently playing track
func (c *windowsClient) GetCurrentTitle() string {
	hwnd := findWinampWindow()
	if hwnd == 0 {
		return ""
	}

	// The IPC_GETPLAYLISTTITLE only works from plugins (in-process)
	// For external apps, we parse the window title instead
	return c.getTitleFromWindowTitle(hwnd)
}

// getTitleFromWindowTitle extracts the track title from Winamp's window title
// Window title format is typically: "N. Artist - Title - Winamp" or "Artist - Title - Winamp"
func (c *windowsClient) getTitleFromWindowTitle(hwnd uintptr) string {
	// Get window title length
	getWindowTextLength := user32.NewProc("GetWindowTextLengthW")
	length, _, _ := getWindowTextLength.Call(hwnd)
	if length == 0 {
		return ""
	}

	// Get window title
	getWindowText := user32.NewProc("GetWindowTextW")
	buf := make([]uint16, length+1)
	getWindowText.Call(
		hwnd,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(length+1),
	)

	title := syscall.UTF16ToString(buf)

	// Remove " - Winamp" suffix if present
	suffix := " - Winamp"
	if len(title) > len(suffix) && title[len(title)-len(suffix):] == suffix {
		title = title[:len(title)-len(suffix)]
	}

	// Remove "[Paused]" or "[Stopped]" prefix if present
	prefixes := []string{"[Paused] ", "[Stopped] "}
	for _, prefix := range prefixes {
		if len(title) > len(prefix) && title[:len(prefix)] == prefix {
			title = title[len(prefix):]
			break
		}
	}

	// Remove track number prefix (e.g., "1. " or "12. ")
	for i := 0; i < len(title); i++ {
		if title[i] == '.' && i > 0 && i < len(title)-1 && title[i+1] == ' ' {
			// Check if everything before the dot is a number
			isNumber := true
			for j := 0; j < i; j++ {
				if title[j] < '0' || title[j] > '9' {
					isNumber = false
					break
				}
			}
			if isNumber {
				title = title[i+2:]
			}
			break
		}
	}

	return title
}

// GetCurrentPosition returns the current playback position in milliseconds
func (c *windowsClient) GetCurrentPosition() int {
	hwnd := findWinampWindow()
	if hwnd == 0 {
		return -1
	}

	// mode=0 returns position in milliseconds
	pos := sendMessage(hwnd, 0, IPC_GETOUTPUTTIME)
	if pos == ^uintptr(0) { // -1
		return -1
	}
	return int(pos)
}

// GetTrackDuration returns the track duration in seconds
func (c *windowsClient) GetTrackDuration() int {
	hwnd := findWinampWindow()
	if hwnd == 0 {
		return -1
	}

	// mode=1 returns duration in seconds
	duration := sendMessage(hwnd, 1, IPC_GETOUTPUTTIME)
	if duration == ^uintptr(0) { // -1
		return -1
	}
	return int(duration)
}

// GetTrackInfo returns comprehensive information about the current track
func (c *windowsClient) GetTrackInfo() *TrackInfo {
	hwnd := findWinampWindow()
	if hwnd == 0 {
		return nil
	}

	status := PlaybackStatus(sendMessage(hwnd, 0, IPC_ISPLAYING))

	info := &TrackInfo{
		Status: status,
	}

	// Get title from window title
	info.Title = c.getTitleFromWindowTitle(hwnd)

	// Get Winamp version
	version := sendMessage(hwnd, 0, IPC_GETVERSION)
	if version != 0 {
		// Version format: 0x5xyy for version 5.xy (e.g., 0x5666 = 5.666)
		major := (version >> 12) & 0xF
		minor := version & 0xFFF
		info.Version = fmt.Sprintf("%d.%d", major, minor)
	}

	// Get playlist info (always available)
	listLength := sendMessage(hwnd, 0, IPC_GETLISTLENGTH)
	info.PlaylistLength = int(listLength)

	listPos := sendMessage(hwnd, 0, IPC_GETLISTPOS)
	info.TrackNumber = int(listPos) + 1 // Convert from 0-indexed to 1-indexed

	// Get shuffle and repeat status
	shuffle := sendMessage(hwnd, 0, IPC_GET_SHUFFLE)
	info.Shuffle = shuffle == 1

	repeat := sendMessage(hwnd, 0, IPC_GET_REPEAT)
	info.Repeat = repeat == 1

	// Get position and duration
	if status != StatusStopped {
		pos := sendMessage(hwnd, 0, IPC_GETOUTPUTTIME)
		if pos != ^uintptr(0) {
			info.PositionMs = int(pos)
		}

		duration := sendMessage(hwnd, 1, IPC_GETOUTPUTTIME)
		if duration != ^uintptr(0) {
			info.DurationS = int(duration)
		}

		// Get audio info
		// mode=0: samplerate, mode=1: bitrate, mode=2: channels
		sampleRate := sendMessage(hwnd, 0, IPC_GETINFO)
		if sampleRate != 0 {
			info.SampleRate = int(sampleRate)
		}

		bitrate := sendMessage(hwnd, 1, IPC_GETINFO)
		if bitrate != 0 {
			info.Bitrate = int(bitrate)
		}

		channels := sendMessage(hwnd, 2, IPC_GETINFO)
		if channels != 0 {
			info.Channels = int(channels)
		}
	}

	// For filename, we'd need shared memory or other techniques
	// For now, we'll leave it empty as external IPC doesn't easily support this
	// The title is usually sufficient for display purposes
	if info.Title != "" {
		info.FileName = filepath.Base(info.Title)
	}

	return info
}
