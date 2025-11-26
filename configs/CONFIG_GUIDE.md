# SteelClock Configuration Guide v2

## Table of Contents

1. [Overview](#overview)
2. [JSON Schema](#json-schema)
3. [Configuration Structure](#configuration-structure)
4. [Widget Types](#widget-types)
5. [Common Properties](#common-properties)
6. [Widget-Specific Properties](#widget-specific-properties)
7. [Examples](#examples)

## Overview

SteelClock uses JSON configuration files with **JSON Schema** support for validation and IDE autocomplete. All aspects of the display are configurable:

- Widget types, positioning, sizing, and z-order
- Multiple instances of same widget type
- Widget styling (backgrounds, borders, transparency)
- Display settings
- Backend selection (GameSense or direct USB)

## JSON Schema

SteelClock includes a comprehensive JSON Schema (`config.schema.json`) that provides:

- **IDE Autocomplete**: Property suggestions while typing
- **Validation**: Real-time error checking
- **Documentation**: Inline descriptions and defaults
- **Type Safety**: Prevents configuration errors

### Enabling Schema in Your Config

Add these lines at the top of your configuration file:

```json
{
  "$schema": "./config.schema.json",
  "schema_version": 2,
  ...
}
```

Supported IDEs: VS Code, JetBrains IDEs, Visual Studio, Sublime Text, and others.

## Configuration Structure

### Top-Level Structure

```json
{
  "$schema": "./config.schema.json",
  "schema_version": 2,
  "game_name": "STEELCLOCK",
  "game_display_name": "SteelClock",
  "refresh_rate_ms": 100,
  "backend": "gamesense",
  "direct_driver": { ... },
  "display": { ... },
  "defaults": { ... },
  "widgets": [ ... ]
}
```

### Global Settings

| Property | Type | Default | Description |
|----------|------|---------|-------------|
| `schema_version` | integer | 2 | Schema version (must be 2) |
| `game_name` | string | "STEELCLOCK" | Internal game name for GameSense |
| `game_display_name` | string | "SteelClock" | Display name in SteelSeries GG |
| `refresh_rate_ms` | integer | 100 | Display refresh rate (see notes) |
| `backend` | string | "gamesense" | Backend: "gamesense", "direct", "any" |
| `unregister_on_exit` | boolean | false | Unregister on exit (may timeout) |
| `deinitialize_timer_ms` | integer | 15000 | Game deactivation timeout (1000-60000ms) |

### Backend Configuration

| Backend | Description | Min Refresh | Max Refresh |
|---------|-------------|-------------|-------------|
| `gamesense` | SteelSeries GG API (default) | 100ms (10Hz) | 100ms |
| `direct` | USB HID (Windows only) | ~16ms (60Hz) | 30ms (33Hz) |
| `any` | Try gamesense, fallback to direct | - | - |

**Direct Driver Config:**

```json
"direct_driver": {
  "vid": "1038",
  "pid": "1612",
  "interface": "mi_01"
}
```

If omitted, auto-detects from known devices (Apex 7, Apex Pro, etc.).

### Display Configuration

```json
"display": {
  "width": 128,
  "height": 40,
  "background": 0
}
```

| Property | Type | Range | Default | Description |
|----------|------|-------|---------|-------------|
| `width` | integer | - | 128 | Display width in pixels |
| `height` | integer | - | 40 | Display height in pixels |
| `background` | integer | 0-255 | 0 | Background color (0=black) |

### Defaults Configuration

Global defaults inherited by all widgets:

```json
"defaults": {
  "colors": {
    "primary": 255,
    "secondary": 200,
    "dim": 100
  },
  "text": {
    "font": "Consolas",
    "size": 10,
    "align": {"h": "center", "v": "center"}
  },
  "update_interval": 1.0
}
```

Widgets can reference default colors with `@name` syntax: `"fill": "@primary"`.

## Widget Types

SteelClock supports these widget types:

| Type | Description | Modes |
|------|-------------|-------|
| `clock` | Time display | text, analog |
| `cpu` | CPU usage monitor | text, bar, graph, gauge |
| `memory` | RAM usage monitor | text, bar, graph, gauge |
| `network` | Network I/O monitor | text, bar, graph, gauge |
| `disk` | Disk I/O monitor | text, bar, graph |
| `volume` | System volume | text, bar, gauge, triangle |
| `volume_meter` | Audio peak meter | text, bar, gauge |
| `audio_visualizer` | Spectrum/oscilloscope | spectrum, oscilloscope |
| `keyboard` | Lock key indicators | - |
| `keyboard_layout` | Current keyboard layout | - |
| `doom` | DOOM game | - |

## Common Properties

### Widget Base Structure

```json
{
  "type": "clock",
  "enabled": true,
  "position": { ... },
  "style": { ... },
  "mode": "text",
  "text": { ... },
  "auto_hide": { ... },
  "update_interval": 1.0
}
```

Note: Colors are defined within mode-specific objects (e.g., `bar.colors`, `graph.colors`, `gauge.colors`).

| Property | Type | Required | Description |
|----------|------|----------|-------------|
| `type` | string | Yes | Widget type |
| `enabled` | boolean | No | Enable widget (default: true) |
| `mode` | string | Depends | Display mode (widget-specific) |
| `update_interval` | number | No | Update interval in seconds (default: 1.0) |
| `poll_interval` | number | No | Internal polling interval for volume/volume_meter widgets in seconds (default: 0.1) |

### Position Object

```json
"position": {
  "x": 0,
  "y": 0,
  "w": 128,
  "h": 40,
  "z": 0
}
```

| Property | Type | Description |
|----------|------|-------------|
| `x` | integer | X coordinate (pixels) |
| `y` | integer | Y coordinate (pixels) |
| `w` | integer | Width (pixels) |
| `h` | integer | Height (pixels) |
| `z` | integer | Z-order (higher = on top) |

### Style Object

```json
"style": {
  "background": 0,
  "border": -1,
  "padding": 2
}
```

| Property | Type | Range | Default | Description |
|----------|------|-------|---------|-------------|
| `background` | integer | -1 to 255 | 0 | -1=transparent, 0-255=grayscale |
| `border` | integer | -1 to 255 | -1 | -1=disabled, 0-255=border color |
| `padding` | integer | 0+ | 0 | Padding from widget edges in pixels |

### Text Object

```json
"text": {
  "format": "%H:%M:%S",
  "font": "Consolas",
  "size": 10,
  "align": {"h": "center", "v": "center"}
}
```

| Property | Type | Description |
|----------|------|-------------|
| `format` | string | Format string (widget-specific) |
| `font` | string | Font name or TTF path |
| `size` | integer | Font size in pixels |
| `align.h` | string | "left", "center", "right" |
| `align.v` | string | "top", "center", "bottom" |

### Auto-Hide Object

```json
"auto_hide": {
  "enabled": true,
  "timeout": 2.0,
  "on_silence": false
}
```

| Property | Type | Default | Description |
|----------|------|---------|-------------|
| `enabled` | boolean | false | Enable auto-hide |
| `timeout` | number | 2.0 | Seconds before hiding |
| `on_silence` | boolean | false | Hide on audio silence |

### Mode-Specific Objects

Widgets with multiple modes use mode-named objects:

**Bar Mode:**
```json
"bar": {
  "direction": "horizontal",
  "border": false,
  "colors": {
    "fill": 255
  }
}
```

**Graph Mode:**
```json
"graph": {
  "history": 60,
  "filled": true,
  "colors": {
    "fill": 255
  }
}
```

**Gauge Mode:**
```json
"gauge": {
  "show_ticks": true,
  "colors": {
    "arc": 200,
    "needle": 255,
    "ticks": 150
  }
}
```

**Note:** Colors are now nested within mode-specific objects (e.g., `bar.colors.fill` instead of `colors.fill`).

## Widget-Specific Properties

### Clock Widget

**Modes:** `text`, `analog`

#### Text Mode

```json
{
  "type": "clock",
  "position": {"x": 0, "y": 0, "w": 128, "h": 40},
  "mode": "text",
  "text": {
    "format": "%H:%M:%S",
    "size": 16,
    "align": {"h": "center", "v": "center"}
  },
  "update_interval": 1.0
}
```

**Format Examples:**
- `"%H:%M:%S"` - 15:43:27 (24-hour)
- `"%I:%M %p"` - 03:43 PM (12-hour)
- `"%Y-%m-%d"` - 2025-11-25

#### Analog Mode

```json
{
  "type": "clock",
  "position": {"x": 44, "y": 0, "w": 40, "h": 40},
  "mode": "analog",
  "analog": {
    "show_seconds": true,
    "show_ticks": true,
    "colors": {
      "face": 50,
      "hour": 255,
      "minute": 200,
      "second": 128
    }
  }
}
```

### CPU Widget

**Modes:** `text`, `bar`, `graph`, `gauge`

```json
{
  "type": "cpu",
  "position": {"x": 0, "y": 0, "w": 128, "h": 40},
  "mode": "gauge",
  "gauge": {
    "show_ticks": true,
    "colors": {
      "arc": 200,
      "needle": 255,
      "ticks": 150
    }
  },
  "per_core": {
    "enabled": false
  },
  "update_interval": 1.0
}
```

| Property | Description |
|----------|-------------|
| `per_core.enabled` | Show per-core usage |
| `per_core.margin` | Margin between core bars |
| `bar.colors.fill` | Bar fill color |
| `graph.colors.fill` | Graph fill color |
| `gauge.colors.arc` | Gauge arc color |
| `gauge.colors.needle` | Gauge needle color |

### Memory Widget

**Modes:** `text`, `bar`, `graph`, `gauge`

Same structure as CPU widget, without `per_core`.

### Network Widget

**Modes:** `text`, `bar`, `graph`, `gauge`

```json
{
  "type": "network",
  "position": {"x": 0, "y": 0, "w": 128, "h": 40},
  "mode": "gauge",
  "interface": null,
  "max_speed_mbps": 100,
  "gauge": {
    "show_ticks": true,
    "colors": {
      "rx": 255,
      "tx": 128,
      "rx_needle": 255,
      "tx_needle": 200
    }
  }
}
```

| Property | Description |
|----------|-------------|
| `interface` | Network interface (null=auto) |
| `max_speed_mbps` | Max speed for scaling (-1=auto) |
| `gauge.colors.rx` | RX (download) arc color |
| `gauge.colors.tx` | TX (upload) arc color |
| `gauge.colors.rx_needle` | RX needle color |
| `gauge.colors.tx_needle` | TX needle color |
| `graph.colors.rx` | RX graph fill color |
| `graph.colors.tx` | TX graph fill color |

### Disk Widget

**Modes:** `text`, `bar`, `graph`

```json
{
  "type": "disk",
  "position": {"x": 0, "y": 0, "w": 128, "h": 40},
  "mode": "graph",
  "disk": null,
  "max_speed_mbps": -1,
  "graph": {
    "history": 60,
    "filled": true,
    "colors": {
      "read": 255,
      "write": 200
    }
  }
}
```

### Volume Widget

**Modes:** `text`, `bar`, `gauge`, `triangle`

```json
{
  "type": "volume",
  "position": {"x": 0, "y": 0, "w": 40, "h": 40},
  "mode": "triangle",
  "triangle": {
    "border": true,
    "colors": {
      "fill": 255
    }
  },
  "auto_hide": {
    "enabled": true,
    "timeout": 2.0
  },
  "poll_interval": 0.1
}
```

### Volume Meter Widget

**Modes:** `text`, `bar`, `gauge`

```json
{
  "type": "volume_meter",
  "position": {"x": 0, "y": 0, "w": 128, "h": 40},
  "mode": "bar",
  "bar": {
    "direction": "vertical",
    "colors": {
      "fill": 255,
      "clipping": 200
    }
  },
  "gauge": {
    "show_ticks": true,
    "colors": {
      "arc": 200,
      "needle": 255,
      "ticks": 150,
      "clipping": 200
    }
  },
  "text": {
    "format": "%d%%",
    "colors": {
      "fill": 255,
      "clipping": 200
    }
  },
  "stereo": {
    "enabled": true,
    "divider": 64
  },
  "metering": {
    "db_scale": false,
    "decay_rate": 2.0,
    "silence_threshold": 0.01
  },
  "peak": {
    "enabled": true,
    "hold_time": 1.0
  },
  "clipping": {
    "enabled": true,
    "threshold": 0.99
  },
  "auto_hide": {
    "on_silence": true
  },
  "poll_interval": 0.05
}
```

| Object | Properties |
|--------|------------|
| `bar.colors` | `fill`, `clipping` |
| `gauge.colors` | `arc`, `needle`, `ticks`, `clipping` |
| `text.colors` | `fill`, `clipping` |
| `stereo` | `enabled`, `divider` |
| `metering` | `db_scale`, `decay_rate`, `silence_threshold` |
| `peak` | `enabled`, `hold_time` |
| `clipping` | `enabled`, `threshold` |

### Audio Visualizer Widget

**Modes:** `spectrum`, `oscilloscope`

#### Spectrum Mode

```json
{
  "type": "audio_visualizer",
  "position": {"x": 0, "y": 0, "w": 128, "h": 40},
  "mode": "spectrum",
  "spectrum": {
    "bars": 32,
    "scale": "logarithmic",
    "style": "bars",
    "smoothing": 0.7,
    "frequency_compensation": true,
    "dynamic_scaling": {
      "strength": 1.0,
      "window": 0.5
    },
    "peak": {
      "enabled": true,
      "hold_time": 1.0
    },
    "colors": {
      "fill": 255
    }
  },
  "channel": "stereo_combined",
  "update_interval": 0.033
}
```

| Property | Options | Description |
|----------|---------|-------------|
| `spectrum.bars` | 8-128 | Number of frequency bars |
| `spectrum.scale` | logarithmic, linear | Frequency distribution |
| `spectrum.style` | bars, line | Rendering style |
| `spectrum.smoothing` | 0.0-1.0 | Fall-off smoothing |
| `spectrum.peak.enabled` | true/false | Show peak hold indicators |
| `spectrum.peak.hold_time` | 0.1+ | Peak hold duration in seconds |

#### Oscilloscope Mode

```json
{
  "type": "audio_visualizer",
  "position": {"x": 0, "y": 0, "w": 128, "h": 40},
  "mode": "oscilloscope",
  "oscilloscope": {
    "style": "line",
    "samples": 128,
    "colors": {
      "fill": 255,
      "left": 255,
      "right": 200
    }
  },
  "channel": "stereo_separated"
}
```

| Property | Options | Description |
|----------|---------|-------------|
| `oscilloscope.style` | line, filled | Waveform style |
| `oscilloscope.samples` | 32-512 | Sample count |
| `channel` | mono, stereo_combined, stereo_separated | Channel mode |

### Keyboard Widget

```json
{
  "type": "keyboard",
  "position": {"x": 0, "y": 0, "w": 40, "h": 20},
  "indicators": {
    "caps": {"on": "CAPS", "off": ""},
    "num": {"on": "NUM", "off": ""},
    "scroll": {"on": "SCR", "off": ""}
  },
  "layout": {
    "spacing": 3,
    "separator": " "
  },
  "colors": {
    "on": 255,
    "off": 100
  },
  "text": {
    "size": 10,
    "align": {"h": "center", "v": "center"}
  }
}
```

**Icon Mode:** Omit all `indicators` to use graphical icons.

### Keyboard Layout Widget

```json
{
  "type": "keyboard_layout",
  "position": {"x": 0, "y": 0, "w": 30, "h": 20},
  "format": "iso639-1",
  "text": {
    "size": 10,
    "align": {"h": "center", "v": "center"}
  }
}
```

| Format | Example |
|--------|---------|
| `iso639-1` | EN, RU, DE |
| `iso639-2` | ENG, RUS, DEU |
| `full` | English, Русский |

### DOOM Widget

```json
{
  "type": "doom",
  "position": {"x": 0, "y": 0, "w": 128, "h": 40},
  "wad": "doom1.wad"
}
```

## Examples

### Example 1: Simple Clock

```json
{
  "$schema": "./config.schema.json",
  "schema_version": 2,
  "display": {"width": 128, "height": 40, "background": 0},
  "widgets": [
    {
      "type": "clock",
      "position": {"x": 0, "y": 0, "w": 128, "h": 40},
      "mode": "text",
      "text": {
        "format": "%H:%M:%S",
        "size": 16,
        "align": {"h": "center", "v": "center"}
      }
    }
  ]
}
```

### Example 2: CPU + Memory Bars

```json
{
  "$schema": "./config.schema.json",
  "schema_version": 2,
  "display": {"width": 128, "height": 40, "background": 0},
  "widgets": [
    {
      "type": "cpu",
      "position": {"x": 0, "y": 0, "w": 128, "h": 20, "z": 0},
      "style": {"border": 255},
      "mode": "bar",
      "bar": {"direction": "horizontal", "colors": {"fill": 255}}
    },
    {
      "type": "memory",
      "position": {"x": 0, "y": 20, "w": 128, "h": 20, "z": 0},
      "style": {"border": 255},
      "mode": "bar",
      "bar": {"direction": "horizontal", "colors": {"fill": 255}}
    }
  ]
}
```

### Example 3: Transparent Overlay

```json
{
  "$schema": "./config.schema.json",
  "schema_version": 2,
  "display": {"width": 128, "height": 40, "background": 0},
  "widgets": [
    {
      "type": "network",
      "position": {"x": 0, "y": 0, "w": 128, "h": 40, "z": 0},
      "mode": "graph",
      "graph": {"history": 30, "colors": {"rx": 200, "tx": 100}}
    },
    {
      "type": "clock",
      "position": {"x": 0, "y": 0, "w": 128, "h": 40, "z": 10},
      "style": {"background": -1},
      "mode": "text",
      "text": {"format": "%H:%M", "size": 16}
    }
  ]
}
```

### Example 4: Gauge Dashboard

```json
{
  "$schema": "./config.schema.json",
  "schema_version": 2,
  "display": {"width": 128, "height": 40, "background": 0},
  "widgets": [
    {
      "type": "cpu",
      "position": {"x": 0, "y": 0, "w": 64, "h": 40},
      "style": {"border": 255},
      "mode": "gauge",
      "gauge": {"show_ticks": true, "colors": {"arc": 200, "needle": 255, "ticks": 150}}
    },
    {
      "type": "memory",
      "position": {"x": 64, "y": 0, "w": 64, "h": 40},
      "style": {"border": 255},
      "mode": "gauge",
      "gauge": {"show_ticks": true, "colors": {"arc": 180, "needle": 255, "ticks": 150}}
    }
  ]
}
```

### Example 5: Audio Visualizer

```json
{
  "$schema": "./config.schema.json",
  "schema_version": 2,
  "refresh_rate_ms": 33,
  "display": {"width": 128, "height": 40, "background": 0},
  "widgets": [
    {
      "type": "audio_visualizer",
      "position": {"x": 0, "y": 0, "w": 128, "h": 40},
      "mode": "spectrum",
      "spectrum": {
        "bars": 32,
        "scale": "logarithmic",
        "smoothing": 0.7,
        "peak": {"enabled": true, "hold_time": 1.0},
        "colors": {"fill": 255}
      },
      "update_interval": 0.033
    }
  ]
}
```

## Tips and Best Practices

### Layout Design

1. **Z-Order**: Use `z` for overlays (background=0, overlay=10+)
2. **Transparency**: Use `background: -1` for transparent overlays
3. **Sections**: Divide 128x40 into logical areas

### Performance

| Content | Recommended Interval |
|---------|---------------------|
| Clock | 1.0s |
| CPU/Memory | 1.0s |
| Network/Disk | 1.0s |
| Keyboard | 0.2s |
| Audio | 0.033s (30 FPS) |

### Visual Design

1. Use borders to separate widgets
2. Consistent alignment
3. Padding (1-2px) from edges
4. Different colors for RX/TX, Read/Write

## Troubleshooting

### Widget Not Showing

Check:
1. `enabled: true` (or omitted)
2. Position within display bounds
3. Non-zero width and height
4. Z-order conflicts

### Configuration Errors

Validate:
- `schema_version: 2` present
- Required fields (type, id, position)
- Valid enum values (mode, align)
- Value ranges (0-255 for colors)

## Additional Resources

- **Schema**: `config.schema.json`
- **Examples**: `configs/examples/` folder
- **Design Doc**: `configs/SCHEMA_V2_DESIGN.md`
