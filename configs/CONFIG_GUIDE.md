# SteelClock Configuration Guide

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
- Virtual canvas / viewport (scrolling)

## JSON Schema

SteelClock includes a comprehensive JSON Schema (`config.schema.json`) that provides:

- **IDE Autocomplete**: Property suggestions while typing
- **Validation**: Real-time error checking
- **Documentation**: Inline descriptions and defaults
- **Type Safety**: Prevents configuration errors

### Enabling Schema in Your Config

Add this line at the top of your configuration file:

```json
{
  "$schema": "./config.schema.json",
  "game_name": "STEELCLOCK",
  ...
}
```

Supported IDEs: VS Code, JetBrains IDEs (IntelliJ, PyCharm, WebStorm), Visual Studio, Sublime Text, and others.

## Configuration Structure

### Top-Level Structure

```json
{
  "$schema": "./config.schema.json",
  "game_name": "STEELCLOCK",
  "game_display_name": "SteelClock",
  "refresh_rate_ms": 100,
  "unregister_on_exit": false,
  "bundled_font_url": "https://github.com/kika/fixedsys/releases/download/v3.02.9/FSEX302.ttf",
  "display": { ... },
  "layout": { ... },
  "widgets": [ ... ]
}
```

### Global Settings

| Property             | Type    | Default                                                                             | Description                                             |
|----------------------|---------|-------------------------------------------------------------------------------------|---------------------------------------------------------|
| `game_name`          | string  | "STEELCLOCK"                                                                        | Game identifier (A-Z, 0-9, -, _ only)                   |
| `game_display_name`  | string  | "SteelClock"                                                                        | Human-readable name                                     |
| `refresh_rate_ms`    | integer | 100                                                                                 | Display refresh rate (min 100ms = 10Hz)                 |
| `unregister_on_exit` | boolean | false                                                                               | Unregister from GameSense API on exit (see notes below) |
| `bundled_font_url`   | string  | "https://github.com/kika/fixedsys/releases/download/v3.02.9/FSEX302.ttf" (optional) | URL for downloading bundled font (see notes below)      |

**About `unregister_on_exit`**:

This option controls whether SteelClock calls the GameSense API `/remove_game` endpoint when shutting down.

- **Default (false)**: Do not unregister on exit. The GameSense registration persists after the application closes.
- **Set to true**: Call `/remove_game` to clean up the registration on exit.

**When to use**:
- **Keep default (false)** in most cases. The GameSense `RegisterGame()` API is idempotent and will overwrite existing registrations on restart. This avoids potential timeout issues with the `/remove_game` endpoint.
- **Set to true** if you want a complete cleanup on exit and are willing to accept potential delays (2-5 seconds) during shutdown due to API timeouts.

**Technical notes**:
- During configuration reload, the game is never unregistered regardless of this setting (to avoid disruption)
- Only affects final application shutdown (via tray menu "Quit")
- The `/remove_game` endpoint can be slow or timeout, which is why the default is false

**About `bundled_font_url`**:

This option allows you to specify a custom URL for downloading the bundled TrueType font when no system font is available.

- **Default**: `https://github.com/kika/fixedsys/releases/download/v3.02.9/FSEX302.ttf`
- **Optional**: This field can be omitted to use the default URL
- **When to use**: If you want to use a different bundled font or host the font on your own server

**Font Loading Behavior**:
1. If a `font` property is specified in widget configuration, SteelClock attempts to load it as:
   - An absolute path to a TTF file
   - A system font name (e.g., "Arial", "Consolas")
   - A mapped Windows font name
2. If the font is not found, SteelClock downloads the bundled font from `bundled_font_url`
3. If the download fails, SteelClock falls back to the built-in basic font (7x13 bitmap font)

**Example**:
```json
{
  "bundled_font_url": "https://example.com/fonts/custom-font.ttf"
}
```

**Notes**:
- The bundled font is downloaded once and cached in the `./fonts/` directory
- The font file must be a valid TrueType font (`.ttf`)
- The URL must be accessible from the machine running SteelClock

### Display Configuration

```json
"display": {
  "width": 128,
  "height": 40,
  "background_color": 0
}
```

| Property           | Type    | Range | Default | Description                           |
|--------------------|---------|-------|---------|---------------------------------------|
| `width`            | integer | -     | 128     | Display width in pixels               |
| `height`           | integer | -     | 40      | Display height in pixels              |
| `background_color` | integer | 0-255 | 0       | Background color (0=black, 255=white) |

### Layout Configuration

#### Basic Layout (Fixed Canvas)

```json
"layout": {
  "type": "basic"
}
```

#### Viewport Layout (Scrolling Canvas)

```json
"layout": {
  "type": "viewport",
  "virtual_width": 256,
  "virtual_height": 80
}
```

| Property         | Type                  | Description                           |
|------------------|-----------------------|---------------------------------------|
| `type`           | "basic" \| "viewport" | Layout mode                           |
| `virtual_width`  | integer               | Virtual canvas width (viewport mode)  |
| `virtual_height` | integer               | Virtual canvas height (viewport mode) |

## Widget Types

SteelClock supports 7 widget types:

1. **clock** - Time display
2. **cpu** - CPU usage monitor
3. **memory** - RAM usage monitor
4. **network** - Network I/O monitor
5. **disk** - Disk I/O monitor
6. **keyboard** - Keyboard lock indicators
7. **volume** - System volume indicator
8. **volume_meter** - Realtime audio peak meter (VU meter)

## Common Properties

All widgets share these common configuration sections:

### Widget Base Structure

```json
{
  "type": "clock",
  "id": "unique_id",
  "enabled": true,
  "position": { ... },
  "style": { ... },
  "properties": { ... }
}
```

| Property  | Type    | Required | Description                           |
|-----------|---------|----------|---------------------------------------|
| `type`    | string  | Yes      | Widget type                           |
| `id`      | string  | Yes      | Unique widget identifier              |
| `enabled` | boolean | No       | Enable/disable widget (default: true) |

### Position Configuration

```json
"position": {
  "x": 0,
  "y": 0,
  "w": 128,
  "h": 40,
  "z_order": 0
}
```

| Property  | Type    | Description                                  |
|-----------|---------|----------------------------------------------|
| `x`       | integer | X coordinate on canvas (pixels)              |
| `y`       | integer | Y coordinate on canvas (pixels)              |
| `w`       | integer | Widget width (pixels)                        |
| `h`       | integer | Widget height (pixels)                       |
| `z_order` | integer | Stacking order (higher = on top, default: 0) |

### Style Configuration

```json
"style": {
  "background_color": 0,
  "background_opacity": 255,
  "border": false,
  "border_color": 255
}
```

| Property             | Type    | Range | Default | Description                                         |
|----------------------|---------|-------|---------|-----------------------------------------------------|
| `background_color`   | integer | 0-255 | 0       | Background color                                    |
| `background_opacity` | integer | 0-255 | 255     | Background transparency (0=transparent, 255=opaque) |
| `border`             | boolean | -     | false   | Draw widget border                                  |
| `border_color`       | integer | 0-255 | 255     | Border color                                        |

### Text Properties

Widgets supporting text mode share these properties:

| Property           | Type    | Options                   | Default  | Description                 |
|--------------------|---------|---------------------------|----------|-----------------------------|
| `font`             | string  | -                         | null     | Font name or TTF file path  |
| `font_size`        | integer | ≥1                        | 10       | Font size in pixels         |
| `horizontal_align` | string  | "left", "center", "right" | "center" | Horizontal alignment        |
| `vertical_align`   | string  | "top", "center", "bottom" | "center" | Vertical alignment          |
| `padding`          | integer | ≥0                        | 0        | Padding from edges (pixels) |

**Font Examples**:
- Font name: `"Arial"`, `"Consolas"`, `"Segoe UI Emoji"`
- TTF path: `"C:/Windows/Fonts/arial.ttf"`

### Auto-Hide Properties

All widgets support auto-hide functionality, which allows widgets to appear temporarily and then hide:

| Property            | Type    | Default | Description                                          |
|---------------------|---------|---------|------------------------------------------------------|
| `auto_hide`         | boolean | false   | Enable auto-hide mode (widget starts hidden)         |
| `auto_hide_timeout` | number  | 2.0     | Seconds to wait before hiding after last trigger     |

**How Auto-Hide Works**:

1. Widget starts hidden when `auto_hide` is enabled
2. Widget becomes visible when triggered (e.g., volume change, notification received)
3. Widget remains visible for `auto_hide_timeout` seconds after last trigger
4. Widget becomes invisible again after timeout expires
5. When hidden, widget returns nil from Render(), allowing widgets below to show through

**Use Cases**:
- Volume indicators that appear only when volume changes
- Notification widgets that appear temporarily
- Status indicators that show only when state changes
- Any widget that should not take permanent screen space

**Widget-Specific Triggers**:
- **Volume Widget**: Triggers when volume level or mute state changes
- **Custom Widgets**: Can call `TriggerAutoHide()` method when content changes

**Example**:
```json
{
  "type": "volume",
  "id": "temp_volume",
  "properties": {
    "auto_hide": true,
    "auto_hide_timeout": 3.0
  }
}
```

## Widget-Specific Properties

### Clock Widget

**Display Modes**: Text only

```json
{
  "type": "clock",
  "properties": {
    "format": "%H:%M:%S",
    "update_interval": 1.0,
    "font": "Arial",
    "font_size": 12,
    "horizontal_align": "center",
    "vertical_align": "center",
    "padding": 0
  }
}
```

| Property          | Type   | Default    | Description                   |
|-------------------|--------|------------|-------------------------------|
| `format`          | string | "%H:%M:%S" | Time format (strftime syntax) |
| `update_interval` | number | 1.0        | Update interval in seconds    |

**Format Examples**:
- `"%H:%M:%S"` → 15:43:27 (24-hour with seconds)
- `"%H:%M"` → 15:43 (24-hour without seconds)
- `"%I:%M %p"` → 03:43 PM (12-hour with AM/PM)
- `"%Y-%m-%d %H:%M"` → 2025-11-14 15:43 (date and time)

