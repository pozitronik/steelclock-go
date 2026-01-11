# SteelClock (Go)

[![CI](https://github.com/pozitronik/steelclock-go/actions/workflows/ci.yml/badge.svg)](https://github.com/pozitronik/steelclock-go/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/pozitronik/steelclock-go/branch/master/graph/badge.svg)](https://codecov.io/gh/pozitronik/steelclock-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/pozitronik/steelclock-go)](https://goreportcard.com/report/github.com/pozitronik/steelclock-go)
[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)


High-performance display manager for SteelSeries devices written in Go.

https://github.com/user-attachments/assets/58f607cb-be31-4af4-bb3d-6e0628f0748c

## Requirements

### Windows
- **Windows 10/11**
- **SteelSeries Engine** or **SteelSeries GG** (optional with direct driver)
- **Go 1.21+** (for building from source)

### Linux
- **Linux** with hidraw support
- **PipeWire** or **PulseAudio** (for audio widgets)
- **GTK 3** and **libayatana-appindicator3** (for system tray)
- **Go 1.21+** (for building from source)

## Features

- **System Tray Integration**: Runs in background with system tray icon
- **Configuration Profiles**: Switch between multiple configurations via tray menu
- **Live Configuration Reload**: Edit and reload config without restarting
- **Multiple Widgets**: Clock, CPU, Memory, Battery, Network, Disk, Keyboard indicators, Keyboard layout, Volume control, Audio visualizer, Winamp integration, Telegram notifications, Claude Code status, Matrix digital rain, Weather, Game of Life, Hyperspace, Star Wars intro
- **Display Modes**: Text, horizontal/vertical bars, graphs, analog gauges, etc
- **Per-Core CPU Monitoring**: Grid layouts showing individual core usage for all display modes
- **Widget Transparency**: Overlay widgets using `background_color: -1` for layered displays
- **Gauge Displays**: Semicircular analog gauges with needles for CPU/Memory/Volume, dual concentric gauges for Network (RX/TX)
- **Auto-Hide Widgets**: Widgets can appear temporarily and hide automatically (ideal for notifications and volume indicators)
- **Volume Control**: Real-time Windows system volume monitoring via Core Audio API
- **Low Resource Usage**: Minimal CPU and memory footprint (~0.5% CPU, ~15MB RAM)
- **Single Executable**: no dependencies, no DLLs required
- **Automatic Logging**: All output logged to `steelclock.log` with timestamps
- **JSON Schema Support**: Full IDE autocomplete and validation via included schema file

And it also runs [DOOM](profiles/DOOM_README.md).

## Quick Start

### Windows

1. Build the application:
   ```bash
   build.cmd
   # Or from WSL/bash:
   ./build.sh

   # For light build (smaller, excludes telegram widgets):
   build.cmd light
   ./build.sh --light
   ```

2. Run the application:
   ```bash
   steelclock.exe
   ```

### Linux

1. Install dependencies:
   ```bash
   sudo apt-get install libgtk-3-dev libayatana-appindicator3-dev
   ```

2. Build the application:
   ```bash
   ./build-linux.sh

   # For light build (smaller, excludes telegram widgets):
   ./build-linux.sh --light
   ```

3. Install udev rules for device access (one-time setup):
   ```bash
   sudo cp profiles/99-steelseries.rules /etc/udev/rules.d/
   sudo udevadm control --reload-rules
   sudo udevadm trigger
   # Unplug and replug your keyboard, or reboot
   ```

4. Run the application:
   ```bash
   ./steelclock
   ```

The application starts in the background with a system tray icon. Right-click the tray icon to access the menu for switching profiles, editing config, or exiting.

### Manual Build

```bash
# Build for Windows (GUI mode - no console window)
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w -H windowsgui" -o steelclock.exe ./cmd/steelclock

# Build for Linux
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o steelclock ./cmd/steelclock

# Light build (add -tags light to exclude telegram widgets)
go build -tags light -ldflags="-s -w" -o steelclock-light ./cmd/steelclock
```

### Build Variants

SteelClock provides two build variants:

| Variant | Size   | Description                                      |
|---------|--------|--------------------------------------------------|
| Full    | ~15 MB | All widgets included (default)                   |
| Light   | ~10 MB | Excludes widgets with external API dependencies  |

**Widgets excluded in light build:**
- `telegram` - Telegram notifications (requires Telegram API)
- `telegram_counter` - Telegram unread counter (requires Telegram API)

To modify the exclusion list, edit `cmd/steelclock/imports_light.go`.

### Command Line Options

```
-config string
    Path to configuration file (bypasses profile system)
```

## Configuration

The application uses `steelclock.json` as the main configuration file. The application supports live reload via the tray menu.

**For complete configuration documentation**, see:
- **[CONFIG_GUIDE.md](profiles/CONFIG_GUIDE.md)** - Comprehensive guide with all properties and examples
- **[config.schema.json](profiles/schema/config.schema.json)** - JSON schema for IDE autocomplete and validation
- **[profiles/](profiles/)** - Example configurations for each widget type

## Configuration Profiles

SteelClock supports multiple configuration profiles that can be switched via the tray menu.

### Profile System

- **Main config**: `steelclock.json` in the current working directory
- **Additional profiles**: JSON files in the `profiles/` subdirectory
- **Profile names**: Set via `config_name` field in JSON, or filename is used as fallback
- **State persistence**: Last active profile is saved to `.steelclock.state` and restored on restart

### Tray Menu Structure

```
Profile 1 (checkmark indicates active)
Profile 2
...
Profile N
─────────
Edit Active Config
Reload Active Config
─────────
Exit
```

### Example Profile Configuration

```json
{
  "$schema": "schema/config.schema.json",
  "config_name": "My Gaming Profile",
  "refresh_rate_ms": 50,
  "display": { ... },
  "widgets": [ ... ]
}
```

The `config_name` field determines how the profile appears in the tray menu. If omitted, the filename (without `.json` extension) is used.

### Supported Widgets

| Widget               | Description                       | Modes                                  | Windows |  Linux   |
|----------------------|-----------------------------------|----------------------------------------|:-------:|:--------:|
| **claude_code**      | Claude Code status with Clawd     | -                                      |   Yes   |   Yes    |
| **clock**            | Current time display              | text, analog                           |   Yes   |   Yes    |
| **cpu**              | CPU usage (per-core support)      | text, bar, graph, gauge                |   Yes   |   Yes    |
| **memory**           | RAM usage                         | text, bar, graph, gauge                |   Yes   |   Yes    |
| **battery**          | Battery level and charging status | text, bar, graph, gauge                |   Yes   |   Yes    |
| **network**          | Network I/O (RX/TX)               | text, bar, graph, gauge                |   Yes   |   Yes    |
| **disk**             | Disk I/O (read/write)             | text, bar, graph                       |   Yes   |   Yes    |
| **keyboard**         | Lock indicators (Caps/Num/Scroll) | icons, text, mixed                     |   Yes   |    No    |
| **keyboard_layout**  | Current keyboard input language   | text (ISO 639-1, ISO 639-2, full name) |   Yes   |    No    |
| **volume**           | System volume level and mute      | text, bar, gauge                       |   Yes   |   Yes*   |
| **volume_meter**     | Realtime audio peak meter         | bar, gauge (stereo & VU support)       |   Yes   | Limited* |
| **audio_visualizer** | Realtime audio spectrum/waveform  | spectrum, oscilloscope                 |   Yes   |   Yes*   |
| **winamp**           | Winamp player info display        | text (with scrolling support)          |   Yes   |    No    |
| **beefweb**          | Foobar2000/DeaDBeeF player        | text (with scrolling support)          |   Yes   |   Yes    |
| **telegram**         | Telegram notifications display    | text (with scrolling/transitions)      |   Yes   |   Yes    |
| **telegram_counter** | Telegram unread message counter   | text                                   |   Yes   |   Yes    |
| **doom**             | Interactive DOOM game display     | game                                   |   Yes   |   Yes    |
| **game_of_life**     | Conway's Game of Life simulation  | -                                      |   Yes   |   Yes    |
| **hyperspace**       | Star Wars hyperspace animation    | -                                      |   Yes   |   Yes    |
| **starwars_intro**   | Star Wars opening crawl text      | -                                      |   Yes   |   Yes    |
| **matrix**           | Matrix "digital rain" effect      | -                                      |   Yes   |   Yes    |
| **weather**          | Current weather conditions        | icon, text                             |   Yes   |   Yes    |

\* See [Linux Limitations](#linux-limitations) section below.

**Note:** The `beefweb` widget requires the [beefweb](https://github.com/hyperblast/beefweb) plugin installed in Foobar2000 (Windows) or DeaDBeeF (Linux).

See [CONFIG_GUIDE.md](profiles/CONFIG_GUIDE.md) for detailed widget properties and configuration examples.

## Direct Mode Connection

SteelClock supports two connection backends for communicating with your SteelSeries device:

### Backend Options

| Backend     | Description                              | Refresh Rate        | Requirements                        |
|-------------|------------------------------------------|---------------------|-------------------------------------|
| `gamesense` | Uses SteelSeries GG/Engine API           | 100ms (10 Hz)       | SteelSeries GG/Engine running       |
| `direct`    | Direct USB HID communication             | ~16-30ms (30-60 Hz) | Device VID/PID, udev rules on Linux |
| (omitted)   | Auto-select: tries gamesense, then direct | Varies             | -                                   |

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

If "direct_driver" section is empty, app will try to detect your hardware automatically.

### Pros of Direct Mode

- **Higher refresh rates**: Up to 60 Hz vs 10 Hz with GameSense
- **Lower latency**: Direct USB communication without HTTP overhead
- **No SteelSeries software required**: Works without GG/Engine installed
- **Better for real-time visualizations**: Audio visualizer, smooth animations
- **Cross-platform**: Works on both Windows and Linux

### Cons of Direct Mode

- **Device-specific configuration**: Need to know VID/PID of your device
- **Exclusive access**: May conflict with SteelSeries GG if running simultaneously
- **Limited testing**: Only tested with specific SteelSeries keyboards (Apex Pro)
- **Linux requires udev rules**: Need to install udev rules for non-root access

### Finding Your Device VID/PID

**Windows:**
1. Open Device Manager
2. Find your SteelSeries device under "Human Interface Devices"
3. Check device properties for VID (Vendor ID) and PID (Product ID)

**Linux:**
```bash
lsusb | grep -i steelseries
# Output: Bus 001 Device 005: ID 1038:1612 SteelSeries ApS SteelSeries Apex Pro
#                                ^^^^ ^^^^
#                                VID  PID
```

Common values: VID `1038` (SteelSeries), PID varies by model.

### Compatibility Notes

- Direct mode bypasses the GameSense API entirely
- Some devices may have multiple HID interfaces - use `interface` to specify (e.g., `mi_01`)
- If experiencing issues, omit the `backend` field to enable auto-selection with fallback
- Direct mode reconnects automatically if the device is disconnected and reconnected

## Linux Limitations

On Linux, some widgets have reduced functionality compared to Windows:

### Unsupported Widgets

| Widget              | Reason                                                    |
|---------------------|-----------------------------------------------------------|
| **keyboard**        | Requires Windows `GetKeyState` API for lock key detection |
| **keyboard_layout** | Requires Windows input language API                       |
| **winamp**          | Winamp is Windows-only software                           |

### Limited Functionality

| Widget               | Limitation                                                                                                                |
|----------------------|---------------------------------------------------------------------------------------------------------------------------|
| **volume**           | Uses command-line tools (`wpctl`, `pactl`, `amixer`) instead of native API. Polling-based, not event-driven.              |
| **volume_meter**     | Real-time audio peak metering is limited. Falls back to volume level as a proxy when actual audio levels are unavailable. |
| **audio_visualizer** | Requires PipeWire with `parec` for audio capture. May need additional configuration for proper audio routing.             |

### Audio Setup on Linux

For audio widgets to work properly on Linux:

1. **PipeWire (recommended)**:
   ```bash
   # Ensure PipeWire is running
   systemctl --user status pipewire

   # Install PipeWire tools if needed
   sudo apt-get install pipewire-audio-client-libraries
   ```

2. **PulseAudio**:
   ```bash
   # Check PulseAudio is running
   pactl info
   ```

3. **Audio capture for visualizer**:
   The audio visualizer captures system audio output. On PipeWire, this should work automatically. On PulseAudio, you may need to configure a monitor source.

## Troubleshooting

### Application won't start
- If using `gamesense` backend: Verify SteelSeries Engine/GG is installed and running
- If using `direct` backend: Verify device VID/PID are correct and device is connected
- Check if `steelclock.json` exists and is valid JSON
- Review `steelclock.log` for initialization errors and stack traces

### Config reload fails
- Check configuration file syntax (must be valid JSON)
- Verify widget configurations are correct
- Check `steelclock.log` for specific validation errors

### Linux-specific issues

**"Permission denied" when accessing device:**
```bash
# Install udev rules
sudo cp profiles/99-steelseries.rules /etc/udev/rules.d/
sudo udevadm control --reload-rules
sudo udevadm trigger
# Unplug and replug your keyboard
```

**"cannot read /sys/class/hidraw" error:**
- Ensure the `hidraw` kernel module is loaded: `lsmod | grep hidraw`
- Check if your device appears: `ls /dev/hidraw*`

**System tray icon not appearing:**
- Ensure you have a system tray implementation (e.g., `gnome-shell-extension-appindicator` for GNOME)
- Check that `libayatana-appindicator3` is installed

**Audio widgets not working:**
- Check which audio system is running: `pactl info` or `wpctl status`
- Ensure audio tools are installed: `wpctl`, `pactl`, or `amixer`
- For audio visualizer, PipeWire is recommended

### Logging

All application output is logged to `steelclock.log` in the same directory as the executable. The log includes:
- Startup and shutdown events
- Configuration loading and validation errors
- Widget initialization
- GameSense API communication (Windows) / HID communication (Linux)
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
