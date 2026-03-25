# GameDAC / Nova Pro Display Support

SteelClock supports GameDAC Gen 2 and Arctis Nova Pro base stations via the direct USB HID driver.

## Supported Devices

| Device                                  | PID    | Display | USB Interface |
|-----------------------------------------|--------|---------|---------------|
| Arctis Nova Pro (Wired)                 | `12cb` | 128x64  | `mi_04`       |
| Arctis Nova Pro Wireless (Base Station) | `12cd` | 128x64  | `mi_04`       |
| Arctis Nova Pro Wireless (USB-C Dongle) | `12e0` | 128x64  | `mi_04`       |
| Arctis Nova Pro Wireless (Xbox)         | `12e5` | 128x64  | `mi_04`       |
| Arctis Nova 5P (USB-C Dongle)           | `225d` | 128x64  | `mi_04`       |

All devices use SteelSeries VID `1038`.

**Note:** GameDAC Gen 1 (PID `1280`) is not supported due to an undocumented USB protocol. It may work via the GameSense backend on Windows if SteelSeries GG is running.

## Quick Start

### 1. Linux: Install udev Rules

On Linux, USB access requires udev rules (one-time setup):

```bash
sudo cp profiles/99-steelseries.rules /etc/udev/rules.d/
sudo udevadm control --reload-rules
sudo udevadm trigger
```

Then unplug and replug the device, or reboot.

### 2. Create a Configuration

Create a profile JSON file (e.g. `profiles/gamedac.json`):

```json
{
  "$schema": "schema/config.schema.json",
  "config_name": "GameDAC",
  "game_name": "STEELCLOCK",
  "game_display_name": "SteelClock",
  "refresh_rate_ms": 100,
  "backend": "direct",
  "display": {
    "width": 128,
    "height": 64,
    "background": 0
  },
  "direct_driver": {
    "vid": "1038",
    "pid": "12cb",
    "brightness": 5
  },
  "widgets": [
    {
      "type": "clock",
      "enabled": true,
      "position": { "x": 0, "y": 0, "w": 128, "h": 64 },
      "mode": "segment",
      "segment": {
        "format": "%H:%M:%S",
        "colon_blink": true
      },
      "update_interval": 0.1
    }
  ]
}
```

### 3. Run

```bash
# With a specific profile:
./steelclock -config profiles/gamedac.json

# Or select from tray menu if placed in profiles/ directory
```

## Configuration Reference

### Direct Driver Settings

```json
{
  "direct_driver": {
    "vid": "1038",
    "pid": "12cb",
    "brightness": 5
  }
}
```

| Field        | Type    | Required | Description                                                                     |
|--------------|---------|----------|---------------------------------------------------------------------------------|
| `vid`        | string  | No       | Vendor ID in hex (e.g. `"1038"`). Omit for auto-detection.                      |
| `pid`        | string  | No       | Product ID in hex (e.g. `"12cb"`). Omit for auto-detection.                     |
| `brightness` | integer | No       | Display brightness, 0 (darkest) to 10 (brightest). Omit to keep device default. |

The `interface` field is not needed for Nova Pro devices. The correct USB interface (`mi_04`) is selected automatically based on the detected device protocol.

### Auto-Detection

If you have only one SteelSeries device connected, you can omit `vid` and `pid` entirely:

```json
{
  "backend": "direct",
  "display": { "width": 128, "height": 64, "background": 0 },
  "direct_driver": {
    "brightness": 5
  }
}
```

SteelClock scans all known device PIDs and connects to the first one found. For multi-device setups, explicit VID/PID is required.

## Multi-Device Setup (Keyboard + GameDAC)

SteelClock can drive multiple displays simultaneously. Use the `devices` array instead of top-level `display`/`widgets`:

```json
{
  "$schema": "schema/config.schema.json",
  "config_name": "Keyboard + GameDAC",
  "game_name": "STEELCLOCK",
  "game_display_name": "SteelClock",
  "refresh_rate_ms": 100,
  "devices": [
    {
      "id": "keyboard",
      "display": { "width": 128, "height": 40, "background": 0 },
      "backend": "direct",
      "direct_driver": { "interface": "mi_01" },
      "widgets": [
        {
          "type": "clock",
          "enabled": true,
          "position": { "x": 0, "y": 0, "w": 128, "h": 40 },
          "mode": "segment",
          "segment": { "format": "%H:%M", "colon_blink": true },
          "update_interval": 0.1
        }
      ]
    },
    {
      "id": "gamedac",
      "display": { "width": 128, "height": 64, "background": 0 },
      "backend": "direct",
      "direct_driver": { "vid": "1038", "pid": "12cb", "brightness": 5 },
      "widgets": [
        {
          "type": "cpu",
          "enabled": true,
          "position": { "x": 0, "y": 0, "w": 64, "h": 32 },
          "mode": "graph",
          "update_interval": 0.5
        },
        {
          "type": "memory",
          "enabled": true,
          "position": { "x": 64, "y": 0, "w": 64, "h": 32 },
          "mode": "bar",
          "update_interval": 1
        },
        {
          "type": "clock",
          "enabled": true,
          "position": { "x": 0, "y": 32, "w": 128, "h": 32 },
          "mode": "segment",
          "segment": { "format": "%H:%M:%S", "colon_blink": true },
          "update_interval": 0.1
        }
      ]
    }
  ]
}
```

Each device has its own display dimensions, backend, driver settings, and widget set. Devices operate independently: if one device disconnects, the others continue running.

## GameDAC-Specific Features

### Brightness Control

Set display brightness (0 to 10) in the `direct_driver` config. Applied once at startup.

### Return-to-UI

When SteelClock exits, it sends a Return-to-UI command to the Nova Pro, restoring the native device screen (volume display, EQ settings, etc.). This happens automatically on clean shutdown.

## Backends Comparison

| Backend     | GameDAC Support | Brightness | Return-to-UI | Refresh Rate | Requirements                          |
|-------------|-----------------|------------|--------------|--------------|---------------------------------------|
| `direct`    | Full            | Yes        | Yes          | Up to 60 Hz  | Device connected via USB              |
| `gamesense` | Partial         | No         | No           | ~10 Hz       | SteelSeries GG running (Windows only) |

The direct driver is recommended for GameDAC devices.

## Testing Without Hardware

Use the web editor to preview GameDAC layouts in a browser:

1. Set `"backend": "webclient"` in your config, or use the web editor's Apply button (it switches to webclient automatically)
2. Run SteelClock
3. Open `http://localhost:27302`
4. The preview canvas renders at the configured display resolution (128x64 for GameDAC)
5. In multi-device mode, each device appears as a separate tab with its own preview

## Troubleshooting

### Device Not Found

```
lsusb | grep -i steelseries
```

Verify the device appears and note the PID. Make sure it matches your `direct_driver.pid` value.

### Permission Denied (Linux)

Install the udev rules as described in the Quick Start section. The device must be unplugged and replugged after installing rules.

### Conflicts with SteelSeries GG

The direct driver takes exclusive access to the HID device. If SteelSeries GG is running, it may hold the device open. Either:
- Close SteelSeries GG before running SteelClock with the direct backend
- Use `"backend": "gamesense"` to work through the GG API instead (limited features)

### Display Shows Garbage or Nothing

- Verify `display.width` and `display.height` match the device (128x64 for GameDAC Gen 2)
- Check that `direct_driver.pid` matches the correct device
- Review `steelclock.log` for protocol or connection errors
