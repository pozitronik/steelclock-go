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
  ...
}
```

Supported IDEs: VS Code, JetBrains IDEs (IntelliJ, PyCharm, WebStorm), Visual Studio, Sublime Text, and others.

## Configuration Structure

### Top-Level Structure

```json
{
  "$schema": "./config.schema.json",
  "refresh_rate_ms": 100,
  "unregister_on_exit": false,
  "deinitialize_timer_length_ms": 15000,
  "supported_resolutions": [
    {"width": 128, "height": 36},
    {"width": 128, "height": 48}
  ],
  "bundled_font_url": "https://github.com/kika/fixedsys/releases/download/v3.02.9/FSEX302.ttf",
  "display": { ... },
  "layout": { ... },
  "widgets": [ ... ]
}
```

### Global Settings

| Property                       | Type    | Default                                                                             | Description                                                               |
|--------------------------------|---------|-------------------------------------------------------------------------------------|---------------------------------------------------------------------------|
| `refresh_rate_ms`              | integer | 100                                                                                 | Display refresh rate (see notes below)                                    |
| `backend`                      | string  | "gamesense"                                                                         | Backend mode: "gamesense", "direct", or "any" (see notes below)           |
| `direct_driver`                | object  | {} (optional)                                                                       | Direct USB driver configuration (see notes below)                         |
| `unregister_on_exit`           | boolean | false                                                                               | Unregister from GameSense API on exit (see notes below)                   |
| `deinitialize_timer_length_ms` | integer | 15000 (optional)                                                                    | Game deactivation timeout in milliseconds (see notes below)               |
| `supported_resolutions`        | array   | [] (optional)                                                                       | Additional display resolutions for multi-device support (see notes below) |
| `bundled_font_url`             | string  | "https://github.com/kika/fixedsys/releases/download/v3.02.9/FSEX302.ttf" (optional) | URL for downloading bundled font (see notes below)                        |

**About `refresh_rate_ms`**:

This option controls how often frames are sent to the display.

- **Default**: 100ms (10Hz)
- **Minimum**: Depends on backend mode (see below)

| Backend Mode | Minimum | Maximum Tested | Notes |
|--------------|---------|----------------|-------|
| `gamesense`  | 100ms (10Hz) | 100ms (10Hz) | Limited by GameSense API |
| `direct`     | ~16ms (60Hz) | 30ms (33Hz) | Limited by USB HID and device |

**Performance notes**:
- With `direct` backend, refresh rates of 30-33ms (30Hz+) work reliably
- Higher refresh rates increase CPU usage proportionally
- The OLED panel itself may have refresh rate limitations
- For animations (clock milliseconds, visualizers), lower values provide smoother display

**About `backend`**:

This option selects how SteelClock communicates with the OLED display.

| Value | Description |
|-------|-------------|
| `gamesense` | Use SteelSeries GameSense API (default). Requires SteelSeries GG to be running. |
| `direct` | Use direct USB HID communication. No SteelSeries GG required. Windows only. |
| `any` | Try `gamesense` first, fall back to `direct` if unavailable. |

**When to use each mode**:

- **`gamesense`** (default): Best compatibility, works with all SteelSeries OLED devices, supports multi-device setups
- **`direct`**: Higher refresh rates (30Hz+), works without SteelSeries GG, lower latency
- **`any`**: Automatic fallback - useful if SteelSeries GG may or may not be running

**Limitations of `direct` mode**:
- Windows only (Linux not yet supported)
- Single device only (no multi-device support)
- Device must be a known SteelSeries keyboard with OLED (Apex 7, Apex Pro, etc.)

**About `direct_driver`**:

Configuration for direct USB HID driver (used when `backend` is `direct` or `any`).

```json
"direct_driver": {
  "vid": "1038",
  "pid": "1612",
  "interface": "mi_01"
}
```

| Property    | Type   | Default | Description |
|-------------|--------|---------|-------------|
| `vid`       | string | auto    | USB Vendor ID in hex (e.g., "1038" for SteelSeries) |
| `pid`       | string | auto    | USB Product ID in hex (e.g., "1612" for Apex 7) |
| `interface` | string | "mi_01" | USB interface identifier |

**Auto-detection**: If `vid` and `pid` are not specified, SteelClock auto-detects from known devices:

| Device | VID | PID |
|--------|-----|-----|
| Apex 7 | 1038 | 1612 |
| Apex 7 TKL | 1038 | 1618 |
| Apex Pro | 1038 | 1610 |
| Apex Pro TKL | 1038 | 1614 |
| Apex 5 | 1038 | 161C |
| Apex Pro (2023) | 1038 | 1630 |
| Apex Pro TKL (2023) | 1038 | 1632 |

**Example configurations**:

*Direct mode with auto-detection:*
```json
{
  "backend": "direct",
  "refresh_rate_ms": 33
}
```

*Direct mode with explicit device:*
```json
{
  "backend": "direct",
  "refresh_rate_ms": 33,
  "direct_driver": {
    "vid": "1038",
    "pid": "1612"
  }
}
```

*Automatic fallback mode:*
```json
{
  "backend": "any",
  "refresh_rate_ms": 50
}
```

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

**About `deinitialize_timer_length_ms`**:

This option controls how long the GameSense API keeps the application active after the last event is sent (in milliseconds).

- **Default**: 15000ms (15 seconds) - used by GameSense if not specified
- **Valid range**: 1000-60000 (1-60 seconds)
- **Optional**: This field can be omitted to use the default behavior

**When to use**:
- **Increase the value** (e.g., 30000-60000) if you want the display to stay active longer after SteelClock stops sending events
- **Decrease the value** (e.g., 1000-5000) if you want the display to clear quickly after the application exits or becomes inactive
- **Omit the field** to use GameSense's default 15-second timeout

**Technical notes**:
- This setting is sent to the GameSense API during game registration
- It affects when the OLED display automatically clears after the last frame is sent
- Useful for customizing the user experience when SteelClock is paused or exits unexpectedly

**About `supported_resolutions`**:

This option enables multi-device support by rendering frames at multiple resolutions simultaneously. All resolution variants are sent in a single frame update.

- **Default**: Empty array (only main `display` resolution is used)
- **Format**: Array of objects with `width` and `height` properties
- **Example**:
```json
"supported_resolutions": [
  {"width": 128, "height": 36},
  {"width": 128, "height": 48},
  {"width": 128, "height": 52}
]
```

**Known SteelSeries Device Resolutions**:

| Resolution | Devices                                                  |
|------------|----------------------------------------------------------|
| **128x36** | SteelSeries Rival 700, Rival 710 (mouse)                 |
| **128x40** | SteelSeries APEX 7 (keyboard)                            |
| **128x48** | SteelSeries Arctis Pro Wireless (headset)                |
| **128x52** | SteelSeries GameDAC, Arctis Pro + GameDAC (audio device) |

**How it works**:
- SteelClock renders the widget canvas at the main `display` resolution
- The same canvas is then scaled/rendered at each `supported_resolutions` entry
- All resolution variants are sent in a single GameSense API frame update
- Connected devices will display the appropriate resolution for their screen

**When to use**:
- **Use this** if you have multiple SteelSeries OLED devices (e.g., APEX 7 + Arctis Pro)
- **Skip this** if you only have one device (just configure `display.width` and `display.height`)

**Example configurations**:

*For APEX 7 keyboard + Arctis Pro headset:*
```json
{
  "display": {"width": 128, "height": 40},  // Main resolution (APEX 7)
  "supported_resolutions": [
    {"width": 128, "height": 48}  // Arctis Pro Wireless
  ]
}
```

*For all SteelSeries OLED devices:*
```json
{
  "display": {"width": 128, "height": 40},
  "supported_resolutions": [
    {"width": 128, "height": 36},  // Rival 700/710
    {"width": 128, "height": 48},  // Arctis Pro Wireless
    {"width": 128, "height": 52}   // GameDAC
  ]
}
```

**Technical notes**:
- Each resolution is independently rendered from the same widget canvas
- Rendering overhead scales linearly with number of resolutions
- GameSense API automatically routes each resolution to matching devices
- No device detection required - the API handles device matching

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
2. If the `font` property is omitted or the specified font is not found, SteelClock downloads the bundled font from `bundled_font_url`
3. If the download fails, SteelClock falls back to the built-in basic font (7x13 bitmap font)

**Recommendation**: Omit the `font` property (or set it to `null`) to use the default bundled font, which provides good readability on small OLED displays.

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
  "border": false,
  "border_color": 255
}
```

