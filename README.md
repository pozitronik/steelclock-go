# SteelClock (Go)

[![CI](https://github.com/pozitronik/steelclock-go/actions/workflows/ci.yml/badge.svg)](https://github.com/pozitronik/steelclock-go/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/pozitronik/steelclock-go/branch/master/graph/badge.svg)](https://codecov.io/gh/pozitronik/steelclock-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/pozitronik/steelclock-go)](https://goreportcard.com/report/github.com/pozitronik/steelclock-go)
[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)


High-performance display manager for SteelSeries devices written in Go.

https://github.com/user-attachments/assets/58f607cb-be31-4af4-bb3d-6e0628f0748c

## Requirements

- **Windows OS** (Windows 10/11)
- **SteelSeries Engine** or **SteelSeries GG** (optional)
- **Go 1.21+** (for building from source)

## Features

- **System Tray Integration**: Runs in background with system tray icon
- **Live Configuration Reload**: Edit and reload config without restarting
- **Multiple Widgets**: Clock, CPU, Memory, Network, Disk, Keyboard indicators, Keyboard layout, Volume control, Audio visualizer, Winamp integration
- **Display Modes**: Text, horizontal/vertical bars, graphs, analog gauges, etc
- **Per-Core CPU Monitoring**: Grid layouts showing individual core usage for all display modes
- **Widget Transparency**: Overlay widgets using `background_color: -1` for layered displays
- **Gauge Displays**: Semicircular analog gauges with needles for CPU/Memory/Volume, dual concentric gauges for Network (RX/TX)
- **Auto-Hide Widgets**: Widgets can appear temporarily and hide automatically (ideal for notifications and volume indicators)
- **Volume Control**: Real-time Windows system volume monitoring via Core Audio API
- **Low Resource Usage**: Minimal CPU and memory footprint (~0.5% CPU, ~15MB RAM)
- **Single Executable**: ~8-9MB, no dependencies, no DLLs required
- **Automatic Logging**: All output logged to `steelclock.log` with timestamps
- **JSON Schema Support**: Full IDE autocomplete and validation via included schema file

And it also runs [DOOM](configs/examples/DOOM_README.md).

## Quick Start

1. Build the application:
   ```bash
   # On Windows
   build.cmd

   # On Linux/WSL
   ./build.sh
   ```

2. Run the application:
   ```bash
   steelclock.exe
   ```
   The application starts in the background with a system tray icon.

3. Use the tray menu:
   - Right-click the tray icon
   - Choose "Edit Config" to modify settings
   - Choose "Reload Config" to apply changes
   - Choose "Exit" to close the application

### Manual Build

```bash
# Build for Windows (GUI mode - no console window)
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w -H windowsgui" -o steelclock.exe ./cmd/steelclock
```

### Command Line Options

```
-config string
    Path to configuration file (default "config.json")
```

## Configuration

Edit `config.json` to customize widgets. The application supports live reload via the tray menu.

**For complete configuration documentation**, see:
- **[CONFIG_GUIDE.md](configs/CONFIG_GUIDE.md)** - Comprehensive guide with all properties and examples
- **[config.schema.json](configs/config.schema.json)** - JSON schema for IDE autocomplete and validation
- **[configs/examples/](configs/examples/)** - Example configurations for each widget type

### Supported Widgets

| Widget               | Description                          | Modes                                                           |
|----------------------|--------------------------------------|-----------------------------------------------------------------|
| **clock**            | Current time display                 | text, analog                                                    |
| **cpu**              | CPU usage (per-core support)         | text, bar, graph, gauge                                         |
| **memory**           | RAM usage                            | text, bar, graph, gauge                                         |
| **network**          | Network I/O (RX/TX)                  | text, bar, graph, gauge                                         |
| **disk**             | Disk I/O (read/write)                | text, bar, graph                                                |
| **keyboard**         | Lock indicators (Caps/Num/Scroll)    | icons, text, mixed                                              |
| **keyboard_layout**  | Current keyboard input language      | text (ISO 639-1, ISO 639-2, full name)                          |
| **volume**           | System volume level and mute         | text, bar, gauge, triangle                                      |
| **volume_meter**     | Realtime audio peak meter            | bar, gauge (stereo & VU support)                                |
| **audio_visualizer** | Realtime audio spectrum/waveform     | spectrum, oscilloscope                                          |
| **winamp**           | Winamp player info display           | text (with scrolling support)                                   |
| **doom**             | Interactive DOOM game display        | game                                                            |

See [CONFIG_GUIDE.md](configs/CONFIG_GUIDE.md) for detailed widget properties and configuration examples.

## Direct Mode Connection

SteelClock supports two connection backends for communicating with your SteelSeries device:

### Backend Options

| Backend     | Description                              | Refresh Rate        | Requirements                  |
|-------------|------------------------------------------|---------------------|-------------------------------|
| `gamesense` | Uses SteelSeries GG/Engine API (default) | 100ms (10 Hz)       | SteelSeries GG/Engine running |
| `direct`    | Direct USB HID communication             | ~16-30ms (30-60 Hz) | Windows only, device VID/PID  |
| `any`       | Try GameSense first, fallback to direct  | Varies              | -                             |

### Configuration

```json
{
  "backend": "direct",
  "direct_driver": {
    "vid": "1038",
    "pid": "1612",
    "interface": "mi_01"
  }
}
```

### Pros of Direct Mode

- **Higher refresh rates**: Up to 60 Hz vs 10 Hz with GameSense
- **Lower latency**: Direct USB communication without HTTP overhead
- **No SteelSeries software required**: Works without GG/Engine installed
- **Better for real-time visualizations**: Audio visualizer, smooth animations

### Cons of Direct Mode

- **Windows only**: USB HID library requires Windows
- **Device-specific configuration**: Need to know VID/PID of your device
- **Exclusive access**: May conflict with SteelSeries GG if running simultaneously
- **No automatic device discovery**: Must configure device identifiers manually
- **Limited testing**: Only tested with specific SteelSeries keyboards (Apex Pro)

### Finding Your Device VID/PID

1. Open Device Manager
2. Find your SteelSeries device under "Human Interface Devices"
3. Check device properties for VID (Vendor ID) and PID (Product ID)
4. Common values: VID `1038` (SteelSeries), PID varies by model

### Compatibility Notes

- Direct mode bypasses the GameSense API entirely
- Some devices may have multiple HID interfaces - use `interface` to specify (e.g., `mi_01`)
- If experiencing issues, try `backend: "any"` to automatically fallback
- Direct mode reconnects automatically if the device is disconnected and reconnected

## Troubleshooting

### Application won't start
- If using `gamesense` backend: Verify SteelSeries Engine/GG is installed and running
- If using `direct` backend: Verify device VID/PID are correct and device is connected
- Check if config file exists and is valid JSON
- Review `steelclock.log` for initialization errors and stack traces

### Config reload fails
- Check `config.json` syntax (must be valid JSON)
- Verify widget configurations are correct
- Check `steelclock.log` for specific validation errors

### Logging

All application output is logged to `steelclock.log` in the same directory as the executable. The log includes:
- Startup and shutdown events
- Configuration loading and validation errors
- Widget initialization
- GameSense API communication
- Runtime errors and warnings

Check this file if you encounter any issues or unexpected behavior.

## Testing

```bash
# Run all tests
go test ./... -cover

# Run with race detection
go test -race ./...

# Run with verbose output
go test ./... -v

# Check coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Dependencies

- `github.com/shirou/gopsutil/v4` - System monitoring
- `github.com/getlantern/systray` - System tray icon
- `golang.org/x/image` - Font rendering and image processing
- `github.com/moutend/go-wca` - Windows Core Audio API
- `github.com/go-ole/go-ole` - COM interface support for Windows APIs
- `github.com/mjibson/go-dsp` - Digital signal processing
- `github.com/go-toast/toast` - Windows toast notifications
- `github.com/AndreRenaud/gore` - DOOM engine port

## License

GNU General Public License v3.0