See [Python strftime](https://docs.python.org/3/library/datetime.html#strftime-and-strptime-format-codes) for all format codes.

### CPU Widget

**Display Modes**: text, bar_horizontal, bar_vertical, graph, gauge

```json
{
  "type": "cpu",
  "properties": {
    "display_mode": "bar_horizontal",
    "per_core": false,
    "update_interval": 1.0,
    "history_length": 30,
    "bar_border": false,
    "bar_margin": 0,
    "fill_color": 255,
    "gauge_color": 200,
    "gauge_needle_color": 255,
    "font": null,
    "font_size": 10,
    "horizontal_align": "center",
    "vertical_align": "center",
    "padding": 0
  }
}
```

| Property             | Type    | Range                                              | Default          | Description                         |
|----------------------|---------|----------------------------------------------------|------------------|-------------------------------------|
| `display_mode`       | string  | text, bar_horizontal, bar_vertical, graph, gauge   | "bar_horizontal" | Display mode                        |
| `per_core`           | boolean | -                                                  | false            | Show per-core usage                 |
| `update_interval`    | number  | ≥0.1                                               | 1.0              | Update interval (seconds)           |
| `history_length`     | integer | ≥2                                                 | 30               | Samples for graph mode              |
| `bar_border`         | boolean | -                                                  | false            | Draw border around bars             |
| `bar_margin`         | integer | ≥0                                                 | 0                | Margin between bars (per-core mode) |
| `fill_color`         | integer | 0-255                                              | 255              | Bar/graph fill color                |
| `gauge_color`        | integer | 0-255                                              | 200              | Gauge arc and tick marks color      |
| `gauge_needle_color` | integer | 0-255                                              | 255              | Gauge needle color                  |

**Gauge Mode**:

The CPU widget can display usage as an old-fashioned semicircular gauge:
- Semicircular arc from 0% (left) to 100% (right)
- Tick marks at 10% intervals (longer marks at 0%, 50%, 100%)
- Needle points to current CPU usage percentage
- Arc and tick marks drawn using `gauge_color`
- Needle drawn using `gauge_needle_color`
- If `per_core` is enabled, shows average of all cores

### Memory Widget

**Display Modes**: text, bar_horizontal, bar_vertical, graph, gauge

```json
{
  "type": "memory",
  "properties": {
    "display_mode": "bar_horizontal",
    "update_interval": 1.0,
    "history_length": 30,
    "bar_border": false,
    "fill_color": 255,
    "gauge_color": 200,
    "gauge_needle_color": 255,
    "font": null,
    "font_size": 10,
    "horizontal_align": "center",
    "vertical_align": "center",
    "padding": 0
  }
}
```

| Property             | Type    | Range                                              | Default          | Description                    |
|----------------------|---------|----------------------------------------------------|------------------|--------------------------------|
| `display_mode`       | string  | text, bar_horizontal, bar_vertical, graph, gauge   | "bar_horizontal" | Display mode                   |
| `update_interval`    | number  | ≥0.1                                               | 1.0              | Update interval (seconds)      |
| `history_length`     | integer | ≥2                                                 | 30               | Samples for graph mode         |
| `bar_border`         | boolean | -                                                  | false            | Draw border around bar         |
| `fill_color`         | integer | 0-255                                              | 255              | Bar/graph fill color           |
| `gauge_color`        | integer | 0-255                                              | 200              | Gauge arc and tick marks color |
| `gauge_needle_color` | integer | 0-255                                              | 255              | Gauge needle color             |

**Gauge Mode**:

The Memory widget displays RAM usage as a semicircular gauge:
- Semicircular arc from 0% (left) to 100% (right)
- Tick marks at 10% intervals (longer marks at 0%, 50%, 100%)
- Needle points to current memory usage percentage
- Arc and tick marks drawn using `gauge_color`
- Needle drawn using `gauge_needle_color`

### Network Widget

**Display Modes**: text, bar_horizontal, bar_vertical, graph, gauge

```json
{
  "type": "network",
  "properties": {
    "interface": "eth0",
    "display_mode": "bar_horizontal",
    "update_interval": 1.0,
    "history_length": 30,
    "max_speed_mbps": 100.0,
    "speed_unit": "kbps",
    "bar_border": false,
    "bar_margin": 1,
    "rx_color": 255,
    "tx_color": 128,
    "rx_needle_color": 255,
    "tx_needle_color": 200,
    "font": null,
    "font_size": 10,
    "horizontal_align": "center",
    "vertical_align": "center",
    "padding": 0
  }
}
```

| Property           | Type           | Range                                              | Default          | Description                            |
|--------------------|----------------|----------------------------------------------------|------------------|----------------------------------------|
| `interface`        | string or null | -                                                  | "eth0"           | Network interface name (null=auto)     |
| `display_mode`     | string         | text, bar_horizontal, bar_vertical, graph, gauge   | "bar_horizontal" | Display mode                           |
| `update_interval`  | number         | ≥0.1                                               | 1.0              | Update interval (seconds)              |
| `history_length`   | integer        | ≥2                                                 | 30               | Samples for graph mode                 |
| `max_speed_mbps`   | number         | -                                                  | 100.0            | Max speed for scaling (-1=auto)        |
| `speed_unit`       | string         | bps, kbps, mbps                                    | "kbps"           | Speed unit (text mode)                 |
| `bar_border`       | boolean        | -                                                  | false            | Draw border around bars                |
| `bar_margin`       | integer        | ≥0                                                 | 1                | Margin between RX/TX bars              |
| `rx_color`         | integer        | 0-255                                              | 255              | RX (download) arc color (gauge mode)   |
| `tx_color`         | integer        | 0-255                                              | 128              | TX (upload) arc color (gauge mode)     |
| `rx_needle_color`  | integer        | 0-255                                              | 255              | RX needle color (gauge mode)           |
| `tx_needle_color`  | integer        | 0-255                                              | 200              | TX needle color (gauge mode)           |

**Interface Names**:
- `"Ethernet"`, `"Wi-Fi"`, etc. (from Network Connections)
- Use `null` for auto-detection

**Gauge Mode**:

The Network widget features a unique **dual/concentric gauge** display that shows both RX and TX speeds simultaneously:
- **Outer gauge** (larger radius): RX (download) speed
  - Arc drawn using `rx_color`
  - Needle drawn using `rx_needle_color`
  - Needle spans from inner gauge edge to outer edge (doesn't overlap inner gauge)
- **Inner gauge** (60% of outer radius): TX (upload) speed
  - Arc drawn using `tx_color`
  - Needle drawn using `tx_needle_color`
  - Needle spans from center to inner gauge edge
- Both gauges show 0-100% of `max_speed_mbps`
- Semicircular arc (180° span)
- Tick marks at regular intervals
- Independent needles for RX and TX

### Disk Widget

**Display Modes**: text, bar_horizontal, bar_vertical, graph

```json
{
  "type": "disk",
  "properties": {
    "disk_name": "PhysicalDrive0",
    "display_mode": "bar_horizontal",
    "update_interval": 1.0,
    "history_length": 30,
    "max_speed_mbps": -1,
    "bar_border": false,
    "read_color": 255,
    "write_color": 200,
    "font": null,
    "font_size": 10,
    "horizontal_align": "center",
    "vertical_align": "center",
    "padding": 0
  }
}
```

| Property          | Type           | Range                                     | Default          | Description                             |
|-------------------|----------------|-------------------------------------------|------------------|-----------------------------------------|
| `disk_name`       | string or null | -                                         | null             | Disk name (null=auto)                   |
| `display_mode`    | string         | text, bar_horizontal, bar_vertical, graph | "bar_horizontal" | Display mode                            |
| `update_interval` | number         | ≥0.1                                      | 1.0              | Update interval (seconds)               |
| `history_length`  | integer        | ≥2                                        | 30               | Samples for graph mode                  |
| `max_speed_mbps`  | number         | -                                         | -1               | Max speed for scaling in MB/s (-1=auto) |
| `bar_border`      | boolean        | -                                         | false            | Draw border around bars                 |
| `read_color`      | integer        | 0-255                                     | 255              | Read color                              |
| `write_color`     | integer        | 0-255                                     | 200              | Write color                             |

**Disk Names**:
- `"PhysicalDrive0"`, `"PhysicalDrive1"`, ...
- Use `null` for auto-selection

Run SteelClock once to see available disks in logs:
```
Available disks: PhysicalDrive0, PhysicalDrive1
```

### Keyboard Widget

**Display Modes**: Text only (customizable symbols)

```json
{
  "type": "keyboard",
  "properties": {
    "update_interval": 0.2,
    "spacing": 3,
    "caps_lock_on": "⬆",
    "caps_lock_off": "",
    "num_lock_on": "🔒",
    "num_lock_off": "",
    "scroll_lock_on": "⬇",
    "scroll_lock_off": "",
    "indicator_color_on": 255,
    "indicator_color_off": 100,
    "font": "Segoe UI Emoji",
    "font_size": 10,
    "horizontal_align": "center",
    "vertical_align": "center",
    "padding": 2
  }
}
```

| Property              | Type    | Range | Default | Description                             |
|-----------------------|---------|-------|---------|-----------------------------------------|
| `update_interval`     | number  | ≥0.1  | 0.2     | Update interval (seconds)               |
| `spacing`             | integer | ≥0    | 3       | Spacing between indicators (pixels)     |
| `caps_lock_on`        | string  | -     | "⬆"     | Symbol for Caps Lock ON                 |
| `caps_lock_off`       | string  | -     | ""      | Symbol for Caps Lock OFF (empty=hide)   |
| `num_lock_on`         | string  | -     | "🔒"    | Symbol for Num Lock ON                  |
| `num_lock_off`        | string  | -     | ""      | Symbol for Num Lock OFF (empty=hide)    |
| `scroll_lock_on`      | string  | -     | "⬇"     | Symbol for Scroll Lock ON               |
| `scroll_lock_off`     | string  | -     | ""      | Symbol for Scroll Lock OFF (empty=hide) |
| `indicator_color_on`  | integer | 0-255 | 255     | Color for ON state                      |
| `indicator_color_off` | integer | 0-255 | 100     | Color for OFF state                     |

**Emoji Support**: Use `"font": "Segoe UI Emoji"` to display emojis.

**Alternative Symbols**:
- Unicode arrows: ↑ ↓ ▲ ▼
- Text: "CAPS", "NUM", "SCR"
- Letters: "C", "N", "S"

### Volume Widget

**Display Modes**: text, bar_horizontal, bar_vertical, gauge, triangle

```json
{
  "type": "volume",
  "properties": {
    "display_mode": "bar_horizontal",
    "update_interval": 0.1,
    "fill_color": 255,
    "bar_border": false,
    "gauge_color": 200,
    "gauge_needle_color": 255,
    "triangle_fill_color": 255,
    "triangle_border": false,
    "font": null,
    "font_size": 10,
    "horizontal_align": "center",
    "vertical_align": "center",
    "padding": 0,
    "auto_hide": false,
    "auto_hide_timeout": 2.0
  }
}
```

| Property              | Type    | Range                                                  | Default          | Description                                  |
|-----------------------|---------|--------------------------------------------------------|------------------|----------------------------------------------|
| `display_mode`        | string  | text, bar_horizontal, bar_vertical, gauge, triangle    | "bar_horizontal" | Display mode                                 |
| `update_interval`     | number  | ≥0.1                                                   | 0.1              | Update interval (seconds)                    |
| `fill_color`          | integer | 0-255                                                  | 255              | Bar/triangle fill color                      |
| `bar_border`          | boolean | -                                                      | false            | Draw border around bars                      |
| `gauge_color`         | integer | 0-255                                                  | 200              | Gauge arc and tick marks color               |
| `gauge_needle_color`  | integer | 0-255                                                  | 255              | Gauge needle color                           |
| `triangle_fill_color` | integer | 0-255                                                  | 255              | Triangle fill color                          |
| `triangle_border`     | boolean | -                                                      | false            | Draw border around triangle                  |

**Note**: The volume widget also supports `auto_hide` and `auto_hide_timeout` properties (see [Auto-Hide Properties](#auto-hide-properties)). When auto-hide is enabled, the widget triggers visibility on volume or mute state changes.

**Display Mode Details**:

**text**: Shows volume as percentage ("75%"). When muted, shows "MUTE".

**bar_horizontal**: Horizontal bar filling left to right based on volume level.

**bar_vertical**: Vertical bar filling bottom to top based on volume level.

**gauge**: Old-fashioned semicircular gauge with needle pointing to current volume level. Features:
- Semicircular arc (180° span from 0% to 100%)
- Tick marks at 10% intervals
- Longer ticks at 0%, 50%, and 100%
- Needle pointing to current volume
- Center pivot point

**triangle**: Pyramid-style volume indicator filling from bottom to top. Features:
- Triangle shape (wider at bottom, narrower at top)
- Fills based on volume level
- Vertical bar pattern (|||) for filled sections
- Optional border

**Auto-Hide Support**:

The volume widget automatically triggers the auto-hide feature (see [Auto-Hide Properties](#auto-hide-properties)) when volume level or mute state changes. This makes it ideal for temporary on-screen volume indicators.

**Mute Indicator**:

When system audio is muted, all display modes show an X pattern (diagonal lines) over the volume indicator.

### Volume Meter Widget

**Display Modes**: text, bar_horizontal, bar_vertical, gauge (all modes support stereo with `stereo_mode: true` and professional metering with `vu_mode: true`)

```json
{
  "type": "volume_meter",
  "properties": {
    "display_mode": "bar_horizontal",
    "update_interval": 0.1,
    "fill_color": 255,
    "clipping_color": 200,
    "left_channel_color": 255,
    "right_channel_color": 200,
    "stereo_mode": false,
    "vu_mode": false,
    "bar_border": false,
    "gauge_color": 200,
    "gauge_needle_color": 255,
    "use_db_scale": false,
    "show_clipping": true,
    "clipping_threshold": 0.99,
    "silence_threshold": 0.01,
    "decay_rate": 2.0,
    "show_peak": false,
    "show_peak_hold": true,
    "peak_hold_time": 1.0,
    "auto_hide_on_silence": false,
    "auto_hide_silence_time": 2.0,
    "font": null,
    "font_size": 10,
    "horizontal_align": "center",
    "vertical_align": "center",
    "padding": 0,
    "auto_hide": false,
    "auto_hide_timeout": 2.0
  }
}
```

| Property                 | Type    | Range                                                                         | Default          | Description                                            |
|--------------------------|---------|-------------------------------------------------------------------------------|------------------|--------------------------------------------------------|
| `display_mode`           | string  | text, bar_horizontal, bar_vertical, gauge                                     | "bar_horizontal" | Display mode                                           |
| `update_interval`        | number  | ≥0.03                                                                         | 0.1              | Meter update interval (seconds)                        |
| `fill_color`             | integer | 0-255                                                                         | 255              | Main meter fill color                                  |
| `clipping_color`         | integer | 0-255                                                                         | 200              | Color when clipping detected                           |
| `left_channel_color`     | integer | 0-255                                                                         | 255              | Left channel color (when stereo_mode enabled)          |
| `right_channel_color`    | integer | 0-255                                                                         | 200              | Right channel color (when stereo_mode enabled)         |
| `stereo_mode`            | boolean | -                                                                             | false            | Display left and right channels separately             |
| `vu_mode`                | boolean | -                                                                             | false            | Enable professional VU metering with visible scales    |
| `bar_border`             | boolean | -                                                                             | false            | Draw border around bars                                |
| `gauge_color`            | integer | 0-255                                                                         | 200              | Gauge arc and tick marks color                         |
| `gauge_needle_color`     | integer | 0-255                                                                         | 255              | Gauge needle color                                     |
| `use_db_scale`           | boolean | -                                                                             | false            | Use logarithmic dB scale (-60dB to 0dB)                |
| `show_clipping`          | boolean | -                                                                             | true             | Show clipping indicator (all modes)                    |
| `clipping_threshold`     | number  | 0.0-1.0                                                                       | 0.99             | Peak level that triggers clipping (0.0=0%, 1.0=100%)   |
| `silence_threshold`      | number  | 0.0-1.0                                                                       | 0.01             | Peak level below which is considered silence           |
| `decay_rate`             | number  | ≥0.1                                                                          | 2.0              | Peak decay rate (units/second, VU meter ballistics)    |
| `show_peak`              | boolean | -                                                                             | false            | Show instantaneous peak line (current actual peak)     |
| `show_peak_hold`         | boolean | -                                                                             | true             | Show peak hold line (held maximum peak)                |
| `peak_hold_time`         | number  | ≥0.1                                                                          | 1.0              | How long to hold peak indicator (seconds)              |
| `auto_hide_on_silence`   | boolean | -                                                                             | false            | Auto-hide when no audio detected                       |
| `auto_hide_silence_time` | number  | ≥0.5                                                                          | 2.0              | Time after last audio before hiding (seconds)          |

**Note**: The volume meter widget also supports `auto_hide` and `auto_hide_timeout` properties (see [Auto-Hide Properties](#auto-hide-properties)). When `auto_hide_on_silence` is enabled, the widget triggers visibility when audio is detected above the `silence_threshold`.

**Display Mode Details**:

**text**: Shows peak level as percentage or dB. When `use_db_scale` is true, displays dB value (e.g., "-12.3 dB"). Shows "CLIP" when clipping is detected.
- **Stereo mode**: Displays both channels (e.g., "L:45% R:52%")

**bar_horizontal**: Horizontal bar showing current audio peak level with smooth decay.
- **Stereo mode**: Two horizontal bars stacked (top = left channel, bottom = right channel)
- **VU mode**: Adds visible scale tick marks at key positions (0%, 70%, 90%, 100% or -60dB, -20dB, -10dB, -3dB, 0dB)
- **show_peak**: Bright line showing instantaneous actual peak (faster than decayed bar)
- **show_peak_hold**: Line showing held maximum peak

**bar_vertical**: Vertical bar showing current audio peak level with smooth decay.
- **Stereo mode**: Two vertical bars side by side (left = left channel, right = right channel)
- **VU mode**: Adds visible scale tick marks on the right edge
- **show_peak**: Bright line showing instantaneous actual peak
- **show_peak_hold**: Line showing held maximum peak

**gauge**: Semicircular gauge with needle pointing to current peak level.
- **Stereo mode**: Two gauges side by side (left = left channel, right = right channel)
- **show_clipping**: Changes needle color to red when clipping detected
- **VU mode**: (Future: adds colored arc segments for green/yellow/red zones)

**Peak Decay Behavior**:

The meter features smooth peak decay (VU meter ballistics) controlled by `decay_rate`:
- Peak values decay gradually after audio quiets
- `decay_rate`: units per second to decay (default 2.0 means full scale decay in 0.5 seconds)
- Instant rise time when audio increases
- Creates smooth, professional-looking meters

**Clipping Detection**:

When audio peaks exceed `clipping_threshold` (default 0.99 = 99% of maximum):
- Meter changes to `clipping_color`
- Text mode shows "CLIP" message
- Helps identify potential audio distortion

**dB Scale Mode**:

When `use_db_scale` is enabled:
- Linear peak values (0.0-1.0) are converted to logarithmic dB scale (-60dB to 0dB)
- More accurate representation of perceived loudness
- Professional audio application standard
- 0dB = maximum (1.0), -60dB = minimum audible (0.001)

**Auto-Hide on Silence**:

When `auto_hide_on_silence` is enabled:
- Widget triggers auto-hide when audio detected above `silence_threshold`
- Automatically hides after `auto_hide_silence_time` seconds of silence
- Perfect for temporary audio level indicators
- Works with standard `auto_hide` and `auto_hide_timeout` properties

## Examples

### Example 1: Simple Clock

```json
{
  "$schema": "./config.schema.json",
  "game_name": "STEELCLOCK",
  "game_display_name": "SteelClock",
  "refresh_rate_ms": 100,
  "display": {
    "width": 128,
    "height": 40,
    "background_color": 0
  },
  "layout": {
    "type": "basic"
  },
  "widgets": [
    {
      "type": "clock",
      "id": "main_clock",
      "enabled": true,
      "position": {"x": 0, "y": 0, "w": 128, "h": 40, "z_order": 0},
      "style": {"background_color": 0, "border": false},
      "properties": {
        "format": "%H:%M:%S",
        "font_size": 12,
        "update_interval": 1.0
      }
    }
  ]
}
```

### Example 2: CPU + Memory Bars

```json
{
  "$schema": "./config.schema.json",
  "game_name": "STEELCLOCK",
  "display": {"width": 128, "height": 40, "background_color": 0},
  "layout": {"type": "basic"},
  "widgets": [
    {
      "type": "cpu",
      "id": "cpu_bar",
      "position": {"x": 0, "y": 0, "w": 128, "h": 20, "z_order": 0},
      "style": {"background_color": 0, "border": true, "border_color": 255},
      "properties": {
        "display_mode": "bar_horizontal",
        "per_core": false,
        "padding": 2
      }
    },
    {
      "type": "memory",
      "id": "memory_bar",
      "position": {"x": 0, "y": 20, "w": 128, "h": 20, "z_order": 0},
      "style": {"background_color": 0, "border": true, "border_color": 255},
      "properties": {
        "display_mode": "bar_horizontal",
        "padding": 2
      }
    }
  ]
}
```

### Example 3: Network Graph with Dynamic Scaling

```json
{
  "$schema": "./config.schema.json",
  "game_name": "STEELCLOCK",
  "display": {"width": 128, "height": 40, "background_color": 0},
  "layout": {"type": "basic"},
  "widgets": [
    {
      "type": "network",
      "id": "net_graph",
      "position": {"x": 0, "y": 0, "w": 128, "h": 40, "z_order": 0},
      "style": {"background_color": 0, "border": true, "border_color": 255},
      "properties": {
        "interface": null,
        "display_mode": "graph",
        "max_speed_mbps": -1,
        "history_length": 30,
        "padding": 2,
        "rx_color": 255,
        "tx_color": 128
      }
    }
  ]
}
```

### Example 4: Complete Dashboard

```json
{
  "$schema": "./config.schema.json",
  "game_name": "STEELCLOCK",
  "game_display_name": "System Monitor",
  "refresh_rate_ms": 100,
  "display": {"width": 128, "height": 40, "background_color": 0},
  "layout": {"type": "basic"},
  "widgets": [
    {
      "type": "clock",
      "id": "time",
      "position": {"x": 0, "y": 0, "w": 96, "h": 8, "z_order": 0},
      "style": {"background_color": 0, "border": false},
      "properties": {"format": "%H:%M:%S", "font_size": 7}
    },
    {
      "type": "keyboard",
      "id": "keys",
      "position": {"x": 96, "y": 0, "w": 32, "h": 8, "z_order": 0},
      "style": {"background_color": 0, "border": false},
      "properties": {
        "font": "Segoe UI Emoji",
        "font_size": 6,
        "spacing": 2
      }
    },
    {
      "type": "cpu",
      "id": "cpu",
      "position": {"x": 0, "y": 8, "w": 32, "h": 8, "z_order": 0},
      "style": {"background_color": 0, "border": true, "border_color": 255},
      "properties": {"display_mode": "bar_horizontal", "padding": 1}
    },
    {
      "type": "memory",
      "id": "mem",
      "position": {"x": 32, "y": 8, "w": 32, "h": 8, "z_order": 0},
      "style": {"background_color": 0, "border": true, "border_color": 255},
      "properties": {"display_mode": "bar_horizontal", "padding": 1}
    },
    {
      "type": "disk",
      "id": "disk",
      "position": {"x": 64, "y": 8, "w": 64, "h": 8, "z_order": 0},
      "style": {"background_color": 0, "border": true, "border_color": 255},
      "properties": {
        "disk_name": "PhysicalDrive0",
        "display_mode": "bar_horizontal",
        "padding": 1
      }
    },
    {
      "type": "network",
      "id": "net",
      "position": {"x": 0, "y": 16, "w": 128, "h": 24, "z_order": 0},
      "style": {"background_color": 0, "border": true, "border_color": 255},
      "properties": {
        "display_mode": "graph",
        "max_speed_mbps": -1,
        "padding": 2
      }
    }
  ]
}
```

### Example 5: Transparent Overlay

```json
{
  "$schema": "./config.schema.json",
  "game_name": "STEELCLOCK",
  "display": {"width": 128, "height": 40, "background_color": 0},
  "layout": {"type": "basic"},
  "widgets": [
    {
      "type": "network",
      "id": "background",
      "position": {"x": 0, "y": 0, "w": 128, "h": 40, "z_order": 0},
      "style": {"background_color": 0, "background_opacity": 255},
      "properties": {"display_mode": "graph"}
    },
    {
      "type": "clock",
      "id": "overlay",
      "position": {"x": 0, "y": 0, "w": 128, "h": 40, "z_order": 10},
      "style": {"background_color": 0, "background_opacity": 128},
      "properties": {"format": "%H:%M", "font_size": 16}
    }
  ]
}
```

### Example 6: Per-Core CPU Display

```json
{
  "$schema": "./config.schema.json",
  "game_name": "STEELCLOCK",
  "display": {"width": 128, "height": 40, "background_color": 0},
  "layout": {"type": "basic"},
  "widgets": [
    {
      "type": "cpu",
      "id": "cpu_cores",
      "position": {"x": 0, "y": 0, "w": 128, "h": 40, "z_order": 0},
      "style": {"background_color": 0, "border": true, "border_color": 255},
      "properties": {
        "display_mode": "graph",
        "per_core": true,
        "history_length": 30,
        "bar_margin": 1,
        "padding": 2
      }
    }
  ]
}
```

### Example 7: Multiple Disk Monitors

```json
{
  "$schema": "./config.schema.json",
  "game_name": "STEELCLOCK",
  "display": {"width": 128, "height": 40, "background_color": 0},
  "layout": {"type": "basic"},
  "widgets": [
    {
      "type": "disk",
      "id": "disk_c",
      "position": {"x": 0, "y": 0, "w": 128, "h": 20, "z_order": 0},
      "style": {"background_color": 0, "border": true, "border_color": 255},
      "properties": {
        "disk_name": "PhysicalDrive0",
        "display_mode": "bar_horizontal",
        "padding": 2
      }
    },
    {
      "type": "disk",
      "id": "disk_d",
      "position": {"x": 0, "y": 20, "w": 128, "h": 20, "z_order": 0},
      "style": {"background_color": 0, "border": true, "border_color": 255},
      "properties": {
        "disk_name": "PhysicalDrive1",
        "display_mode": "bar_horizontal",
        "padding": 2
      }
    }
  ]
}
```

### Example 8: Gauge Mode Dashboard

```json
{
  "$schema": "./config.schema.json",
  "game_name": "STEELCLOCK",
  "game_display_name": "System Gauges",
  "refresh_rate_ms": 100,
  "display": {"width": 128, "height": 40, "background_color": 0},
  "layout": {"type": "basic"},
  "widgets": [
    {
      "type": "cpu",
      "id": "cpu_gauge",
      "position": {"x": 0, "y": 0, "w": 64, "h": 40, "z_order": 0},
      "style": {"background_color": 0, "border": true, "border_color": 255},
      "properties": {
        "display_mode": "gauge",
        "gauge_color": 200,
        "gauge_needle_color": 255,
        "padding": 2
      }
    },
    {
      "type": "memory",
      "id": "memory_gauge",
      "position": {"x": 64, "y": 0, "w": 64, "h": 40, "z_order": 0},
      "style": {"background_color": 0, "border": true, "border_color": 255},
      "properties": {
        "display_mode": "gauge",
        "gauge_color": 180,
        "gauge_needle_color": 255,
        "padding": 2
      }
    }
  ]
}
```

### Example 9: Network Dual Gauge

```json
{
  "$schema": "./config.schema.json",
  "game_name": "STEELCLOCK",
  "display": {"width": 128, "height": 40, "background_color": 0},
  "layout": {"type": "basic"},
  "widgets": [
    {
      "type": "network",
      "id": "net_gauge",
      "position": {"x": 0, "y": 0, "w": 128, "h": 40, "z_order": 0},
      "style": {"background_color": 0, "border": true, "border_color": 255},
      "properties": {
        "display_mode": "gauge",
        "max_speed_mbps": 100.0,
        "rx_color": 255,
        "tx_color": 180,
        "rx_needle_color": 255,
        "tx_needle_color": 200,
        "padding": 2
      }
    }
  ]
}
```

## Tips and Best Practices

### Layout Design

1. **Split Sections**: Divide the 128x40 display into logical sections
   - Top bar: 128x8 for clock and status
   - Middle row: Multiple 32x8 or 64x8 sections for metrics
   - Bottom section: 128x24 for graphs

2. **Z-Order**: Use z-order for overlays
   - Background widgets: z_order = 0
   - Overlay widgets: z_order = 10+

3. **Transparency**: Use `background_opacity` for overlays
   - Full opacity (255): Normal widgets
   - Partial (128): Overlay text over graphs
   - Transparent (0): Invisible background

### Performance

1. **Update Intervals**:
   - Fast changing: 0.2-0.5s (keyboard, network)
   - Medium: 1.0s (CPU, memory, disk)
   - Slow: 5.0s+ (clock, static info)

2. **Graph History**:
   - Short (10-15): Quick response, less memory
   - Medium (30): Default, good balance
   - Long (60+): Smooth but more memory

3. **Display Mode**:
   - Text: Lowest CPU usage
   - Bars: Medium CPU usage
   - Graphs: Highest CPU usage (history tracking)

### Visual Design

1. **Borders**: Use borders to visually separate widgets
2. **Colors**: Use different colors for RX/TX, Read/Write to distinguish
3. **Alignment**: Consistent alignment creates professional look
4. **Padding**: Use padding (1-2px) to prevent content from touching edges

### Fonts

1. **Standard Text**: Arial, Consolas (monospace)
2. **Emojis**: Segoe UI Emoji
3. **Size**: 6-8px for dense dashboards, 10-16px for readability

## Troubleshooting

### Configuration Errors

**Schema validation errors**: Check:
- Required fields present (type, id, position)
- Valid enum values (display_mode, alignment)
- Correct data types (integers vs strings)
- Value ranges (0-255 for colors)

### Widget Not Showing

Check:
1. `enabled: true`
2. Position within display bounds
3. Non-zero width and height
4. No overlapping widgets with higher z_order

### Graph Not Scrolling

Ensure:
- `history_length` ≥ 2
- `update_interval` appropriate (not too slow)
- Widget has width for multiple samples

### Emoji Not Displaying

Solution:
- Set `font: "Segoe UI Emoji"` (Windows)
- Emojis render as monochrome (display limitation)

## Additional Resources

- **JSON Schema**: `config.schema.json` - Complete validation schema
- **Examples**: `configs/` folder - Multiple example configurations
- **Main Documentation**: `README.md` - User guide
- **Development Notes**: `NOTES.md` - Technical details