| Property           | Type    | Range     | Default | Description                                           |
|--------------------|---------|-----------|---------|-------------------------------------------------------|
| `background_color` | integer | -1 to 255 | 0       | Background color (-1=transparent, 0=black, 255=white) |
| `border`           | boolean | -         | false   | Draw widget border                                    |
| `border_color`     | integer | 0-255     | 255     | Border color                                          |

**Transparent Backgrounds**:

Set `background_color: -1` to make a widget's background transparent, allowing underlying widgets to show through. This is useful for overlaying text or indicators on top of graphs, gauges, or other widgets.

**How transparency works**:
- Widgets with `background_color: -1` only render their foreground pixels (text, shapes, etc.)
- Background pixels (value 0 = black) are skipped during compositing
- Widgets are layered according to their `z_order` (higher values on top)

**Example - Text overlay on graph**:
```json
{
  "widgets": [
    {
      "type": "cpu",
      "id": "cpu_graph",
      "position": {"x": 0, "y": 0, "w": 128, "h": 40, "z_order": 0},
      "style": {"background_color": 0},
      "properties": {"display_mode": "graph"}
    },
    {
      "type": "cpu",
      "id": "cpu_text",
      "position": {"x": 0, "y": 0, "w": 128, "h": 40, "z_order": 1},
      "style": {"background_color": -1},
      "properties": {
        "display_mode": "text",
        "horizontal_align": "left",
        "vertical_align": "top"
      }
    }
  ]
}
```

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

