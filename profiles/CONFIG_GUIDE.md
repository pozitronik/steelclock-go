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
  "$schema": "schema/config.schema.json",
  "schema_version": 2,
  ...
}
```

Supported IDEs: VS Code, JetBrains IDEs, Visual Studio, Sublime Text, and others.

## Configuration Structure

### Top-Level Structure

```json
{
  "$schema": "schema/config.schema.json",
  "schema_version": 2,
  "game_name": "STEELCLOCK",
  "game_display_name": "SteelClock",
  "refresh_rate_ms": 100,
  "backend": "gamesense",
  "direct_driver": {
    ...
  },
  "display": {
    ...
  },
  "defaults": {
    ...
  },
  "widgets": [
    ...
  ]
}
```

### Global Settings

| Property                | Type    | Default      | Description                                      |
|-------------------------|---------|--------------|--------------------------------------------------|
| `schema_version`        | integer | 2            | Schema version (must be 2)                       |
| `game_name`             | string  | "STEELCLOCK" | Internal game name for GameSense                 |
| `game_display_name`     | string  | "SteelClock" | Display name in SteelSeries GG                   |
| `refresh_rate_ms`       | integer | 100          | Display refresh rate (see notes)                 |
| `backend`               | string  | (auto)       | Backend: "gamesense", "direct", or omit for auto |
| `unregister_on_exit`    | boolean | false        | Unregister on exit (may timeout)                 |
| `deinitialize_timer_ms` | integer | 15000        | Game deactivation timeout (1000-60000ms)         |

### Backend Configuration

| Backend     | Description                               | Min Refresh  | Max Refresh |
|-------------|-------------------------------------------|--------------|-------------|
| `gamesense` | SteelSeries GG API                        | 100ms (10Hz) | 100ms       |
| `direct`    | USB HID (Windows only)                    | ~16ms (60Hz) | 30ms (33Hz) |
| (omitted)   | Auto-select: tries gamesense, then direct | -            | -           |

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

| Property     | Type    | Range | Default | Description                |
|--------------|---------|-------|---------|----------------------------|
| `width`      | integer | -     | 128     | Display width in pixels    |
| `height`     | integer | -     | 40      | Display height in pixels   |
| `background` | integer | 0-255 | 0       | Background color (0=black) |

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

| Type               | Description             | Modes                         |
|--------------------|-------------------------|-------------------------------|
| `clipboard`        | Clipboard content       | text                          |
| `clock`            | Time display            | text, analog, binary, segment |
| `cpu`              | CPU usage monitor       | text, bar, graph, gauge       |
| `memory`           | RAM usage monitor       | text, bar, graph, gauge       |
| `network`          | Network I/O monitor     | text, bar, graph, gauge       |
| `disk`             | Disk I/O monitor        | text, bar, graph              |
| `volume`           | System volume           | text, bar, gauge, triangle    |
| `volume_meter`     | Audio peak meter        | text, bar, gauge              |
| `audio_visualizer` | Spectrum/oscilloscope   | spectrum, oscilloscope        |
| `keyboard`         | Lock key indicators     | -                             |
| `keyboard_layout`  | Current keyboard layout | -                             |
| `doom`             | DOOM game               | -                             |
| `winamp`           | Winamp media player     | -                             |
| `matrix`           | Matrix digital rain     | -                             |
| `weather`          | Current weather         | icon, text                    |
| `game_of_life`     | Conway's Game of Life   | -                             |
| `hacker_code`      | Procedural code typing  | c, asm, mixed                 |
| `hyperspace`       | Star Wars lightspeed    | continuous, cycle             |
| `screen_mirror`    | Screen capture display  | -                             |

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

| Property          | Type    | Required | Description                                                                         |
|-------------------|---------|----------|-------------------------------------------------------------------------------------|
| `type`            | string  | Yes      | Widget type                                                                         |
| `enabled`         | boolean | No       | Enable widget (default: true)                                                       |
| `mode`            | string  | Depends  | Display mode (widget-specific)                                                      |
| `update_interval` | number  | No       | Update interval in seconds (default: 1.0)                                           |
| `poll_interval`   | number  | No       | Internal polling interval for volume/volume_meter widgets in seconds (default: 0.1) |

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

| Property | Type    | Description               |
|----------|---------|---------------------------|
| `x`      | integer | X coordinate (pixels)     |
| `y`      | integer | Y coordinate (pixels)     |
| `w`      | integer | Width (pixels)            |
| `h`      | integer | Height (pixels)           |
| `z`      | integer | Z-order (higher = on top) |

### Style Object

```json
"style": {
  "background": 0,
  "border": -1,
  "padding": 2
}
```

| Property     | Type    | Range     | Default | Description                         |
|--------------|---------|-----------|---------|-------------------------------------|
| `background` | integer | -1 to 255 | 0       | -1=transparent, 0-255=grayscale     |
| `border`     | integer | -1 to 255 | -1      | -1=disabled, 0-255=border color     |
| `padding`    | integer | 0+        | 0       | Padding from widget edges in pixels |

### Text Object

```json
"text": {
  "format": "%H:%M:%S",
  "font": "Consolas",
  "size": 10,
  "align": {"h": "center", "v": "center"}
}
```

| Property  | Type    | Description                     |
|-----------|---------|---------------------------------|
| `format`  | string  | Format string (widget-specific) |
| `font`    | string  | Font name or TTF path           |
| `size`    | integer | Font size in pixels             |
| `align.h` | string  | "left", "center", "right"       |
| `align.v` | string  | "top", "center", "bottom"       |

### Auto-Hide Object

```json
"auto_hide": {
  "enabled": true,
  "timeout": 2.0,
  "on_silence": false
}
```

| Property     | Type    | Default | Description           |
|--------------|---------|---------|-----------------------|
| `enabled`    | boolean | false   | Enable auto-hide      |
| `timeout`    | number  | 2.0     | Seconds before hiding |
| `on_silence` | boolean | false   | Hide on audio silence |

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

### Clipboard Widget

Displays clipboard content or content type description. Supports auto-show mode that shows the widget briefly when clipboard changes - useful as a "copy notification".

```json
{
  "type": "clipboard",
  "position": {"x": 0, "y": 0, "w": 128, "h": 20},
  "auto_hide": {
    "enabled": true,
    "timeout": 3.0
  },
  "text": {
    "format": "{content}",
    "font": "5x7",
    "align": {"h": "left", "v": "center"}
  },
  "scroll": {
    "enabled": true,
    "speed": 30
  }
}
```

#### Content Types

The widget automatically detects clipboard content type:

| Content Type | Display Behavior                      |
|--------------|---------------------------------------|
| Plain text   | Shows text content (scrolled if long) |
| Image        | Shows `[Image]` (metadata only)       |
| Files        | Shows `filename.ext (+N more)`        |
| HTML         | Shows `[HTML]`                        |
| Empty        | Shows `[Empty]`                       |
| Unknown      | Shows `[Unknown]`                     |

#### Format Tokens

Use these tokens in `text.format`:

| Token       | Description                           | Example                    |
|-------------|---------------------------------------|----------------------------|
| `{content}` | Clipboard content or type description | "Hello World" or "[Image]" |
| `{type}`    | Content type label                    | "Text", "Image", "Files"   |
| `{length}`  | Content length (chars for text)       | "42"                       |
| `{preview}` | First 20 characters of text           | "Hello Wor..."             |

**Examples:**
- `"{content}"` - Just the content (default)
- `"{type}: {content}"` - "Text: Hello World"
- `"{type} ({length})"` - "Text (42)"

#### Auto-Show Mode

Use `auto_hide` to create a notification-style widget that appears when clipboard changes:

```json
{
  "auto_hide": {
    "enabled": true,
    "timeout": 3.0
  }
}
```

The widget:
1. Starts hidden
2. Shows when clipboard changes (triggers `auto_hide`)
3. Hides after `timeout` seconds
4. Shows again on next clipboard change

#### Clipboard Configuration

| Property                     | Type | Default | Description                          |
|------------------------------|------|---------|--------------------------------------|
| `clipboard.max_length`       | int  | 100     | Max characters to display            |
| `clipboard.show_type`        | bool | true    | Show content type prefix             |
| `clipboard.scroll_long_text` | bool | true    | Enable horizontal scroll             |
| `clipboard.poll_interval_ms` | int  | 500     | Clipboard check interval             |
| `clipboard.show_invisible`   | bool | false   | Show invisible characters as symbols |

#### Invisible Characters

When `show_invisible` is enabled, invisible characters are displayed as escape sequences:

| Character | Display | Description         |
|-----------|---------|---------------------|
| `\r\n`    | `\n`    | Windows line ending |
| `\n`      | `\n`    | Unix line ending    |
| `\r`      | `\r`    | Old Mac line ending |
| `\t`      | `\t`    | Tab character       |

#### Platform Support

| Platform | Implementation          | Change Detection            |
|----------|-------------------------|-----------------------------|
| Windows  | Win32 API               | Sequence number (efficient) |
| Linux    | wl-paste / xclip / xsel | Content hash comparison     |
| Other    | Not supported           | -                           |

**Linux requirements:** Install `wl-paste` (Wayland) or `xclip`/`xsel` (X11).

#### Examples

**Simple notification on copy:**
```json
{
  "type": "clipboard",
  "position": {"x": 0, "y": 0, "w": 128, "h": 20},
  "auto_hide": {
    "enabled": true,
    "timeout": 3.0
  },
  "text": {
    "format": "{content}"
  }
}
```

**Show content type with text:**
```json
{
  "type": "clipboard",
  "position": {"x": 0, "y": 0, "w": 128, "h": 40},
  "text": {
    "format": "{type}: {content}"
  },
  "scroll": {
    "enabled": true,
    "speed": 30
  }
}
```

**Always visible clipboard monitor:**
```json
{
  "type": "clipboard",
  "position": {"x": 0, "y": 0, "w": 128, "h": 40},
  "style": {"border": 255},
  "text": {
    "format": "{preview}",
    "font": "5x7"
  }
}
```

---

### Hacker Code Widget

Displays procedurally generated code being "typed" in real-time, creating an authentic hacking/coding visual effect. The code looks realistic but is generated on-the-fly using templates for C-like code or x86 assembly.

#### Code Styles

| Style   | Description                                                   |
|---------|---------------------------------------------------------------|
| `c`     | C-like code with functions, variables, pointers, control flow |
| `asm`   | x86-style assembly with MOV, XOR, JMP, CALL, etc.             |
| `mixed` | Alternates between C and assembly blocks                      |

#### Basic Usage

**C-style hacker effect:**
```json
{
  "type": "hacker_code",
  "position": {"x": 0, "y": 0, "w": 128, "h": 40},
  "hacker_code": {
    "style": "c",
    "typing_speed": 60,
    "show_cursor": true
  }
}
```

**Assembly code:**
```json
{
  "type": "hacker_code",
  "position": {"x": 0, "y": 0, "w": 128, "h": 40},
  "hacker_code": {
    "style": "asm",
    "typing_speed": 100,
    "line_delay": 100
  }
}
```

**Mixed mode:**
```json
{
  "type": "hacker_code",
  "position": {"x": 0, "y": 0, "w": 128, "h": 40},
  "hacker_code": {
    "style": "mixed",
    "typing_speed": 50
  }
}
```

#### Animation Behavior

The widget simulates a typewriter effect:
1. Characters appear one at a time at the cursor position
2. When a line completes, the cursor moves to the next line after a brief delay
3. When the screen fills, all lines scroll up by one
4. A new line starts at the bottom
5. The process repeats infinitely with procedurally generated code

#### Configuration

| Property                      | Type   | Default | Description                                |
|-------------------------------|--------|---------|--------------------------------------------|
| `hacker_code.style`           | string | "c"     | Code style: c, asm, mixed                  |
| `hacker_code.typing_speed`    | int    | 50      | Characters per second (char-by-char)       |
| `hacker_code.line_delay`      | int    | 200     | Milliseconds pause at end of line          |
| `hacker_code.show_cursor`     | bool   | true    | Show blinking cursor                       |
| `hacker_code.cursor_blink_ms` | int    | 500     | Cursor blink interval (ms)                 |
| `hacker_code.indent_size`     | int    | 2       | Spaces per indent level                    |
| `text.font`                   | string | (auto)  | Font: "3x5", "5x7", "pixel3x5", "pixel5x7" |

**Font Selection:** Use standard `text.font` setting. If not specified, auto-selects based on display height (3x5 for small displays, 5x7 for larger).

**Scroll Behavior:** Lines scroll up only when the typing cursor needs space beyond the visible area (when the first character is typed on a new line that would be off-screen).

#### Examples

**Fast assembly scrolling (no cursor):**
```json
{
  "type": "hacker_code",
  "position": {"x": 0, "y": 0, "w": 128, "h": 40},
  "hacker_code": {
    "style": "asm",
    "typing_speed": 150,
    "line_delay": 50,
    "show_cursor": false
  }
}
```

**Slow typing with visible cursor:**
```json
{
  "type": "hacker_code",
  "position": {"x": 0, "y": 0, "w": 128, "h": 40},
  "hacker_code": {
    "style": "c",
    "typing_speed": 30,
    "cursor_blink_ms": 300
  }
}
```

---

### Screen Mirror Widget

Captures and displays screen content on the OLED. Supports full screen capture, region capture, and window capture. The captured image is scaled to fit the widget and converted to grayscale with optional dithering.

**Platform Support:**
- **Windows**: Uses GDI BitBlt (works in VMs, RDP, etc.)
- **Linux**: Uses ffmpeg x11grab (requires `ffmpeg`, window capture requires `xdotool`)

#### Basic Usage

**Full screen capture:**
```json
{
  "type": "screen_mirror",
  "position": {"x": 0, "y": 0, "w": 128, "h": 40},
  "screen_mirror": {
    "scale_mode": "fit",
    "fps": 15,
    "dither_mode": "floyd_steinberg"
  }
}
```

#### Capture Modes

**Region capture** - capture a specific area of the screen:
```json
{
  "type": "screen_mirror",
  "position": {"x": 0, "y": 0, "w": 128, "h": 40},
  "screen_mirror": {
    "region": {
      "x": 100,
      "y": 100,
      "w": 400,
      "h": 300
    },
    "scale_mode": "crop"
  }
}
```

**Window capture by title** - capture a specific window:
```json
{
  "type": "screen_mirror",
  "position": {"x": 0, "y": 0, "w": 128, "h": 40},
  "screen_mirror": {
    "window": {
      "title": "Calculator"
    },
    "scale_mode": "fit"
  }
}
```

**Active window capture** - always capture the focused window:
```json
{
  "type": "screen_mirror",
  "position": {"x": 0, "y": 0, "w": 128, "h": 40},
  "screen_mirror": {
    "window": {
      "active": true
    },
    "scale_mode": "stretch"
  }
}
```

#### Scale Modes

| Mode      | Behavior                                          |
|-----------|---------------------------------------------------|
| `fit`     | Preserve aspect ratio, add black bars (letterbox) |
| `stretch` | Fill entire widget, may distort aspect ratio      |
| `crop`    | Preserve aspect ratio, crop edges to fill         |

#### Dither Modes

| Mode              | Description                                        |
|-------------------|----------------------------------------------------|
| `floyd_steinberg` | Error diffusion dithering (best quality, default)  |
| `ordered`         | Bayer matrix ordered dithering (faster, patterned) |
| `none`            | No dithering (simple threshold)                    |

#### Configuration

| Property                    | Type            | Default           | Description                           |
|-----------------------------|-----------------|-------------------|---------------------------------------|
| `screen_mirror.display`     | int/string/null | null              | Display selector (see below)          |
| `screen_mirror.region`      | object          | null              | Capture region {x, y, w, h}           |
| `screen_mirror.window`      | object          | null              | Window capture {title, class, active} |
| `screen_mirror.scale_mode`  | string          | "fit"             | Scale mode: fit, stretch, crop        |
| `screen_mirror.fps`         | int             | 15                | Capture framerate (1-30)              |
| `screen_mirror.dither_mode` | string          | "floyd_steinberg" | Dithering algorithm                   |

#### Display Selection

The `display` parameter accepts multiple types:

| Value            | Type   | Description                                       |
|------------------|--------|---------------------------------------------------|
| `null`           | null   | Primary monitor (default)                         |
| `0`, `1`, `2`... | int    | Specific monitor by index                         |
| `-1`             | int    | All monitors combined (virtual screen)            |
| `"HDMI-1"`       | string | Match monitor by name (partial, case-insensitive) |

**Examples:**
```json
"display": null      // Primary monitor
"display": 0         // First monitor (usually primary)
"display": 1         // Second monitor
"display": -1        // All monitors combined
"display": "HDMI"    // Monitor with "HDMI" in name
"display": "DP-1"    // Monitor named "DP-1"
```

#### Window Configuration

| Property        | Type   | Description                          |
|-----------------|--------|--------------------------------------|
| `window.title`  | string | Substring to match in window title   |
| `window.class`  | string | Window class name (Windows-specific) |
| `window.active` | bool   | Capture currently focused window     |

#### Examples

**Mini screen preview in corner:**
```json
{
  "type": "screen_mirror",
  "position": {"x": 88, "y": 0, "w": 40, "h": 40},
  "screen_mirror": {
    "fps": 10,
    "scale_mode": "fit"
  }
}
```

**High FPS for active window:**
```json
{
  "type": "screen_mirror",
  "position": {"x": 0, "y": 0, "w": 128, "h": 40},
  "screen_mirror": {
    "window": {"active": true},
    "fps": 30,
    "dither_mode": "ordered"
  }
}
```

**Second monitor:**
```json
{
  "type": "screen_mirror",
  "position": {"x": 0, "y": 0, "w": 128, "h": 40},
  "screen_mirror": {
    "display": 1,
    "scale_mode": "fit"
  }
}
```

**All monitors combined:**
```json
{
  "type": "screen_mirror",
  "position": {"x": 0, "y": 0, "w": 128, "h": 40},
  "screen_mirror": {
    "display": -1,
    "scale_mode": "fit"
  }
}
```

**Monitor by name:**
```json
{
  "type": "screen_mirror",
  "position": {"x": 0, "y": 0, "w": 128, "h": 40},
  "screen_mirror": {
    "display": "HDMI",
    "scale_mode": "fit"
  }
}
```

---

### Clock Widget

**Modes:** `text`, `analog`, `binary`, `segment`

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

#### Binary Mode

Displays time as a binary clock using LED-style dots.

**BCD Style (default):** Each decimal digit is represented in 4-bit BCD (Binary Coded Decimal).

```
Vertical layout:           Horizontal layout:
     H  H  M  M  S  S      H  8 4 2 1  8 4 2 1
