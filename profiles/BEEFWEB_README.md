# Beefweb Widget Setup Guide

The beefweb widget displays track information from **Foobar2000** (Windows) or **DeaDBeeF** (Linux) music players using the beefweb REST API plugin.

## Requirements

- Foobar2000 (Windows) or DeaDBeeF (Linux)
- beefweb plugin installed and enabled

## Installation

### Foobar2000 (Windows)

1. **Download the plugin**
   - Go to: https://github.com/hyperblast/beefweb/releases
   - Download the latest `foo_beefweb-*.zip` file

2. **Install the plugin**
   - Extract the archive
   - Copy `foo_beefweb.dll` to Foobar2000 components folder:
     - Default: `C:\Program Files\foobar2000\components\`
     - Or: `C:\Program Files (x86)\foobar2000\components\`
   - Restart Foobar2000

3. **Verify installation**
   - In Foobar2000, go to **File > Preferences > Tools**
   - You should see "beefweb Remote Control" in the list

### DeaDBeeF (Linux)

1. **Install from package manager** (if available)
   ```bash
   # Check your distribution's repositories
   ```

2. **Or build from source**
   ```bash
   git clone https://github.com/hyperblast/beefweb.git
   cd beefweb
   # Follow build instructions in the repository
   ```

3. **Enable the plugin**
   - In DeaDBeeF, go to **Edit > Preferences > Plugins**
   - Enable the beefweb plugin

## Configuration

### Plugin Settings

1. In Foobar2000: **File > Preferences > Tools > beefweb Remote Control**
2. In DeaDBeeF: **Edit > Preferences > Plugins > beefweb**

Default settings:
- **Port**: 8880
- **Allow remote connections**: Disabled (localhost only)

### Verify Plugin Works

Open a browser and navigate to:
```
http://localhost:8880
```

You should see the beefweb web interface. If you see it, the plugin is working correctly.

To test the API directly:
```
http://localhost:8880/api/player
```

This should return JSON with player state information.

## SteelClock Configuration

### Basic Configuration

```json
{
  "type": "beefweb",
  "position": { "x": 0, "y": 0, "w": 128, "h": 40 },
  "text": {
    "format": "{artist} - {title}"
  }
}
```

### Full Configuration Example

```json
{
  "type": "beefweb",
  "id": "player",
  "enabled": true,
  "position": {
    "x": 0,
    "y": 0,
    "w": 128,
    "h": 40,
    "z": 0
  },
  "style": {
    "background": 0,
    "border": -1,
    "padding": 2
  },
  "text": {
    "font": "5x7",
    "size": 12,
    "format": "{artist} - {title}",
    "align": {
      "h": "left",
      "v": "center"
    }
  },
  "beefweb": {
    "server_url": "http://localhost:8880",
    "placeholder": {
      "mode": "text",
      "text": "[Not running]"
    }
  },
  "beefweb_auto_show": {
    "on_track_change": true,
    "on_play": true,
    "on_pause": false,
    "on_stop": false,
    "duration_sec": 5
  },
  "scroll": {
    "enabled": true,
    "direction": "left",
    "speed": 30,
    "mode": "pause_ends",
    "pause_ms": 2000,
    "gap": 20
  },
  "auto_hide": {
    "enabled": false,
    "timeout": 5
  },
  "update_interval": 0.5
}
```

### Configuration Options

#### beefweb section

| Option             | Type   | Default                 | Description                                            |
|--------------------|--------|-------------------------|--------------------------------------------------------|
| `server_url`       | string | `http://localhost:8880` | Beefweb server URL                                     |
| `placeholder.mode` | string | `text`                  | What to show when player not running: `text` or `hide` |
| `placeholder.text` | string | `[Not running]`         | Text to display when mode is `text`                    |

#### beefweb_auto_show section

| Option            | Type   | Default | Description                           |
|-------------------|--------|---------|---------------------------------------|
| `on_track_change` | bool   | `true`  | Show widget when track changes        |
| `on_play`         | bool   | `false` | Show widget when playback starts      |
| `on_pause`        | bool   | `false` | Show widget when playback pauses      |
| `on_stop`         | bool   | `false` | Show widget when playback stops       |
| `duration_sec`    | number | `5`     | How long to show the widget (seconds) |

#### Format Tokens

| Token        | Description              | Example          |
|--------------|--------------------------|------------------|
| `{artist}`   | Track artist             | Pink Floyd       |
| `{title}`    | Track title              | Comfortably Numb |
| `{album}`    | Album name               | The Wall         |
| `{position}` | Current position (MM:SS) | 02:15            |
| `{duration}` | Track duration (MM:SS)   | 06:24            |
| `{state}`    | Playback state           | Playing          |

## Troubleshooting

### Widget shows "[Not running]"

1. Check that Foobar2000/DeaDBeeF is running
2. Verify beefweb plugin is installed and enabled
3. Test API access: open `http://localhost:8880/api/player` in browser
4. Check `server_url` in your config matches the plugin port

### Widget shows no track info

1. Make sure a track is loaded and playing/paused
2. Check that the track has proper metadata tags (artist, title, album)

### Connection refused

1. Check that the plugin is enabled in player settings
2. Verify the port number matches your config
3. If using a custom port, update `server_url` accordingly

### Custom port configuration

If you changed the beefweb port (e.g., to 9000):

```json
"beefweb": {
  "server_url": "http://localhost:9000"
}
```

## Remote Access

By default, beefweb only accepts connections from localhost. To allow remote connections:

1. In plugin settings, enable "Allow remote connections"
2. Update `server_url` to use the machine's IP address:
   ```json
   "beefweb": {
     "server_url": "http://192.168.1.100:8880"
   }
   ```

**Security note**: Enabling remote connections exposes your player to the network. Use only on trusted networks.

## Links

- beefweb repository: https://github.com/hyperblast/beefweb
- beefweb releases: https://github.com/hyperblast/beefweb/releases
- Foobar2000: https://www.foobar2000.org/
- DeaDBeeF: https://deadbeef.sourceforge.io/
