# Spotify Widget Setup Guide

This guide explains how to configure the Spotify widget to display currently playing track information on your SteelSeries OLED display.

## Prerequisites

- Spotify account (Free or Premium)
- Spotify desktop or web client running

## Step 1: Create a Spotify Developer Application

1. Go to [Spotify Developer Dashboard](https://developer.spotify.com/dashboard)
2. Log in with your Spotify account
3. Click **Create app**
4. Fill in the details:
   - **App name**: SteelClock (or any name you prefer)
   - **App description**: OLED display widget
   - **Redirect URI**: `http://127.0.0.1:8888/callback`
   - **APIs used**: Check "Web API"
5. Click **Save**
6. On the app page, click **Settings**
7. Copy the **Client ID** (you'll need this for the configuration)

**Important**: Make sure the Redirect URI is exactly `http://127.0.0.1:8888/callback` (or use a different port if you configure `callback_port` in the config).

## Step 2: Configure SteelClock

Edit your profile configuration (e.g., `spotify.json`) and add your Client ID:

```json
{
  "widgets": [
    {
      "type": "spotify",
      "position": { "x": 0, "y": 0, "w": 128, "h": 40 },
      "spotify_auth": {
        "mode": "oauth",
        "client_id": "YOUR_CLIENT_ID_HERE",
        "callback_port": 8888
      },
      "text": {
        "format": "{artist} - {title}"
      }
    }
  ]
}
```

Replace `YOUR_CLIENT_ID_HERE` with your actual Spotify Client ID.

## Step 3: First Run - Authorization

1. Start SteelClock with the Spotify profile
2. Your default web browser will open automatically
3. Log in to Spotify if prompted
4. Click **Agree** to authorize the application
5. The browser will show "Authorization successful!"
6. You can close the browser window

The widget will now display your currently playing track.

## Token Persistence

After successful authorization:
- Access tokens are saved to `spotify_token.json` in the application directory
- Tokens are automatically refreshed when they expire
- You won't need to re-authorize unless you revoke access or delete the token file

## Configuration Options

### Authentication (`spotify_auth`)

| Option          | Type   | Default                | Description                                                        |
|-----------------|--------|------------------------|--------------------------------------------------------------------|
| `mode`          | string | `"oauth"`              | `"oauth"` for interactive flow, `"manual"` for pre-obtained tokens |
| `client_id`     | string | required               | Your Spotify application Client ID                                 |
| `callback_port` | int    | `8888`                 | Local port for OAuth callback server                               |
| `token_path`    | string | `"spotify_token.json"` | Path to token storage file                                         |
| `access_token`  | string | -                      | Pre-obtained access token (manual mode only)                       |
| `refresh_token` | string | -                      | Pre-obtained refresh token (manual mode only)                      |

### Display (`spotify`)

| Option             | Type   | Default           | Description                                                    |
|--------------------|--------|-------------------|----------------------------------------------------------------|
| `placeholder.mode` | string | `"text"`          | What to show when not playing: `"text"`, `"icon"`, or `"hide"` |
| `placeholder.text` | string | `"[Not playing]"` | Text for placeholder mode                                      |

### Auto-Show Events (`spotify_auto_show`)

| Option            | Type  | Default | Description                       |
|-------------------|-------|---------|-----------------------------------|
| `on_track_change` | bool  | `true`  | Show widget when track changes    |
| `on_play`         | bool  | `false` | Show widget when playback starts  |
| `on_pause`        | bool  | `false` | Show widget when playback pauses  |
| `on_stop`         | bool  | `false` | Show widget when playback stops   |
| `duration_sec`    | float | `5`     | How long to show widget (seconds) |

### Text Format (`text.format`)

Available placeholders:

| Placeholder  | Description                   | Example                    |
|--------------|-------------------------------|----------------------------|
| `{artist}`   | First artist name             | "Taylor Swift"             |
| `{artists}`  | All artists (comma-separated) | "Taylor Swift, Ed Sheeran" |
| `{title}`    | Track name                    | "Anti-Hero"                |
| `{album}`    | Album name                    | "Midnights"                |
| `{position}` | Current position (MM:SS)      | "02:34"                    |
| `{duration}` | Track duration (MM:SS)        | "03:21"                    |
| `{state}`    | Playback state                | "Playing"                  |
| `{device}`   | Active device name            | "Desktop"                  |
| `{volume}`   | Volume percentage             | "75"                       |

### Text Scrolling (`scroll`)

| Option      | Type   | Default        | Description                                   |
|-------------|--------|----------------|-----------------------------------------------|
| `enabled`   | bool   | `false`        | Enable text scrolling                         |
| `direction` | string | `"left"`       | Scroll direction: `"left"`, `"right"`         |
| `speed`     | float  | `30`           | Scroll speed in pixels per second             |
| `mode`      | string | `"continuous"` | `"continuous"`, `"bounce"`, or `"pause_ends"` |
| `pause_ms`  | int    | `1000`         | Pause duration at ends (ms)                   |
| `gap`       | int    | `20`           | Gap between text repetitions (px)             |

## Manual Token Mode

If you prefer not to use the interactive OAuth flow, you can obtain tokens manually:

1. Use the [Spotify Web API Console](https://developer.spotify.com/console/) or a tool like [spotify-token-gen](https://github.com/spotify/web-api-auth-examples)
2. Request tokens with the `user-read-currently-playing` scope
3. Configure manual mode:

```json
{
  "spotify_auth": {
    "mode": "manual",
    "client_id": "YOUR_CLIENT_ID",
    "access_token": "YOUR_ACCESS_TOKEN",
    "refresh_token": "YOUR_REFRESH_TOKEN"
  }
}
```

Note: In manual mode, tokens will still be automatically refreshed when they expire.

## Troubleshooting

### "Missing required parameter: redirect_uri"

- Verify the Redirect URI in your Spotify Developer Dashboard matches **exactly**: `http://127.0.0.1:8888/callback`
- If you configured a custom `callback_port`, the URI must use that port (e.g., `http://127.0.0.1:9999/callback` for port 9999)
- Use `127.0.0.1`, not `localhost` -- Spotify treats them as different URIs

### "Auth required - check browser"

- Make sure you completed the OAuth flow in your browser
- Check that your browser opened and you authorized the application
- Verify your Client ID is correct

### Browser doesn't open automatically

- Check the log file for the authorization URL
- Copy and paste the URL manually into your browser

### "Token refresh failed"

- Your refresh token may have been revoked
- Delete `spotify_token.json` and re-authorize

### No track information displayed

- Make sure Spotify is playing on one of your devices
- The Spotify app must be running (desktop, web, or mobile)
- Check that the correct Client ID is configured

### Widget shows "[Not playing]"

- No active Spotify playback was detected
- Start playing music on any Spotify device
- The widget updates every 1 second (configurable via `update_interval`)

## Privacy & Security

- Your Spotify credentials are never stored by SteelClock
- Only OAuth tokens are stored locally
- The `user-read-currently-playing` scope only allows reading current playback
- No personal data or listening history is accessed
- Tokens can be revoked at any time from your [Spotify account settings](https://www.spotify.com/account/apps/)

## Example Configurations

### Minimal (just artist and title)

```json
{
  "type": "spotify",
  "position": { "x": 0, "y": 0, "w": 128, "h": 40 },
  "spotify_auth": {
    "client_id": "YOUR_CLIENT_ID"
  },
  "text": {
    "format": "{artist} - {title}"
  }
}
```

### With progress

```json
{
  "type": "spotify",
  "position": { "x": 0, "y": 0, "w": 128, "h": 40 },
  "spotify_auth": {
    "client_id": "YOUR_CLIENT_ID"
  },
  "text": {
    "format": "{artist} - {title} [{position}/{duration}]"
  }
}
```

### Auto-hide with scrolling

```json
{
  "type": "spotify",
  "position": { "x": 0, "y": 0, "w": 128, "h": 40 },
  "spotify_auth": {
    "client_id": "YOUR_CLIENT_ID"
  },
  "text": {
    "format": "{artist} - {title}"
  },
  "auto_hide": {
    "enabled": true,
    "timeout": 10
  },
  "spotify_auto_show": {
    "on_track_change": true,
    "on_play": true
  },
  "scroll": {
    "enabled": true,
    "mode": "pause_ends",
    "speed": 30
  }
}
```