8    .  .  .  .  .  .      M  8 4 2 1  8 4 2 1
4    .  *  .  *  .  *      S  8 4 2 1  8 4 2 1
2    *  .  *  .  *  .
1    .  *  .  *  .  *
```

**True Binary Style:** Hours, minutes, and seconds as raw binary numbers (5-6 bits each).

```json
{
  "type": "clock",
  "position": {"x": 0, "y": 0, "w": 128, "h": 40},
  "mode": "binary",
  "binary": {
    "format": "%H:%M:%S",
    "style": "bcd",
    "layout": "vertical",
    "dot_size": 5,
    "dot_spacing": 2,
    "dot_style": "circle",
    "on_color": 255,
    "off_color": 40,
    "show_labels": true,
    "show_hint": true
  }
}
```

| Property      | Options                  | Default    | Description                           |
|---------------|--------------------------|------------|---------------------------------------|
| `format`      | strftime                 | `%H:%M:%S` | Which components to show (%H, %M, %S) |
| `style`       | `bcd`, `true`            | `bcd`      | Binary representation style           |
| `layout`      | `vertical`, `horizontal` | `vertical` | Bit layout orientation                |
| `dot_size`    | 1+                       | 4          | Dot diameter in pixels                |
| `dot_spacing` | 0+                       | 2          | Gap between dots in pixels            |
| `dot_style`   | `circle`, `square`       | `circle`   | Dot shape                             |
| `on_color`    | 0-255                    | 255        | Color for "on" bits (1)               |
| `off_color`   | 0-255                    | 40         | Color for "off" bits (0 = invisible)  |
| `show_labels` | true/false               | false      | Show H/M/S labels                     |
| `show_hint`   | true/false               | false      | Show decimal values alongside binary  |

#### Segment Mode

Displays time using a seven-segment display style, like digital alarm clocks.

```
 ___     ___
