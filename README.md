# SteelClock (Go)

[![CI](https://github.com/pozitronik/steelclock-go/actions/workflows/ci.yml/badge.svg)](https://github.com/pozitronik/steelclock-go/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/pozitronik/steelclock-go/branch/master/graph/badge.svg)](https://codecov.io/gh/pozitronik/steelclock-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/pozitronik/steelclock-go)](https://goreportcard.com/report/github.com/pozitronik/steelclock-go)
[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)


High-performance display manager for SteelSeries devices written in Go.

## Features

- **System Tray Integration**: Runs in background with system tray icon
- **Live Configuration Reload**: Edit and reload config without restarting
- **Multiple Widgets**: Clock, CPU, Memory, Network, Disk, Keyboard indicators, Volume control
- **Display Modes**: Text, horizontal/vertical bars, graphs, analog gauges, and triangle indicators
- **Gauge Displays**: Semicircular analog gauges with needles for CPU/Memory/Volume, dual concentric gauges for Network (RX/TX)
- **Auto-Hide Widgets**: Widgets can appear temporarily and hide automatically (ideal for notifications and volume indicators)
- **Volume Control**: Real-time Windows system volume monitoring via Core Audio API
- **Low Resource Usage**: Minimal CPU and memory footprint (~0.5% CPU, ~15MB RAM)
- **Single Executable**: ~10MB, no dependencies, no DLLs required
- **Automatic Logging**: All output logged to `steelclock.log` with timestamps
- **Cross-Platform Build**: Build for Windows from Linux/WSL or Windows
- **JSON Schema Support**: Full IDE autocomplete and validation via included schema file

## Quick Start

1. Build the application:
   ```bash
   # On Windows
   build.cmd

   # On Linux/WSL
   ./build.sh
   ```

   The build script automatically handles icon resources if configured.

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

### Logging

All application output is logged to `steelclock.log` in the same directory as the executable. The log includes:
- Startup and shutdown events
- Configuration loading and validation errors
- Widget initialization
- GameSense API communication
- Runtime errors and warnings

Check this file if you encounter any issues or unexpected behavior.

## Configuration

Edit `config.json` to customize widgets. The application supports live reload via the tray menu.

For detailed configuration documentation with all widget properties and examples, see [CONFIG_GUIDE.md](configs/CONFIG_GUIDE.md) and [config.schema.json](configs/config.schema.json).

### Example Configuration

```json
{
  "game_name": "STEELCLOCK",
  "game_display_name": "Steel Clock",
  "refresh_rate_ms": 100,
  "display": {
    "width": 128,
    "height": 40,
    "background_color": 0
  },
  "widgets": [
    {
      "type": "clock",
      "id": "main_clock",
      "enabled": true,
      "position": {"x": 0, "y": 0, "w": 128, "h": 40},
      "style": {
        "background_color": 0,
        "border": false,
        "border_color": 255
      },
      "properties": {
        "format": "15:04:05",
        "font_size": 16,
        "horizontal_align": "center",
        "vertical_align": "center"
      }
    }
  ]
}
```

### Supported Widgets

| Widget       | Description                       | Modes                                                   | Platform Support    |
|--------------|-----------------------------------|---------------------------------------------------------|---------------------|
| **clock**    | Current time display              | text                                                    | All                 |
| **cpu**      | CPU usage                         | text, bar_horizontal, bar_vertical, graph, gauge        | All                 |
| **memory**   | RAM usage                         | text, bar_horizontal, bar_vertical, graph, gauge        | All                 |
| **network**  | Network I/O (RX/TX)               | text, bar_horizontal, bar_vertical, graph, gauge        | All                 |
| **disk**     | Disk I/O (read/write)             | text, bar_horizontal, bar_vertical, graph               | All                 |
| **keyboard** | Lock indicators (Caps/Num/Scroll) | text                                                    | Windows only        |
| **volume**   | System volume level and mute      | text, bar_horizontal, bar_vertical, gauge, triangle     | Windows only        |

### Time Format

Go uses reference time format: `Mon Jan 2 15:04:05 MST 2006`

Common formats:
- `15:04:05` - HH:MM:SS (24-hour)
- `15:04` - HH:MM
- `03:04 PM` - hh:mm AM/PM (12-hour)
- `2006-01-02 15:04` - YYYY-MM-DD HH:MM

Or use common Python-style formats (auto-converted):
- `%H:%M:%S` → `15:04:05`
- `%H:%M` → `15:04`

## Troubleshooting

### Application won't start
- Verify SteelSeries Engine/GG is installed and running
- Check if config file exists and is valid
- Review `steelclock.log` for initialization errors and stack traces

### Config reload fails
- Check `config.json` syntax (must be valid JSON)
- Verify widget configurations are correct
- Check `steelclock.log` for specific validation errors

## Testing

```bash
# Run all tests
go test ./... -cover

# Run with verbose output
go test ./... -v

# Check coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Dependencies

- `github.com/shirou/gopsutil/v4` - System monitoring (CPU, memory, disk, network)
- `github.com/getlantern/systray` - System tray icon
- `golang.org/x/image` - Font rendering and image processing
- `github.com/moutend/go-wca` - Windows Core Audio API access (volume widget, Windows only)
- `github.com/go-ole/go-ole` - COM interface support (volume widget, Windows only)

## License

GNU General Public License v3.0