**Display Modes**: text, clock_face

#### Text Mode (Default)
Displays time as formatted text.

```json
{
  "type": "clock",
  "properties": {
    "display_mode": "text",
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

| Property          | Type   | Default    | Description                                   |
|-------------------|--------|------------|-----------------------------------------------|
| `display_mode`    | string | "text"     | Display mode: text or clock_face              |
| `format`          | string | "%H:%M:%S" | Time format (strftime syntax, text mode only) |
| `update_interval` | number | 1.0        | Update interval in seconds                    |

**Format Examples** (text mode):
- `"%H:%M:%S"` → 15:43:27 (24-hour with seconds)
- `"%H:%M"` → 15:43 (24-hour without seconds)
- `"%I:%M %p"` → 03:43 PM (12-hour with AM/PM)
- `"%Y-%m-%d %H:%M"` → 2025-11-14 15:43 (date and time)

See [Python strftime](https://docs.python.org/3/library/datetime.html#strftime-and-strptime-format-codes) for all format codes.

#### Clock Face Mode
Displays an analog clock face with hour, minute, and second hands.

```json
{
  "type": "clock",
  "properties": {
    "display_mode": "clock_face"
  }
}
```

**Features**:
- Circular clock face with border (using `border_color` from style)
- 12 hour markers with longer ticks at 12, 3, 6, 9 o'clock positions
- Hour hand (50% of radius) with fractional positioning based on minutes
- Minute hand (75% of radius) with fractional positioning based on seconds
- Second hand (90% of radius)
- Center dot marker
- Real-time updates

**Recommendations for Clock Face Mode**:
- Use square dimensions (e.g., 40x40, 60x60, 80x80) for best appearance
- Set `refresh_rate_ms` to 1000 for smooth second hand movement
- Use high contrast colors (e.g., `border_color: 255`, `background_color: 0`)
- All clock elements use the `border_color` from the widget style
- Alignment properties (`horizontal_align`, `vertical_align`, `padding`) work with clock face mode
- For non-square widgets, use alignment to position the clock face within the widget bounds

See `configs/examples/clock_face_example.json` and `configs/examples/CLOCK_FACE_README.md` for more details.

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
    "core_border": false,
    "core_margin": 0,
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

| Property             | Type    | Range                                            | Default          | Description                         |
|----------------------|---------|--------------------------------------------------|------------------|-------------------------------------|
| `display_mode`       | string  | text, bar_horizontal, bar_vertical, graph, gauge | "bar_horizontal" | Display mode                        |
| `per_core`           | boolean | -                                                | false            | Show per-core usage                 |
| `update_interval`    | number  | ≥0.1                                             | 1.0              | Update interval (seconds)           |
| `history_length`     | integer | ≥2                                               | 30               | Samples for graph mode              |
| `core_border`        | boolean | -                                                | false            | Draw border around bar              |
| `core_margin`        | integer | ≥0                                               | 0                | Margin between bars (per-core mode) |
| `fill_color`         | integer | 0-255                                            | 255              | Bar/graph fill color                |
| `gauge_color`        | integer | 0-255                                            | 200              | Gauge arc and tick marks color      |
| `gauge_needle_color` | integer | 0-255                                            | 255              | Gauge needle color                  |

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

| Property          | Type           | Range                                            | Default          | Description                          |
|-------------------|----------------|--------------------------------------------------|------------------|--------------------------------------|
| `interface`       | string or null | -                                                | "eth0"           | Network interface name (null=auto)   |
| `display_mode`    | string         | text, bar_horizontal, bar_vertical, graph, gauge | "bar_horizontal" | Display mode                         |
| `update_interval` | number         | ≥0.1                                             | 1.0              | Update interval (seconds)            |
| `history_length`  | integer        | ≥2                                               | 30               | Samples for graph mode               |
| `max_speed_mbps`  | number         | -                                                | 100.0            | Max speed for scaling (-1=auto)      |
| `speed_unit`      | string         | bps, kbps, mbps                                  | "kbps"           | Speed unit (text mode)               |
| `bar_border`      | boolean        | -                                                | false            | Draw border around bar               |
| `rx_color`        | integer        | 0-255                                            | 255              | RX (download) arc color (gauge mode) |
| `tx_color`        | integer        | 0-255                                            | 128              | TX (upload) arc color (gauge mode)   |
| `rx_needle_color` | integer        | 0-255                                            | 255              | RX needle color (gauge mode)         |
| `tx_needle_color` | integer        | 0-255                                            | 200              | TX needle color (gauge mode)         |

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
| `bar_border`      | boolean        | -                                         | false            | Draw border around bar                  |
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

**Display Modes**: Auto-detected - Text (customizable symbols) or Icon (graphical icons)

The widget automatically detects the display mode based on indicator configuration:
- **Text mode**: Used when ANY `*_lock_on/off` property is explicitly defined
- **Icon mode**: Used when ALL `*_lock_on/off` properties are omitted

**Text Mode Example:**
```json
{
  "type": "keyboard",
  "properties": {
    "update_interval": 0.2,
    "spacing": 3,
    "separator": " ",
    "caps_lock_on": "C",  
    "caps_lock_off": "c",
    "num_lock_on": "N",
    "num_lock_off": "n",
    "scroll_lock_on": "S",
    "scroll_lock_off": "s",
    "indicator_color_on": 255,
    "indicator_color_off": 100,
    "font": "Consolas",
    "font_size": 10,
    "horizontal_align": "center",
    "vertical_align": "center",
    "padding": 2
  }
}
```

**Icon Mode Example** (omit all indicator properties):
```json
{
  "type": "keyboard",
  "properties": {
    "update_interval": 0.2,
    "spacing": 4,
    "indicator_color_on": 255,
    "indicator_color_off": 80,
    "horizontal_align": "center",
    "vertical_align": "center",
    "padding": 2
  }
}
```

**Configuration Properties:**

| Property                  | Type    | Range       | Default  | Description                                          |
|---------------------------|---------|-------------|----------|------------------------------------------------------|
| `update_interval`         | number  | ≥0.1        | 0.2      | Update interval (seconds)                            |
| `spacing`                 | integer | ≥0          | 3        | Spacing between indicators (pixels)                  |
| `separator`               | string  | -           | ""       | Separator between indicators (text mode only)        |
| `caps_lock_on`            | string  | -           | "C"      | Symbol for Caps Lock ON (omit for icon mode)         |
| `caps_lock_off`           | string  | -           | "c"      | Symbol for Caps Lock OFF (omit for icon mode)        |
| `num_lock_on`             | string  | -           | "N"      | Symbol for Num Lock ON (omit for icon mode)          |
| `num_lock_off`            | string  | -           | "n"      | Symbol for Num Lock OFF (omit for icon mode)         |
| `scroll_lock_on`          | string  | -           | "S"      | Symbol for Scroll Lock ON (omit for icon mode)       |
| `scroll_lock_off`         | string  | -           | "s"      | Symbol for Scroll Lock OFF (omit for icon mode)      |
| `indicator_color_on`      | integer | 0-255       | 255      | Color for ON state                                   |
| `indicator_color_off`     | integer | 0-255       | 100      | Color for OFF state                                  |

**Separator Examples**:
- `""` (default): Condensed output → `cns` or `CNS`
- `" "`: Spaced output → `c n s` or `C N S`
- `"|"`: Pipe-separated → `c|n|s` or `C|N|S`

**Empty String Behavior**:
- **Omit key** from config → Use default symbol (e.g., `"C"` for caps_lock_on)
- **Set to `""`** explicitly → Hide that indicator (e.g., `"caps_lock_on": ""` hides caps when on)
- **Set to value** → Use that value (e.g., `"caps_lock_on": "⬆"`)

**Symbol Examples** (Text Mode):
- Default letters: `"C"`, `"N"`, `"S"` (uppercase for ON, lowercase for OFF)
- Unicode arrows: `"↑"`, `"↓"`, `"▲"`, `"▼"`
- Text labels: `"CAPS"`, `"NUM"`, `"SCR"`
- Emoji (requires emoji font): `"⬆"`, `"🔒"`, `"⬇"`

**Icon Mode Details**:

When all `*_lock_on/off` properties are omitted, the widget displays graphical icons:
- **Caps Lock**: Up arrow (bright when ON, dim when OFF) - indicates uppercase
- **Num Lock**: Closed lock (ON) / Open lock (OFF) - lock indicator
- **Scroll Lock**: Down arrow (bright when ON, dim when OFF) - scroll indicator

**Icon Size**: Automatically calculated based on widget dimensions
- Widget tries sizes in descending order: 16px → 12px → 8px
- Selects the largest size that fits within available space
- Respects `padding` and `spacing` when calculating fit

**Properties used in Icon Mode**:
- **Respected**: `horizontal_align`, `vertical_align`, `padding`, `spacing`, `indicator_color_on`, `indicator_color_off`
- **Ignored**: `caps_lock_on/off`, `num_lock_on/off`, `scroll_lock_on/off`, `separator`, `font`, `font_size`

### Keyboard Layout Widget

**Display Modes**: Text only (shows current keyboard layout)

```json
{
  "type": "keyboard_layout",
  "properties": {
    "display_format": "iso639-1",
    "update_interval": 0.2,
    "font": null,
    "font_size": 10,
    "horizontal_align": "center",
    "vertical_align": "center",
    "padding": 2
  }
}
```

| Property          | Type   | Values                          | Default        | Description                                      |
|-------------------|--------|---------------------------------|----------------|--------------------------------------------------|
| `update_interval` | number | ≥0.1                            | 0.2            | Check interval (seconds)                         |
| `display_format`  | string | iso639-1, iso639-2, full        | "iso639-1"     | Language code display format                     |
| `font`            | string | -                               | null           | Font name (null = default)                       |
| `font_size`       | int    | ≥6                              | 10             | Font size in points                              |

**Note**: Uses efficient polling with caching - only updates display when layout actually changes. Low CPU usage even at 0.1-0.2s intervals.

**Display Format Options**:
- `iso639-1`: 2-letter codes (EN, RU, DE, FR, ES, IT, etc.)
- `iso639-2`: 3-letter codes (ENG, RUS, DEU, FRA, SPA, ITA, etc.)
- `full`: Full language names (English, Русский, Deutsch, Français, etc.)

**Supported Languages**:
- English (US, UK, AU, CA)
- Russian (Русский)
- German (Deutsch)
- French (Français)
- Spanish (Español)
- Italian (Italiano)
- Polish (Polski)
- Portuguese (Português)
- Dutch (Nederlands)
- Norwegian (Norsk)
- Swedish (Svenska)
- Finnish (Suomi)
- Danish (Dansk)
- Czech (Čeština)
- Hungarian (Magyar)
- Romanian (Română)
- Slovenian (Slovenščina)
- Slovak (Slovenčina)
- Greek (Ελληνικά)
- Turkish (Türkçe)
- Japanese (日本語)
- Korean (한국어)
- Chinese (中文)
- Hebrew (עברית)
- Arabic (العربية)
- Thai (ไทย)
- Vietnamese (Tiếng Việt)
- And more...

**Note**: For unknown layouts, displays LCID in hex format (e.g., "0x0419"). Windows only - shows "N/A" on Linux.

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

| Property              | Type    | Range                                               | Default          | Description                    |
|-----------------------|---------|-----------------------------------------------------|------------------|--------------------------------|
| `display_mode`        | string  | text, bar_horizontal, bar_vertical, gauge, triangle | "bar_horizontal" | Display mode                   |
| `update_interval`     | number  | ≥0.1                                                | 0.1              | Update interval (seconds)      |
| `fill_color`          | integer | 0-255                                               | 255              | Bar/triangle fill color        |
| `bar_border`          | boolean | -                                                   | false            | Draw border around bar         |
| `gauge_color`         | integer | 0-255                                               | 200              | Gauge arc and tick marks color |
| `gauge_needle_color`  | integer | 0-255                                               | 255              | Gauge needle color             |
| `triangle_fill_color` | integer | 0-255                                               | 255              | Triangle fill color            |
| `triangle_border`     | boolean | -                                                   | false            | Draw border around triangle    |

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

**Display Modes**: text, bar_horizontal, bar_vertical, gauge (all modes support stereo with `stereo_mode: true`)

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
    "bar_border": false,
    "gauge_color": 200,
    "gauge_needle_color": 255,
    "use_db_scale": false,
    "show_clipping": true,
    "clipping_threshold": 0.99,
    "silence_threshold": 0.01,
    "decay_rate": 2.0,
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

| Property                 | Type    | Range                                     | Default          | Description                                          |
|--------------------------|---------|-------------------------------------------|------------------|------------------------------------------------------|
| `display_mode`           | string  | text, bar_horizontal, bar_vertical, gauge | "bar_horizontal" | Display mode                                         |
| `update_interval`        | number  | ≥0.03                                     | 0.1              | Meter update interval (seconds)                      |
| `fill_color`             | integer | 0-255                                     | 255              | Main meter fill color                                |
| `clipping_color`         | integer | 0-255                                     | 200              | Color when clipping detected                         |
| `left_channel_color`     | integer | 0-255                                     | 255              | Left channel color (when stereo_mode enabled)        |
| `right_channel_color`    | integer | 0-255                                     | 200              | Right channel color (when stereo_mode enabled)       |
| `stereo_mode`            | boolean | -                                         | false            | Display left and right channels separately           |
| `bar_border`             | boolean | -                                         | false            | Draw border around bar                               |
| `gauge_color`            | integer | 0-255                                     | 200              | Gauge arc and tick marks color                       |
| `gauge_needle_color`     | integer | 0-255                                     | 255              | Gauge needle color                                   |
| `use_db_scale`           | boolean | -                                         | false            | Use logarithmic dB scale (-60dB to 0dB)              |
| `show_clipping`          | boolean | -                                         | true             | Show clipping indicator (all modes)                  |
| `clipping_threshold`     | number  | 0.0-1.0                                   | 0.99             | Peak level that triggers clipping (0.0=0%, 1.0=100%) |
| `silence_threshold`      | number  | 0.0-1.0                                   | 0.01             | Peak level below which is considered silence         |
| `decay_rate`             | number  | ≥0.1                                      | 2.0              | Peak decay rate (units/second, VU meter ballistics)  |
| `show_peak_hold`         | boolean | -                                         | true             | Show peak hold line (held maximum peak)              |
| `peak_hold_time`         | number  | ≥0.1                                      | 1.0              | How long to hold peak indicator (seconds)            |
| `auto_hide_on_silence`   | boolean | -                                         | false            | Auto-hide when no audio detected                     |
| `auto_hide_silence_time` | number  | ≥0.5                                      | 2.0              | Time after last audio before hiding (seconds)        |

**Note**: The volume meter widget also supports `auto_hide` and `auto_hide_timeout` properties (see [Auto-Hide Properties](#auto-hide-properties)). When `auto_hide_on_silence` is enabled, the widget triggers visibility when audio is detected above the `silence_threshold`.

**Display Mode Details**:

**text**: Shows peak level as percentage or dB. When `use_db_scale` is true, displays dB value (e.g., "-12.3 dB"). Shows "CLIP" when clipping is detected.
- **Stereo mode**: Displays both channels (e.g., "L:45% R:52%")

**bar_horizontal**: Horizontal bar showing current audio peak level with smooth decay.
- **Stereo mode**: Two horizontal bars stacked (top = left channel, bottom = right channel)
- **show_peak_hold**: Line showing held maximum peak per channel

**bar_vertical**: Vertical bar showing current audio peak level with smooth decay.
- **Stereo mode**: Two vertical bars side by side (left = left channel, right = right channel)
- **show_peak_hold**: Line showing held maximum peak per channel

**gauge**: Semicircular gauge with needle pointing to current peak level.
- **Stereo mode**: Two gauges side by side (left = left channel, right = right channel)
- **show_clipping**: Changes needle color to red when clipping detected

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

### Audio Visualizer Widget

**Platform**: Windows only (uses Windows Core Audio API)

**Display Modes**: spectrum (frequency bars), oscilloscope (waveform)

```json
{
  "type": "audio_visualizer",
  "properties": {
    "display_mode": "spectrum",
    "update_interval": 0.033,
    "bar_count": 32,
    "frequency_scale": "logarithmic",
    "bar_style": "bars",
    "smoothing": 0.7,
    "peak_hold": true,
    "peak_hold_time": 1.0,
    "waveform_style": "line",
    "channel_mode": "stereo_combined",
    "sample_count": 128,
    "fill_color": 255,
    "left_channel_color": 255,
    "right_channel_color": 200,
    "auto_hide": false,
    "auto_hide_timeout": 2.0
  }
}
```

**Common Properties**:

| Property | Type | Range | Default | Description |
|----------|------|-------|---------|-------------|
| `display_mode` | string | spectrum, oscilloscope | "spectrum" | Visualization mode |
| `update_interval` | number | ≥0.016 | 0.033 | Update interval in seconds (~30 FPS) |
| `fill_color` | integer | 0-255 | 255 | Main visualization color |

**Spectrum Analyzer Properties** (when `display_mode: "spectrum"`):

| Property | Type | Range | Default | Description |
|----------|------|-------|---------|-------------|
| `bar_count` | integer | 8-128 | 32 | Number of frequency bars |
| `frequency_scale` | string | logarithmic, linear | "logarithmic" | Frequency distribution (logarithmic = Winamp-style) |
| `bar_style` | string | bars, line | "bars" | Bar rendering: filled bars or line graph |
| `smoothing` | number | 0.0-1.0 | 0.7 | Bar fall smoothing (0.0 = instant, 1.0 = very smooth) |
| `peak_hold` | boolean | - | true | Show peak indicators on top of bars |
| `peak_hold_time` | number | ≥0.1 | 1.0 | Peak hold duration in seconds |

**Oscilloscope Properties** (when `display_mode: "oscilloscope"`):

| Property | Type | Range | Default | Description |
|----------|------|-------|---------|-------------|
| `waveform_style` | string | line, filled | "line" | Waveform rendering style |
| `channel_mode` | string | mono, stereo_combined, stereo_separated | "stereo_combined" | Audio channel display mode |
| `sample_count` | integer | 32-512 | 128 | Number of audio samples to display |
| `left_channel_color` | integer | 0-255 | 255 | Left channel color (stereo modes) |
| `right_channel_color` | integer | 0-255 | 200 | Right channel color (stereo modes) |

**Display Mode Details**:

**spectrum**: Classic spectrum analyzer with frequency bars (like Winamp, Windows Media Player).
- Captures system audio via WASAPI loopback
- Performs FFT (Fast Fourier Transform) to analyze frequencies
- Displays frequency magnitude as vertical bars
- **logarithmic scale**: More bars for bass/mid frequencies, fewer for high (natural hearing perception)
- **linear scale**: Equal frequency distribution across all bars
- **peak_hold**: Shows peak dots that slowly decay after peaks
- **smoothing**: Controls how fast bars fall after audio quiets

**oscilloscope**: Real-time audio waveform display.
- Shows raw audio signal as time-domain waveform
- **line style**: Single line connecting sample points
- **filled style**: Filled area from center to waveform
- **mono**: Single waveform (left channel only)
- **stereo_combined**: Left and right channels overlaid on same waveform
- **stereo_separated**: Left channel top half, right channel bottom half

**Performance Considerations**:

- **Update rate**: 0.033 (~30 FPS) recommended. Faster rates (0.02 = 50 FPS) use more CPU.
- **Bar count**: More bars (64+) require more FFT processing. 16-32 bars optimal for 128px width.
- **Smoothing**: Higher values (0.8-0.9) create fluid animations but may feel less responsive.

**Audio Source**:

The widget captures audio from the system's default output device in loopback mode. This means:
- It visualizes whatever is currently playing on your system
- Works with any audio source (music players, games, browsers, etc.)
- Requires Windows Core Audio API (Vista and later)
- No special audio device configuration needed

**Platform Support**:

This widget only works on Windows due to Windows Core Audio API dependency. On Linux/Unix systems, the widget will display an error message.

**Example Configurations**:

See `configs/examples/audio_visualizer.json` for spectrum analyzer example and `configs/examples/audio_visualizer_oscilloscope.json` for oscilloscope example.

## Examples

### Example 1: Simple Clock

```json
{
  "$schema": "./config.schema.json",
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
  "display": {"width": 128, "height": 40, "background_color": 0},
  "layout": {"type": "basic"},
  "widgets": [
    {
      "type": "network",
      "id": "background",
      "position": {"x": 0, "y": 0, "w": 128, "h": 40, "z_order": 0},
      "style": {"background_color": 0},
      "properties": {"display_mode": "graph"}
    },
    {
      "type": "clock",
      "id": "overlay",
      "position": {"x": 0, "y": 0, "w": 128, "h": 40, "z_order": 10},
      "style": {"background_color": -1},
      "properties": {"format": "%H:%M", "font_size": 16}
    }
  ]
}
```

### Example 6: Per-Core CPU Display

```json
{
  "$schema": "./config.schema.json",
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
        "core_margin": 1,
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

3. **Transparency**: Use `background_color: -1` for transparent overlays
   - Opaque background (0-255): Normal widgets
   - Transparent (-1): Overlay text/graphics over other widgets
   - Background pixels (black/0) are skipped during compositing

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