|   |   |   |
|___|   |___|
|   | . |   |
|___|   |___|
```

```json
{
  "type": "clock",
  "position": {"x": 0, "y": 0, "w": 128, "h": 40},
  "mode": "segment",
  "segment": {
    "format": "%H:%M:%S",
    "digit_height": 0,
    "segment_thickness": 3,
    "segment_style": "hexagon",
    "digit_spacing": 2,
    "colon_style": "dots",
    "colon_blink": true,
    "on_color": 255,
    "off_color": 30,
    "flip": {
      "style": "fade",
      "speed": 0.15
    }
  }
}
```

| Property            | Options                           | Default     | Description                          |
|---------------------|-----------------------------------|-------------|--------------------------------------|
| `format`            | see below                         | `%H:%M:%S`  | Time format with optional literals   |
| `digit_height`      | 0+                                | 0           | Digit height (0 = auto-fit)          |
| `segment_thickness` | 1+                                | 2           | Segment line thickness               |
| `segment_style`     | `rectangle`, `hexagon`, `rounded` | `rectangle` | Segment shape style                  |
| `digit_spacing`     | 0+                                | 2           | Space between digits                 |
| `colon_style`       | `dots`, `bar`, `none`             | `dots`      | Colon separator style                |
| `colon_blink`       | true/false                        | true        | Blink colons each second             |
| `on_color`          | 0-255                             | 255         | Active segment color                 |
| `off_color`         | 0-255                             | 30          | Inactive segment color (0=invisible) |

**Segment Styles:**
- `rectangle` - Simple rectangular bars (default)
- `hexagon` - Classic LCD style with angled/pointed ends
- `rounded` - Segments with rounded/semicircular ends

**Format String:**
Supports time specifiers and literal digits:
- `%H` - Hours (00-23)
- `%M` - Minutes (00-59)
- `%S` - Seconds (00-59)
- `0-9` - Literal digits (for testing)
- `:` - Colon separator

Examples:
- `"%H:%M:%S"` - Full time display (default)
- `"%H:%M"` - Hours and minutes only
- `"88:88:88"` - All 8s (tests all segments lit)
- `"12:34:56"` - Static digits for testing
- `"%H:00"` - Current hour with static `:00`

**Flip Animation:**

| Property | Options        | Default | Description                     |
|----------|----------------|---------|---------------------------------|
| `style`  | `none`, `fade` | `none`  | Animation style (none=disabled) |
| `speed`  | 0.05-1.0       | 0.15    | Animation duration in seconds   |

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

| Property              | Description              |
|-----------------------|--------------------------|
| `per_core.enabled`    | Show per-core usage      |
| `per_core.margin`     | Margin between core bars |
| `bar.colors.fill`     | Bar fill color           |
| `graph.colors.fill`   | Graph fill color         |
| `gauge.colors.arc`    | Gauge arc color          |
| `gauge.colors.needle` | Gauge needle color       |

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

| Property                 | Description                     |
|--------------------------|---------------------------------|
| `interface`              | Network interface (null=auto)   |
| `max_speed_mbps`         | Max speed for scaling (-1=auto) |
| `gauge.colors.rx`        | RX (download) arc color         |
| `gauge.colors.tx`        | TX (upload) arc color           |
| `gauge.colors.rx_needle` | RX needle color                 |
| `gauge.colors.tx_needle` | TX needle color                 |
| `graph.colors.rx`        | RX graph fill color             |
| `graph.colors.tx`        | TX graph fill color             |

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
      "clipping": 200,
      "peak": 180
    }
  },
  "gauge": {
    "show_ticks": true,
    "colors": {
      "arc": 200,
      "needle": 255,
      "ticks": 150,
      "clipping": 200,
      "peak": 180
    }
  },
  "text": {
    "format": "%d%%",
    "font": null,
    "size": 10,
    "align": {"h": "center", "v": "center"}
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

| Object         | Properties                                                       |
|----------------|------------------------------------------------------------------|
| `bar.colors`   | `fill`, `clipping`, `peak` (bar mode only)                       |
| `gauge.colors` | `arc`, `needle`, `ticks`, `clipping`, `peak` (gauge mode only)   |
| `text`         | `format`, `font`, `size`, `align` (no colors - uses font glyphs) |
| `stereo`       | `enabled`, `divider` (divider applies to all modes)              |
| `metering`     | `db_scale`, `decay_rate`, `silence_threshold`                    |
| `peak`         | `enabled`, `hold_time` (color configured in mode colors)         |
| `clipping`     | `enabled`, `threshold` (color configured in mode colors)         |

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

| Property                  | Options             | Description                   |
|---------------------------|---------------------|-------------------------------|
| `spectrum.bars`           | 8-128               | Number of frequency bars      |
| `spectrum.scale`          | logarithmic, linear | Frequency distribution        |
| `spectrum.style`          | bars, line          | Rendering style               |
| `spectrum.smoothing`      | 0.0-1.0             | Fall-off smoothing            |
| `spectrum.peak.enabled`   | true/false          | Show peak hold indicators     |
| `spectrum.peak.hold_time` | 0.1+                | Peak hold duration in seconds |

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

| Property               | Options                                 | Description    |
|------------------------|-----------------------------------------|----------------|
| `oscilloscope.style`   | line, filled                            | Waveform style |
| `oscilloscope.samples` | 32-512                                  | Sample count   |
| `channel`              | mono, stereo_combined, stereo_separated | Channel mode   |

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

| Format     | Example          |
|------------|------------------|
| `iso639-1` | EN, RU, DE       |
| `iso639-2` | ENG, RUS, DEU    |
| `full`     | English, Русский |

### DOOM Widget

Plays DOOM shareware demo on the OLED display. Auto-downloads doom1.wad if not found.

```json
{
  "type": "doom",
  "position": {"x": 0, "y": 0, "w": 128, "h": 40},
  "wad": "doom1.wad",
  "doom": {
    "render_mode": "posterize",
    "posterize_levels": 4
  }
}
```

#### Render Modes

The `doom.render_mode` setting controls how color frames are converted to grayscale for the OLED display:

| Mode        | Description                                                           |
|-------------|-----------------------------------------------------------------------|
| `normal`    | Standard luminance conversion (default)                               |
| `contrast`  | Auto-contrast stretching - maps actual min/max to full 0-255 range    |
| `posterize` | Reduces to N discrete gray levels - reduces noise while keeping depth |
| `threshold` | Pure black/white conversion - maximum clarity but loses depth         |
| `dither`    | Ordered dithering using Bayer matrix - retro dot-pattern look         |
| `gamma`     | Gamma correction with contrast boost - brightens dark scenes          |

#### Render Mode Settings

| Setting            | Mode      | Description                                  | Default |
|--------------------|-----------|----------------------------------------------|---------|
| `posterize_levels` | posterize | Number of gray levels (2-16)                 | 4       |
| `threshold_value`  | threshold | Cutoff brightness (0-255)                    | 128     |
| `gamma`            | gamma     | Gamma value (0.1-3.0, >1 brightens midtones) | 1.5     |
| `contrast_boost`   | gamma     | Contrast multiplier (1.0-3.0)                | 1.2     |
| `dither_size`      | dither    | Bayer matrix size (2, 4, or 8)               | 4       |

**Recommended settings:**
- For best visibility: `"render_mode": "posterize"` with `"posterize_levels": 4`
- For dark scenes: `"render_mode": "gamma"` with `"gamma": 2.0`
- For retro look: `"render_mode": "dither"` with `"dither_size": 4`

### Winamp Widget

Displays information from Winamp media player. Windows only (shows placeholder on other platforms).

```json
{
  "type": "winamp",
  "position": {"x": 0, "y": 0, "w": 128, "h": 40},
  "update_interval": 0.2,
  "text": {
    "format": "{title}",
    "size": 12,
    "align": {"h": "center", "v": "center"}
  },
  "winamp": {
    "placeholder": {
      "mode": "icon",
      "text": "No Winamp"
    }
  },
  "scroll": {
    "enabled": true,
    "direction": "left",
    "speed": 30,
    "mode": "continuous",
    "gap": 20
  },
  "auto_show": {
    "on_track_change": true,
    "on_play": false,
    "on_pause": false,
    "on_stop": false,
    "on_seek": false
  }
}
```

#### Format Placeholders

| Placeholder          | Description                                          |
|----------------------|------------------------------------------------------|
| `{title}`            | Track title from playlist                            |
| `{filename}`         | File name without path                               |
| `{filepath}`         | Full file path                                       |
| `{position}`         | Current position (MM:SS)                             |
| `{duration}`         | Track duration (MM:SS)                               |
| `{position_ms}`      | Current position in milliseconds                     |
| `{duration_s}`       | Track duration in seconds                            |
| `{bitrate}`          | Audio bitrate in kbps                                |
| `{samplerate}`       | Sample rate in Hz                                    |
| `{channels}`         | Number of audio channels                             |
| `{status}`           | Playback status (Playing/Paused/Stopped)             |
| `{track_num}`        | Current track number in playlist                     |
| `{playlist_length}`  | Total tracks in playlist                             |
| `{shuffle}`          | "S" if shuffle enabled, empty otherwise              |
| `{repeat}`           | "R" if repeat enabled, empty otherwise               |
| `{version}`          | Winamp version string                                |

#### Placeholder Configuration

| Property | Description                                              |
|----------|----------------------------------------------------------|
| `mode`   | `icon` (Winamp icon) or `text` (custom text)             |
| `text`   | Text to display when mode is `text` (default: No Winamp) |

#### Scroll Configuration

| Property    | Description                                                               |
|-------------|---------------------------------------------------------------------------|
| `enabled`   | Enable text scrolling                                                     |
| `direction` | Scroll direction: `left`, `right`, `up`, `down`                           |
| `speed`     | Scroll speed in pixels per second (default: 30)                           |
| `mode`      | `continuous` (loop), `bounce` (reverse at edges), `pause_ends` (pause)    |
| `pause_ms`  | Pause duration at ends in ms (for bounce/pause_ends modes, default: 1000) |
| `gap`       | Gap between text repetitions in pixels (for continuous mode, default: 20) |

#### Auto-Show Events

Works with `auto_hide` to show the widget when specific events occur:

| Property          | Description                                | Default |
|-------------------|--------------------------------------------|---------|
| `on_track_change` | Show when track changes                    | true    |
| `on_play`         | Show when playback starts                  | false   |
| `on_pause`        | Show when playback is paused               | false   |
| `on_stop`         | Show when playback stops                   | false   |
| `on_seek`         | Show when user seeks to different position | false   |

### Matrix Widget

Displays the classic "Matrix digital rain" effect with falling characters.

```json
{
  "type": "matrix",
  "position": {"x": 0, "y": 0, "w": 128, "h": 40},
  "update_interval": 0.033,
  "matrix": {
    "charset": "ascii",
    "density": 0.5,
    "min_speed": 0.5,
    "max_speed": 2.5,
    "min_length": 4,
    "max_length": 12,
    "head_color": 255,
    "trail_fade": 0.8,
    "char_change_rate": 0.03
  }
}
```

#### Character Sets

| Charset    | Characters                                      |
|------------|-------------------------------------------------|
| `ascii`    | A-Z, 0-9, symbols (default)                     |
| `katakana` | Japanese Katakana characters                    |
| `binary`   | 0 and 1 only                                    |
| `digits`   | 0-9 only                                        |
| `hex`      | 0-9, A-F                                        |

#### Matrix Configuration

| Property           | Type   | Range   | Default | Description                                |
|--------------------|--------|---------|---------|--------------------------------------------|
| `charset`          | string | -       | "ascii" | Character set to use                       |
| `font_size`        | string | -       | "auto"  | Font: "small" (3x5), "large" (5x7), "auto" |
| `density`          | number | 0.0-1.0 | 0.4     | Column density (probability of active)     |
| `min_speed`        | number | 0.1+    | 0.5     | Minimum fall speed (pixels/frame)          |
| `max_speed`        | number | 0.1+    | 2.0     | Maximum fall speed (pixels/frame)          |
| `min_length`       | int    | 1+      | 4       | Minimum trail length (characters)          |
| `max_length`       | int    | 1+      | 15      | Maximum trail length (characters)          |
| `head_color`       | int    | 0-255   | 255     | Brightness of leading character            |
| `trail_fade`       | number | 0.0-1.0 | 0.85    | Trail fade factor (lower = faster fade)    |
| `char_change_rate` | number | 0.0-1.0 | 0.02    | Character change probability per frame     |

#### Tips

- Use low `update_interval` (0.033 = 30fps) for smooth animation
- Higher `density` = more columns active simultaneously
- Lower `trail_fade` = shorter visible trails
- Use `font_size: "small"` for denser rain effect with more columns
- Use `font_size: "large"` for more readable characters

### Weather Widget

Displays weather information using a flexible format string system with tokens. Supports current weather, forecasts, air quality index (AQI), and UV index.

```json
{
  "type": "weather",
  "position": {"x": 0, "y": 0, "w": 128, "h": 40},
  "update_interval": 300,
  "weather": {
    "provider": "open-meteo",
    "location": {
      "lat": 51.5074,
      "lon": -0.1278
    },
    "units": "metric",
    "format": "{icon} {temp}"
  }
}
```

#### Format String System

The weather widget uses format strings with tokens to define what to display. Tokens are enclosed in curly braces: `{token_name}`.

**Basic tokens (text):**

| Token           | Description             | Example Output  |
|-----------------|-------------------------|-----------------|
| `{temp}`        | Current temperature     | `15C` or `59F`  |
| `{feels}`       | Feels-like temperature  | `13C`           |
| `{humidity}`    | Humidity percentage     | `75%`           |
| `{wind}`        | Wind speed              | `12 km/h`       |
| `{wind_dir}`    | Wind direction          | `NE`            |
| `{pressure}`    | Atmospheric pressure    | `1013 hPa`      |
| `{visibility}`  | Visibility distance     | `10 km`         |
| `{condition}`   | Weather condition       | `Cloudy`        |
| `{description}` | Detailed description    | `Partly cloudy` |
| `{aqi}`         | Air quality index value | `42`            |
| `{aqi_level}`   | AQI level text          | `Good`          |
| `{uv}`          | UV index value          | `6.5`           |
| `{uv_level}`    | UV level text           | `High`          |

**Icon tokens:**

| Token             | Description                                            |
|-------------------|--------------------------------------------------------|
| `{icon}`          | Weather condition icon (sun, cloud, rain, etc.)        |
| `{aqi_icon}`      | AQI level icon (checkmark/warning/X based on level)    |
| `{uv_icon}`       | UV level icon (sun with varying intensity)             |
| `{humidity_icon}` | Humidity level icon (water drop fill level)            |
| `{wind_icon}`     | Wind level icon (wind lines with varying intensity)    |
| `{wind_dir_icon}` | Wind direction arrow icon (N, NE, E, SE, S, SW, W, NW) |

**Large tokens (expand to fill available space):**

| Token               | Description                                    |
|---------------------|------------------------------------------------|
| `{forecast:graph}`  | Temperature trend line graph for next hours    |
| `{forecast:icons}`  | Multi-day forecast with icons and temperatures |
| `{forecast:scroll}` | Scrolling text with current weather + forecast |

#### Multi-line Layouts

Use `\n` in the format string to create multi-line displays:

```json
{
  "weather": {
    "format": "{icon} {temp}\n{humidity} {wind}"
  }
}
```

This displays the icon and temperature on the first line, and humidity and wind on the second line.

#### Format Cycling

Rotate between different formats automatically by passing an array to `format`:

```json
{
  "weather": {
    "format": [
      "{icon} {temp}",
      "{humidity} {wind}",
      "{aqi_level}"
    ],
    "cycle": {
      "interval": 10,
      "transition": "dissolve_fade",
      "speed": 0.5
    }
  }
}
```

This rotates through three different displays every 10 seconds with a crossfade transition.

#### Cycle Configuration

| Property     | Type   | Default  | Description                                    |
|--------------|--------|----------|------------------------------------------------|
| `interval`   | int    | `10`     | Seconds between format changes (0 to disable)  |
| `transition` | string | `"none"` | Transition effect between formats              |
| `speed`      | number | `0.5`    | Transition duration in seconds                 |

**Available transitions:**

| Transition        | Description                                              |
|-------------------|----------------------------------------------------------|
| `none`            | Instant switch (no animation)                            |
| `push_left`       | New content pushes old content out to the left           |
| `push_right`      | New content pushes old content out to the right          |
| `push_up`         | New content pushes old content up                        |
| `push_down`       | New content pushes old content down                      |
| `slide_left`      | New content slides in from right, covering old           |
| `slide_right`     | New content slides in from left, covering old            |
| `slide_up`        | New content slides in from bottom, covering old          |
| `slide_down`      | New content slides in from top, covering old             |
| `dissolve_fade`   | Smooth crossfade between old and new                     |
| `dissolve_pixel`  | Random pixels switch from old to new                     |
| `dissolve_dither` | Ordered dithering pattern reveal                         |
| `box_in`          | Box shrinks from edges, revealing new content            |
| `box_out`         | Box expands from center, revealing new content           |
| `clock_wipe`      | Radial sweep from 12 o'clock clockwise                   |
| `random`          | Randomly selects a transition for each cycle             |

#### Weather Providers

| Provider         | API Key Required | Location Support     | AQI Support | UV Support | Notes                        |
|------------------|------------------|----------------------|-------------|------------|------------------------------|
| `open-meteo`     | No               | Coordinates only     | Yes         | Yes        | Free, no registration needed |
| `openweathermap` | Yes              | City name or coords  | Yes         | Yes        | Free tier: 1000 calls/day    |

#### Weather Configuration

| Property         | Type              | Default           | Description                                       |
|------------------|-------------------|-------------------|---------------------------------------------------|
| `provider`       | string            | `"open-meteo"`    | Weather data provider                             |
| `api_key`        | string            | -                 | API key (required for openweathermap)             |
| `location`       | object            | -                 | Location settings (see below)                     |
| `units`          | string            | `"metric"`        | Temperature units: "metric" (C) or "imperial" (F) |
| `icon_size`      | int               | `16`              | Icon size in pixels (16 or 24)                    |
| `format`         | string or array   | `"{icon} {temp}"` | Display format(s) with tokens                     |
| `cycle`          | object            | -                 | Cycle and transition settings (see above)         |

#### Forecast Configuration

| Property       | Type   | Default | Description                      |
|----------------|--------|---------|----------------------------------|
| `hours`        | int    | `24`    | Hours for hourly forecast (6-48) |
| `days`         | int    | `3`     | Days for daily forecast (1-7)    |
| `scroll_speed` | number | `30`    | Pixels/second for scroll mode    |

Forecast data is automatically fetched when `{forecast:*}` tokens are used in the format string.

#### Air Quality and UV Index

AQI and UV data are automatically fetched when their tokens are used in the format string.

**AQI levels:** Good, Moderate, Unhealthy for Sensitive, Unhealthy, Very Unhealthy, Hazardous

**UV levels:** Low (0-2), Moderate (3-5), High (6-7), Very High (8-10), Extreme (11+)

#### Location Configuration

| Property | Type   | Description                                                       |
|----------|--------|-------------------------------------------------------------------|
| `city`   | string | City name (e.g., "London" or "New York,US"). OpenWeatherMap only. |
| `lat`    | number | Latitude coordinate (-90 to 90)                                   |
| `lon`    | number | Longitude coordinate (-180 to 180)                                |

#### Weather Icons

The widget displays appropriate icons for weather conditions:
- Sun (clear sky)
- Cloud (overcast)
- Partly cloudy (sun with cloud)
- Rain (cloud with raindrops)
- Drizzle (light rain)
- Snow (cloud with snowflakes)
- Storm (cloud with lightning)
- Fog (horizontal lines)

#### Examples

**Basic weather with icon:**
```json
{
  "type": "weather",
  "weather": {
    "format": "{icon} {temp}",
    "location": {"lat": 51.5074, "lon": -0.1278}
  }
}
```

**Multi-line with humidity and wind:**
```json
{
  "type": "weather",
  "weather": {
    "format": "{icon} {temp} {feels}\n{humidity} {wind} {wind_dir}",
    "location": {"lat": 51.5074, "lon": -0.1278}
  }
}
```

**Temperature graph:**
```json
{
  "type": "weather",
  "weather": {
    "format": "{forecast:graph}",
    "location": {"lat": 51.5074, "lon": -0.1278},
    "forecast": {
      "hours": 24
    }
  }
}
```

**Multi-day forecast with icons:**
```json
{
  "type": "weather",
  "weather": {
    "format": "{forecast:icons}",
    "location": {"lat": 51.5074, "lon": -0.1278},
    "forecast": {
      "days": 3
    }
  }
}
```

**Scrolling forecast:**
```json
{
  "type": "weather",
  "weather": {
    "format": "{forecast:scroll}",
    "location": {"lat": 51.5074, "lon": -0.1278},
    "forecast": {
      "scroll_speed": 30
    }
  }
}
```

**With air quality:**
```json
{
  "type": "weather",
  "weather": {
    "format": "{icon} {temp} AQI:{aqi}",
    "location": {"lat": 51.5074, "lon": -0.1278}
  }
}
```

**Cycling between displays with transitions:**
```json
{
  "type": "weather",
  "weather": {
    "format": [
      "{icon} {temp}",
      "{humidity} {wind}",
      "{aqi} {uv}",
      "{forecast:icons}"
    ],
    "cycle": {
      "interval": 10,
      "transition": "random",
      "speed": 0.5
    },
    "location": {"lat": 51.5074, "lon": -0.1278}
  }
}
```

#### Tips

- Use `update_interval: 300` (5 minutes) to avoid hitting API rate limits
- Open-Meteo is completely free and requires no registration
- For OpenWeatherMap, get a free API key at https://openweathermap.org/api
- Use coordinates (lat/lon) for more precise location
- AQI and UV tokens automatically enable their respective API fetching
- Large tokens (forecast:*) expand to fill available horizontal space
- Format cycling is useful for displaying more information on small screens
- Scroll mode combines current weather with hourly and daily forecasts

### Battery Widget

Displays device battery level and charging status. Supports multiple display modes including a battery-shaped progressbar, text percentage, bar, gauge, and historical graph. Configuration follows the same pattern as CPU widget: widget-level `mode` with mode-specific sections.

```json
{
  "type": "battery",
  "position": {"x": 0, "y": 0, "w": 128, "h": 40},
  "update_interval": 10,
  "mode": "battery",
  "power_status": {
    "show_charging": "always",
    "show_plugged": "always",
    "show_economy": "blink"
  },
  "battery": {
    "show_percentage": true
  }
}
```

#### Display Modes

| Mode      | Description                                              |
|-----------|----------------------------------------------------------|
| `battery` | Battery-shaped progressbar with fill level (default)     |
| `text`    | Formatted text with tokens (e.g., "{percent}% {status}") |
| `bar`     | Horizontal or vertical progress bar                      |
| `gauge`   | Circular gauge                                           |
| `graph`   | Historical battery level over time                       |

#### Battery Configuration

| Property             | Type   | Default      | Description                                  |
|----------------------|--------|--------------|----------------------------------------------|
| `orientation`        | string | `horizontal` | "horizontal" or "vertical" for battery mode  |
| `show_percentage`    | bool   | `true`       | Show percentage text                         |
| `low_threshold`      | int    | `20`         | Percentage below which battery is "low"      |
| `critical_threshold` | int    | `10`         | Percentage below which battery is "critical" |

#### Power Status Configuration

Controls how power status indicators (charging, plugged, economy mode) are displayed:

```json
"power_status": {
  "show_charging": "always",
  "show_plugged": "notify",
  "show_economy": "blink",
  "notify_duration": 60
}
```

| Property          | Type   | Default   | Description                                    |
|-------------------|--------|-----------|------------------------------------------------|
| `show_charging`   | string | `always`  | Display mode for charging indicator            |
| `show_plugged`    | string | `always`  | Display mode for AC power indicator            |
| `show_economy`    | string | `blink`   | Display mode for economy/power saver indicator |
| `notify_duration` | int    | `60`      | Seconds to show indicator in notify modes      |

**Display mode values:**
- `always` - Show indicator constantly when status is active
- `never` - Never show this indicator
- `notify` - Show for `notify_duration` seconds when status becomes active
- `blink` - Show indicator blinking when status is active
- `notify_blink` - Show blinking indicator for duration, then hide

#### Text Format Tokens

When using `mode: "text"`, you can customize the display format using tokens:

```json
"text": {
  "format": "{percent}% {status}"
}
```

| Token             | Description                                                     |
|-------------------|-----------------------------------------------------------------|
| `{percent}`       | Battery percentage (e.g., "85")                                 |
| `{pct}`           | Alias for `{percent}`                                           |
| `{status}`        | Short status: "CHG", "AC", "ECO", or "" (respects power_status) |
| `{status_full}`   | Full status: "Charging", "AC Power", "Economy", or ""           |
| `{time}`          | Smart: time to full (charging) or time to empty (discharging)   |
| `{time_left}`     | Time until empty (e.g., "1h 30m")                               |
| `{time_to_full}`  | Time until fully charged                                        |
| `{time_left_min}` | Raw minutes remaining as number                                 |
| `{level}`         | Battery level: "critical", "low", or "normal"                   |
| `{charging}`      | "CHG" if charging, "" otherwise (ignores power_status)          |
| `{plugged}`       | "AC" if plugged, "" otherwise (ignores power_status)            |
| `{economy}`       | "ECO" if economy mode, "" otherwise (ignores power_status)      |

**Examples:**
- `"{percent}%"` → "85%"
- `"{percent}% {status}"` → "85% CHG"
- `"{percent}% {time}"` → "85% 1h 30m"
- `"{level}: {percent}%"` → "normal: 85%"

#### Color Configuration (battery.colors)

Colors use 0-255 range. Setting a color to 0 (black) is supported.

| Property     | Type | Default | Description                        |
|--------------|------|---------|------------------------------------|
| `normal`     | int  | `255`   | Fill color when battery is normal  |
| `low`        | int  | `200`   | Fill color when battery is low     |
| `critical`   | int  | `150`   | Fill color when critical           |
| `charging`   | int  | `255`   | Charging indicator color           |
| `background` | int  | `0`     | Background inside battery body     |
| `border`     | int  | `255`   | Battery outline color              |

#### Mode-Specific Settings

For bar/graph/gauge modes, use the shared widget-level configurations:

**Bar mode (`bar`):**
- `direction`: "left", "right", "up", "down" (up/down = vertical)
- `border`: Show border around bar

**Graph mode (`graph`):**
- `history`: Number of data points (default: 60)
- `filled`: Fill under the graph line

**Gauge mode (`gauge`):**
- `show_ticks`: Show tick marks

#### Examples

**Battery mode with percentage:**
```json
{
  "type": "battery",
  "position": {"x": 0, "y": 0, "w": 64, "h": 40},
  "mode": "battery",
  "power_status": {
    "show_charging": "always",
    "show_economy": "blink"
  },
  "battery": {
    "show_percentage": true
  }
}
```

**Text mode with format tokens:**
```json
{
  "type": "battery",
  "position": {"x": 0, "y": 0, "w": 128, "h": 40},
  "mode": "text",
  "text": {
    "format": "{percent}% {status} {time}"
  }
}
```

**Vertical bar:**
```json
{
  "type": "battery",
  "position": {"x": 0, "y": 0, "w": 20, "h": 40},
  "mode": "bar",
  "bar": {
    "direction": "up",
    "border": true
  }
}
```

**Circular gauge:**
```json
{
  "type": "battery",
  "position": {"x": 0, "y": 0, "w": 40, "h": 40},
  "mode": "gauge",
  "gauge": {
    "show_ticks": true
  }
}
```

**Historical graph:**
```json
{
  "type": "battery",
  "position": {"x": 0, "y": 0, "w": 128, "h": 40},
  "mode": "graph",
  "graph": {
    "history": 200,
    "filled": true
  }
}
```

**Custom colors for low battery:**
```json
{
  "type": "battery",
  "mode": "icon",
  "battery": {
    "low_threshold": 30,
    "critical_threshold": 15,
    "colors": {
      "normal": 255,
      "low": 150,
      "critical": 80
    }
  }
}
```

#### Platform Support

- **Windows**: Uses GetSystemPowerStatus API
- **Linux**: Reads from /sys/class/power_supply/

#### Tips

- Use `update_interval: 10` or higher to avoid excessive system calls
- Icon mode supports both horizontal and vertical orientation (via `bar.direction`)
- Horizontal: Battery with terminal on right, status icon in top-left corner
- Vertical: Battery with terminal on top, status icon at bottom-center
- Status indicators (charging bolt, AC plug) have white fill with black border for visibility
- All colors support 0 (black) values

### Game of Life Widget

Displays Conway's Game of Life cellular automaton - a classic zero-player game where patterns evolve based on simple rules. The 128x40 display provides 5,120 cells for emergent complexity.

```json
{
  "type": "game_of_life",
  "position": {"x": 0, "y": 0, "w": 128, "h": 40},
  "update_interval": 0.1,
  "game_of_life": {
    "rules": "B3/S23",
    "wrap_edges": true,
    "initial_pattern": "random",
    "random_density": 0.35,
    "cell_size": 1,
    "trail_effect": true,
    "trail_decay": 25,
    "cell_color": 255
  }
}
```

#### Speed Control

The simulation speed is controlled by `update_interval` (in seconds):
- `0.05` - Very fast (20 generations/second)
- `0.1` - Fast (10 generations/second, recommended)
- `0.2` - Medium (5 generations/second)
- `0.5` - Slow (2 generations/second)

#### Configuration

| Property          | Type    | Default    | Description                                                |
|-------------------|---------|------------|------------------------------------------------------------|
| `rules`           | string  | `"B3/S23"` | Birth/Survival rules in B/S notation                       |
| `wrap_edges`      | boolean | `true`     | Wrap edges (torus topology)                                |
| `initial_pattern` | string  | `"random"` | Starting pattern                                           |
| `random_density`  | number  | `0.3`      | Cell density for random pattern (0.0-1.0)                  |
| `cell_size`       | integer | `1`        | Pixels per cell (1-4)                                      |
| `trail_effect`    | boolean | `true`     | Enable fading trail when cells die                         |
| `trail_decay`     | integer | `30`       | Brightness decay per frame (1-255, higher = faster)        |
| `cell_color`      | integer | `255`      | Alive cell brightness (1-255)                              |
| `restart_timeout` | number  | `3.0`      | Seconds to wait before restart (0 = immediate, -1 = never) |
| `restart_mode`    | string  | `"reset"`  | How to restart: "reset", "inject", or "random"             |

#### Rules Format

Rules use B/S notation: `B` followed by birth neighbor counts, `/S` followed by survival counts.

| Rule           | Name               | Description                                    |
|----------------|--------------------|------------------------------------------------|
| `B3/S23`       | Conway             | Standard rules - balanced complexity (default) |
| `B36/S23`      | HighLife           | Like Conway but with replicators               |
| `B1357/S1357`  | Replicator         | Patterns replicate themselves                  |
| `B2/S`         | Seeds              | Explosive growth                               |
| `B3/S12345678` | Life without Death | Cells never die                                |

#### Initial Patterns

| Pattern       | Description                                                  |
|---------------|--------------------------------------------------------------|
| `random`      | Random cells based on `random_density`                       |
| `clear`       | Empty grid                                                   |
| `glider`      | Small pattern that moves diagonally                          |
| `r_pentomino` | Methuselah - small pattern that evolves for 1103 generations |
| `acorn`       | Another methuselah - evolves for 5206 generations            |
| `diehard`     | Pattern that dies after 130 generations                      |
| `lwss`        | Lightweight spaceship - moves horizontally                   |
| `pulsar`      | Period-3 oscillator - stable and mesmerizing                 |
| `glider_gun`  | Gosper glider gun - produces infinite stream of gliders      |

#### Examples

**Fast random simulation with trails:**
```json
{
  "type": "game_of_life",
  "position": {"x": 0, "y": 0, "w": 128, "h": 40},
  "update_interval": 0.05,
  "game_of_life": {
    "initial_pattern": "random",
    "random_density": 0.4,
    "trail_effect": true,
    "trail_decay": 20
  }
}
```

**Glider gun (infinite gliders):**
```json
{
  "type": "game_of_life",
  "position": {"x": 0, "y": 0, "w": 128, "h": 40},
  "update_interval": 0.1,
  "game_of_life": {
    "initial_pattern": "glider_gun",
    "wrap_edges": true
  }
}
```

**HighLife with larger cells:**
```json
{
  "type": "game_of_life",
  "position": {"x": 0, "y": 0, "w": 128, "h": 40},
  "update_interval": 0.15,
  "game_of_life": {
    "rules": "B36/S23",
    "initial_pattern": "random",
    "cell_size": 2,
    "trail_effect": false
  }
}
```

#### Tips

- Use `wrap_edges: true` for patterns that move (gliders, spaceships)
- Lower `trail_decay` for longer ghost trails
- `cell_size: 2` gives 64x20 grid - easier to see individual cells
- `glider_gun` needs `wrap_edges: true` or gliders pile up at edges
- `pulsar` is good for testing - stable, predictable oscillation
- Simulation restarts when all cells die or pattern becomes stable
- Set `restart_timeout: -1` to disable auto-restart (stays in final state)
- Set `restart_timeout: 0` to restart immediately without pause
- `restart_mode: "inject"` adds new cells to existing survivors - keeps the game evolving
- `restart_mode: "random"` always uses fresh random pattern, ignoring initial_pattern

### Hyperspace Widget

Displays the Star Wars hyperspace/lightspeed jump effect with stars streaking toward or away from a vanishing point.

```json
{
  "type": "hyperspace",
  "position": {"x": 0, "y": 0, "w": 128, "h": 40},
  "update_interval": 0.033,
  "hyperspace": {
    "star_count": 100,
    "speed": 0.02,
    "max_speed": 0.5,
    "trail_length": 1.0,
    "star_color": 255,
    "mode": "continuous"
  }
}
```

#### Animation Modes

| Mode         | Description                                                       |
|--------------|-------------------------------------------------------------------|
| `continuous` | Always in hyperspace at max speed - constant light streaks        |
| `cycle`      | Phases: idle (slow stars) -> jump -> hyperspace travel -> exit    |

#### Configuration

| Property       | Type    | Default        | Description                                     |
|----------------|---------|----------------|-------------------------------------------------|
| `star_count`   | integer | `100`          | Number of stars in the field (10-500)           |
| `speed`        | number  | `0.02`         | Base speed for idle phase                       |
| `max_speed`    | number  | `0.5`          | Maximum speed during hyperspace                 |
| `trail_length` | number  | `1.0`          | Trail length multiplier (0.1-5.0)               |
| `center_x`     | integer | widget center  | Focal point X coordinate                        |
| `center_y`     | integer | widget center  | Focal point Y coordinate                        |
| `star_color`   | integer | `255`          | Maximum star brightness (1-255)                 |
| `mode`         | string  | `"continuous"` | Animation mode: "continuous" or "cycle"         |
| `idle_time`    | number  | `5.0`          | Seconds in idle phase before jump (cycle mode)  |
| `travel_time`  | number  | `3.0`          | Seconds in hyperspace (cycle mode)              |
| `acceleration` | number  | `0.1`          | Speed change rate during jump/exit (cycle mode) |

#### Cycle Mode Phases

In `cycle` mode, the animation goes through four phases:

1. **Idle** - Stars drift slowly toward viewer for `idle_time` seconds
2. **Jump** - Stars accelerate from `speed` to `max_speed`
3. **Hyperspace** - Full light-streak effect for `travel_time` seconds
4. **Exit** - Stars decelerate back to `speed`, then returns to Idle

#### Examples

**Continuous hyperspace (always fast):**
```json
{
  "type": "hyperspace",
  "position": {"x": 0, "y": 0, "w": 128, "h": 40},
  "update_interval": 0.033,
  "hyperspace": {
    "mode": "continuous",
    "star_count": 150,
    "max_speed": 0.6,
    "trail_length": 1.5
  }
}
```

**Cycle mode with jumps:**
```json
{
  "type": "hyperspace",
  "position": {"x": 0, "y": 0, "w": 128, "h": 40},
  "update_interval": 0.033,
  "hyperspace": {
    "mode": "cycle",
    "star_count": 100,
    "idle_time": 5.0,
    "travel_time": 3.0,
    "acceleration": 0.15
  }
}
```

**Off-center focal point:**
```json
{
  "type": "hyperspace",
  "position": {"x": 0, "y": 0, "w": 128, "h": 40},
  "update_interval": 0.033,
  "hyperspace": {
    "center_x": 32,
    "center_y": 20
  }
}
```

**Dense starfield with long trails:**
```json
{
  "type": "hyperspace",
  "position": {"x": 0, "y": 0, "w": 128, "h": 40},
  "update_interval": 0.033,
  "hyperspace": {
    "star_count": 200,
    "trail_length": 2.0,
    "max_speed": 0.4
  }
}
```

#### Tips

- Use `update_interval: 0.033` (30 FPS) for smooth animation
- Higher `star_count` gives denser starfield but uses more CPU
- `trail_length > 1.0` creates longer light streaks
- `center_x`/`center_y` can create asymmetric "flying sideways" effect
- In `cycle` mode, lower `acceleration` gives smoother transitions
- Combine with transparent overlay to add hyperspace behind other widgets

### Star Wars Intro Widget

Displays the complete iconic Star Wars opening sequence with three phases:
1. **Pre-intro**: "A long time ago in a galaxy far, far away...." fades in and out
2. **Logo**: "STAR WARS" logo appears at full size and shrinks toward the vanishing point
3. **Crawl**: Text scrolls upward with perspective, slanted letters matching the tilt angle

Background stars are displayed during the logo and crawl phases.

```json
{
  "type": "starwars_intro",
  "position": {"x": 0, "y": 0, "w": 128, "h": 40},
  "update_interval": 0.05,
  "starwars_intro": {
    "pre_intro": {
      "enabled": true,
      "text": "A long time ago in a galaxy far, far away...."
    },
    "logo": {
      "enabled": true,
      "text": "STAR\nWARS"
    },
    "text": [
      "Episode IV",
      "A NEW HOPE",
      "",
      "It is a period of civil war."
    ]
  }
}
```

#### Animation Phases

| Phase     | Description                                              |
|-----------|----------------------------------------------------------|
| Pre-intro | Blue text fades in, holds, then fades out                |
| Logo      | Logo displays at full size, then shrinks toward center   |
| Crawl     | Perspective text scrolls upward with slanted letters     |
| End pause | Optional pause before looping (if loop enabled)          |

#### Pre-intro Configuration (`pre_intro`)

| Option     | Type    | Default                                           | Description                                |
|------------|---------|---------------------------------------------------|--------------------------------------------|
| `enabled`  | boolean | `true`                                            | Show the pre-intro phase                   |
| `text`     | string  | `"A long time ago in a galaxy far, far away...."` | The pre-intro message (use `\n` for lines) |
| `color`    | integer | `80`                                              | Text brightness (dim blue look)            |
| `fade_in`  | number  | `2.0`                                             | Fade in duration (seconds)                 |
| `hold`     | number  | `2.0`                                             | Hold duration after fade in                |
| `fade_out` | number  | `1.0`                                             | Fade out duration (seconds)                |

#### Logo Configuration (`logo`)

| Option            | Type    | Default        | Description                              |
|-------------------|---------|----------------|------------------------------------------|
| `enabled`         | boolean | `true`         | Show the logo phase                      |
| `text`            | string  | `"STAR\nWARS"` | Logo text (use \n for line breaks)       |
| `color`           | integer | `255`          | Logo brightness                          |
| `hold_before`     | number  | `0.5`          | Seconds to hold at full size             |
| `shrink_duration` | number  | `4.0`          | Seconds for shrink animation             |
| `final_scale`     | number  | `0.1`          | Scale at which logo disappears (0.0-0.5) |

#### Stars Configuration (`stars`)

| Option       | Type    | Default | Description              |
|--------------|---------|---------|--------------------------|
| `enabled`    | boolean | `true`  | Show background stars    |
| `count`      | integer | `50`    | Number of stars (10-200) |
| `brightness` | integer | `200`   | Maximum star brightness  |

#### Crawl Configuration

| Option         | Type    | Default        | Description                                      |
|----------------|---------|----------------|--------------------------------------------------|
| `text`         | array   | (default text) | Array of strings for the crawl                   |
| `scroll_speed` | number  | `0.5`          | Scroll speed in pixels per frame                 |
| `perspective`  | number  | `0.7`          | Perspective strength (0.0 = none, 1.0 = maximum) |
| `slant`        | number  | `30.0`         | Text italic/slant angle in degrees               |
| `fade_top`     | number  | `0.3`          | Position where fade starts (0.0 = top)           |
| `text_color`   | integer | `255`          | Text brightness (1-255)                          |
| `line_spacing` | integer | `8`            | Pixels between text lines                        |
| `loop`         | boolean | `true`         | Loop the entire sequence                         |
| `pause_at_end` | number  | `3.0`          | Seconds to pause before looping                  |

#### Examples

**Full movie-accurate intro:**
```json
{
  "type": "starwars_intro",
  "position": {"x": 0, "y": 0, "w": 128, "h": 40},
  "update_interval": 0.05,
  "starwars_intro": {
    "pre_intro": {
      "fade_in": 2.5,
      "hold": 3.0,
      "fade_out": 1.5
    },
    "logo": {
      "shrink_duration": 5.0
    },
    "stars": {
      "count": 80
    },
    "text": [
      "Episode IV",
      "A NEW HOPE",
      "",
      "It is a period of civil war.",
      "Rebel spaceships, striking",
      "from a hidden base, have won",
      "their first victory against",
      "the evil Galactic Empire."
    ],
    "slant": 20.0
  }
}
```

**Crawl only (skip intro and logo):**
```json
{
  "type": "starwars_intro",
  "position": {"x": 0, "y": 0, "w": 128, "h": 40},
  "update_interval": 0.05,
  "starwars_intro": {
    "pre_intro": {"enabled": false},
    "logo": {"enabled": false},
    "text": ["Custom message", "scrolling with", "perspective effect"]
  }
}
```

**Custom branding:**
```json
{
  "type": "starwars_intro",
  "position": {"x": 0, "y": 0, "w": 128, "h": 40},
  "update_interval": 0.05,
  "starwars_intro": {
    "pre_intro": {
      "text": "In a workspace not so far away...."
    },
    "logo": {
      "text": "STEEL\nCLOCK"
    },
    "text": [
      "Version 1.0",
      "",
      "A powerful tool for",
      "keyboard displays."
    ]
  }
}
```

#### Tips

- Use `update_interval: 0.05` (20 FPS) for smooth animation
- The `slant` angle should roughly match `perspective` for natural appearance
- Higher `perspective` values create stronger "receding into distance" effect
- Empty strings in `text` array create blank lines for spacing
- Keep text lines short to fit within the display width at perspective scale
- Set `pre_intro.enabled: false` and `logo.enabled: false` to skip directly to crawl

### Telegram Widget

Displays notifications from your personal Telegram account. Uses the official Telegram MTProto API to receive real-time messages from private chats, groups, and channels.

#### Prerequisites

1. **Create Telegram App**: Go to [my.telegram.org](https://my.telegram.org) and create an application to get your `api_id` and `api_hash`.
2. **First Run Authentication**: On first run, you'll be prompted to enter a verification code sent to your Telegram account.

#### Authentication Properties (at widget root level)

| Property              | Type    | Default                              | Description                                                |
|-----------------------|---------|--------------------------------------|------------------------------------------------------------|
| `auth.api_id`         | integer | required                             | Telegram API ID from my.telegram.org                       |
| `auth.api_hash`       | string  | required                             | Telegram API Hash from my.telegram.org                     |
| `auth.phone_number`   | string  | required                             | Phone number in international format (e.g., "+1234567890") |
| `auth.session_path`   | string  | "telegram/{api_id}_{phone}.session"  | Path to session file for persistent login                  |

#### Filter Configuration (at widget root level)

| Property                                | Type    | Default | Description                                             |
|-----------------------------------------|---------|---------|---------------------------------------------------------|
| `filters.private_chats.enabled`         | boolean | true    | Include messages from private chats                     |
| `filters.private_chats.whitelist`       | array   | []      | Always include these chat IDs, even if type is disabled |
| `filters.private_chats.blacklist`       | array   | []      | Never include these chat IDs, even if type is enabled   |
| `filters.private_chats.pinned_messages` | boolean | true    | Include pinned message notifications                    |
| `filters.groups.enabled`                | boolean | false   | Include messages from groups                            |
| `filters.groups.whitelist`              | array   | []      | Always include these group IDs                          |
| `filters.groups.blacklist`              | array   | []      | Never include these group IDs                           |
| `filters.groups.pinned_messages`        | boolean | true    | Include pinned message notifications                    |
| `filters.channels.enabled`              | boolean | false   | Include messages from channels                          |
| `filters.channels.whitelist`            | array   | []      | Always include these channel IDs                        |
| `filters.channels.blacklist`            | array   | []      | Never include these channel IDs                         |
| `filters.channels.pinned_messages`      | boolean | false   | Include pinned message notifications                    |

#### Appearance Configuration (at widget root level)

| Property                         | Type    | Default  | Description                                   |
|----------------------------------|---------|----------|-----------------------------------------------|
| `appearance.header.enabled`      | boolean | true     | Show header (sender/chat name)                |
| `appearance.header.blink`        | boolean | false    | Make header blink                             |
| `appearance.header.text`         | object  | -        | Text rendering settings (font, size, align)   |
| `appearance.header.scroll`       | object  | -        | Scroll settings (enabled, direction, speed)   |
| `appearance.message.enabled`     | boolean | true     | Show message content                          |
| `appearance.message.blink`       | boolean | false    | Make message blink                            |
| `appearance.message.text`        | object  | -        | Text rendering settings                       |
| `appearance.message.scroll`      | object  | -        | Scroll settings                               |
| `appearance.message.word_break`  | string  | "normal" | How to break lines: "normal" or "break-all"   |
| `appearance.separator.color`     | integer | 128      | Separator line color (0-255)                  |
| `appearance.separator.thickness` | integer | 1        | Separator line thickness (0 = disabled)       |
| `appearance.timeout`             | integer | 0        | Seconds to show notification (0 = until next) |
| `appearance.transitions.in`      | string  | "none"   | Transition effect when showing                |
| `appearance.transitions.out`     | string  | "none"   | Transition effect when hiding                 |

#### Example Configuration

```json
{
  "type": "telegram",
  "position": {"x": 0, "y": 0, "w": 128, "h": 40},
  "auth": {
    "api_id": 12345678,
    "api_hash": "your_api_hash_here",
    "phone_number": "+1234567890"
  },
  "filters": {
    "private_chats": {
      "enabled": true,
      "whitelist": [],
      "blacklist": [],
      "pinned_messages": true
    },
    "groups": {
      "enabled": false,
      "whitelist": ["123456789"],
      "blacklist": []
    },
    "channels": {
      "enabled": false
    }
  },
  "appearance": {
    "header": {
      "enabled": true,
      "text": {"font": "5x7", "size": 1}
    },
    "message": {
      "enabled": true,
      "scroll": {"enabled": true, "direction": "left", "speed": 30}
    },
    "timeout": 10,
    "transitions": {"in": "random", "out": "random"}
  }
}
```

#### Tips

- **Security**: The session file contains your Telegram login credentials. Keep it secure and don't share it.
- **First Run**: Authentication happens on first run via console prompts. Ensure you can see the console output.
- **Group/Channel IDs**: You can find chat IDs using Telegram bots like @userinfobot or by forwarding a message to @RawDataBot.
- **Whitelist/Blacklist**: Whitelist has priority over enabled setting; blacklist has priority over whitelist.
- **2FA**: If you have Two-Factor Authentication enabled, you'll be prompted for your password on first login.

---

### Telegram Counter Widget

Displays unread message count from your Telegram account. Uses the same Telegram MTProto API as the Telegram widget. Can be used alongside the Telegram notification widget - both share the same client connection.

**Important**: This widget has independent filter settings from the Telegram notification widget. Each widget can count different chat types.

#### Prerequisites

Same as Telegram Widget - see above.

#### Authentication Properties (at widget root level)

| Property              | Type    | Default                              | Description                                                |
|-----------------------|---------|--------------------------------------|------------------------------------------------------------|
| `auth.api_id`         | integer | required                             | Telegram API ID from my.telegram.org                       |
| `auth.api_hash`       | string  | required                             | Telegram API Hash from my.telegram.org                     |
| `auth.phone_number`   | string  | required                             | Phone number in international format (e.g., "+1234567890") |
| `auth.session_path`   | string  | "telegram/{api_id}_{phone}.session"  | Path to session file for persistent login                  |

#### Filter Configuration (independent from telegram widget)

Same structure as Telegram Widget filters - can be configured independently.

#### Display Settings (at widget root level)

| Property                  | Type    | Default           | Description                                     |
|---------------------------|---------|-------------------|-------------------------------------------------|
| `mode`                    | string  | "badge"           | Display mode: "badge" or "text"                 |
| `badge.blink`             | string  | "never"           | Blink mode: "never", "always", or "progressive" |
| `badge.colors.foreground` | integer | 255 (white)       | Icon foreground color (0-255, -1 = transparent) |
| `badge.colors.background` | integer | 0 (black)         | Icon background color (0-255, -1 = transparent) |
| `text.format`             | string  | "{unread} unread" | Format string for text mode (see tokens below)  |
| `text.font`               | string  | null              | Font name or TTF path (null = bundled font)     |
| `text.size`               | integer | 16                | Font size in pixels                             |
| `text.align.h`            | string  | "center"          | Horizontal alignment: "left", "center", "right" |
| `text.align.v`            | string  | "center"          | Vertical alignment: "top", "center", "bottom"   |

#### Display Modes

- **`badge`**: Shows the Telegram paper airplane icon (size auto-scales to widget dimensions)
- **`text`**: Shows formatted text with tokens (see below)

#### Text Format Tokens

Available tokens for `text.format`:

| Token              | Description                                      |
|--------------------|--------------------------------------------------|
| `{icon}`           | Telegram paper airplane icon (sized to font)     |
| `{unread}`         | Total unread messages                            |
| `{total}`          | Same as {unread}                                 |
| `{mentions}`       | Unread @mentions                                 |
| `{reactions}`      | Unread reactions                                 |
| `{private}`        | Unread in private chats                          |
| `{groups}`         | Unread in groups                                 |
| `{channels}`       | Unread in channels                               |
| `{muted}`          | Unread in muted chats                            |
| `{private_muted}`  | Unread in muted private chats                    |
| `{groups_muted}`   | Unread in muted groups                           |
| `{channels_muted}` | Unread in muted channels                         |

Examples:
- `"{icon} {unread}"` -> icon followed by "5"
- `"{unread} ({mentions}@)"` -> "5 (2@)"

#### Blink Modes

- **`never`**: No blinking
- **`always`**: Constant 2Hz blink (toggles every 500ms)
- **`progressive`**: Blink frequency increases with unread count:
  - 1 message: 1 blink/second
  - 5 messages: 5 blinks/second
  - 10+ messages: 10 blinks/second

#### Example Configuration (Badge Mode)

```json
{
  "type": "telegram_counter",
  "position": {"x": 100, "y": 0, "w": 28, "h": 40},
  "auth": {
    "api_id": 12345678,
    "api_hash": "your_api_hash_here",
    "phone_number": "+1234567890"
  },
  "mode": "badge",
  "badge": {
    "blink": "progressive",
    "colors": {
      "foreground": 255,
      "background": -1
    }
  }
}
```

#### Example Configuration (Text Mode)

```json
{
  "type": "telegram_counter",
  "position": {"x": 0, "y": 0, "w": 128, "h": 40},
  "auth": {
    "api_id": 12345678,
    "api_hash": "your_api_hash_here",
    "phone_number": "+1234567890"
  },
  "mode": "text",
  "text": {
    "format": "{unread} new ({mentions} mentions)",
    "font": null,
    "size": 16,
    "align": {"h": "center", "v": "center"}
  }
}
```

#### Using Both Widgets Together

You can use `telegram` and `telegram_counter` widgets simultaneously. They share the same Telegram client connection based on `api_id` and `phone_number`, so authentication only happens once. Each widget has its own independent filter settings.

```json
{
  "widgets": [
    {
      "type": "telegram",
      "position": {"x": 0, "y": 0, "w": 100, "h": 40},
      "auth": {
        "api_id": 12345678,
        "api_hash": "your_api_hash_here",
        "phone_number": "+1234567890"
      },
      "filters": {
        "private_chats": {"enabled": true}
      }
    },
    {
      "type": "telegram_counter",
      "position": {"x": 100, "y": 0, "w": 28, "h": 40},
      "style": {"background": -1},
      "auth": {
        "api_id": 12345678,
        "api_hash": "your_api_hash_here",
        "phone_number": "+1234567890"
      },
      "filters": {
        "private_chats": {"enabled": true},
        "groups": {"enabled": true},
        "channels": {"enabled": true}
      },
      "mode": "badge",
      "badge": {"blink": "progressive"}
    }
  ]
}
```

---

## Examples

### Example 1: Simple Clock

```json
{
  "$schema": "schema/config.schema.json",
  "schema_version": 2,
  "display": {
    "width": 128,
    "height": 40,
    "background": 0
  },
  "widgets": [
    {
      "type": "clock",
      "position": {
        "x": 0,
        "y": 0,
        "w": 128,
        "h": 40
      },
      "mode": "text",
      "text": {
        "format": "%H:%M:%S",
        "size": 16,
        "align": {
          "h": "center",
          "v": "center"
        }
      }
    }
  ]
}
```

### Example 2: CPU + Memory Bars

```json
{
  "$schema": "schema/config.schema.json",
  "schema_version": 2,
  "display": {
    "width": 128,
    "height": 40,
    "background": 0
  },
  "widgets": [
    {
      "type": "cpu",
      "position": {
        "x": 0,
        "y": 0,
        "w": 128,
        "h": 20,
        "z": 0
      },
      "style": {
        "border": 255
      },
      "mode": "bar",
      "bar": {
        "direction": "horizontal",
        "colors": {
          "fill": 255
        }
      }
    },
    {
      "type": "memory",
      "position": {
        "x": 0,
        "y": 20,
        "w": 128,
        "h": 20,
        "z": 0
      },
      "style": {
        "border": 255
      },
      "mode": "bar",
      "bar": {
        "direction": "horizontal",
        "colors": {
          "fill": 255
        }
      }
    }
  ]
}
```

### Example 3: Transparent Overlay

```json
{
  "$schema": "schema/config.schema.json",
  "schema_version": 2,
  "display": {
    "width": 128,
    "height": 40,
    "background": 0
  },
  "widgets": [
    {
      "type": "network",
      "position": {
        "x": 0,
        "y": 0,
        "w": 128,
        "h": 40,
        "z": 0
      },
      "mode": "graph",
      "graph": {
        "history": 30,
        "colors": {
          "rx": 200,
          "tx": 100
        }
      }
    },
    {
      "type": "clock",
      "position": {
        "x": 0,
        "y": 0,
        "w": 128,
        "h": 40,
        "z": 10
      },
      "style": {
        "background": -1
      },
      "mode": "text",
      "text": {
        "format": "%H:%M",
        "size": 16
      }
    }
  ]
}
```

### Example 4: Gauge Dashboard

```json
{
  "$schema": "schema/config.schema.json",
  "schema_version": 2,
  "display": {
    "width": 128,
    "height": 40,
    "background": 0
  },
  "widgets": [
    {
      "type": "cpu",
      "position": {
        "x": 0,
        "y": 0,
        "w": 64,
        "h": 40
      },
      "style": {
        "border": 255
      },
      "mode": "gauge",
      "gauge": {
        "show_ticks": true,
        "colors": {
          "arc": 200,
          "needle": 255,
          "ticks": 150
        }
      }
    },
    {
      "type": "memory",
      "position": {
        "x": 64,
        "y": 0,
        "w": 64,
        "h": 40
      },
      "style": {
        "border": 255
      },
      "mode": "gauge",
      "gauge": {
        "show_ticks": true,
        "colors": {
          "arc": 180,
          "needle": 255,
          "ticks": 150
        }
      }
    }
  ]
}
```

### Example 5: Audio Visualizer

```json
{
  "$schema": "schema/config.schema.json",
  "schema_version": 2,
  "refresh_rate_ms": 33,
  "display": {
    "width": 128,
    "height": 40,
    "background": 0
  },
  "widgets": [
    {
      "type": "audio_visualizer",
      "position": {
        "x": 0,
        "y": 0,
        "w": 128,
        "h": 40
      },
      "mode": "spectrum",
      "spectrum": {
        "bars": 32,
        "scale": "logarithmic",
        "smoothing": 0.7,
        "peak": {
          "enabled": true,
          "hold_time": 1.0
        },
        "colors": {
          "fill": 255
        }
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

| Content      | Recommended Interval |
|--------------|----------------------|
| Clock        | 1.0s                 |
| CPU/Memory   | 1.0s                 |
| Network/Disk | 1.0s                 |
| Keyboard     | 0.2s                 |
| Audio        | 0.033s (30 FPS)      |

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
