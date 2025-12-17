# DOOM Widget - Running DOOM on Your SteelSeries Display

Yes, you can run DOOM on your SteelSeries OLED display! This widget uses the [Gore](https://github.com/AndreRenaud/gore) pure Go DOOM engine port.

https://github.com/user-attachments/assets/37cbdbb5-6179-4a45-ab52-74192a529420

## Features

- Runs DOOM shareware (doom1.wad) or full version
- **Automatically downloads the shareware WAD if not found!**
- **Visual progress bar on display during download**
- Automatically plays demos (no input needed)
- Downscales and converts to grayscale for your 128x40 monochrome display
- Pure demonstration mode - just sit back and watch DOOM run!

## Quick Start

**Just run it!** No WAD file needed - it will auto-download:

```bash
steelclock.exe -config configs/examples/doom.json
```

The widget will automatically download the free DOOM shareware WAD (`doom1.wad`) to the current directory on first run.

## Advanced Setup

### Using a Different WAD File

**Shareware (Free & Legal):**
- The shareware WAD will auto-download on first run
- Or manually download `doom1.wad` from: https://distro.ibiblio.org/slitaz/sources/packages/d/doom1.wad

**Full Version:**
- Use your purchased copy of DOOM (`doom.wad` or `doom2.wad`)
- Available on Steam, GOG, etc.
- Place in same directory as `steelclock.exe` and use just the filename in config

### WAD File Locations

The widget looks for WAD files in this order:
1. Exact path specified in config (relative to current directory)
2. Current directory
3. Auto-downloads to `doom1.wad` in current directory if not found

**Note:** Gore works best with WAD files in the current directory. Use relative paths like `doom1.wad` rather than absolute paths.

### Configure

Use the provided `doom_example.json` or add to your existing config:

```json
{
"type": "doom",
"enabled": true,
"position": {
"x": 0,
"y": 0,
"w": 128,
"h": 40
},
"properties": {
"wad_name": "doom1.wad"
}
}
```

### Custom Download URL (Optional)

You can specify a custom WAD download URL in the main config:

```json
{
"bundled_wad_url": "https://example.com/path/to/doom.wad",
...
}
```

The default URL points to the official DOOM shareware release.

## What You'll See

- Progress bar with percentage during WAD download (if downloading)
- DOOM running at 128x40 resolution in glorious grayscale
- Automatic demo playback cycling through levels
- Classic DOOM monsters, weapons, and action
- The satisfaction of running a legendary game on a tiny OLED display

## Technical Details

- **Resolution:** DOOM renders at 320x200, downscaled to your display size
- **Color:** RGBA converted to grayscale using standard luminance formula
- **Frame Rate:** Limited by your `refresh_rate_ms` config (50ms = 20 FPS recommended)
- **Input:** Currently demo mode only (no keyboard/mouse input)
- **Audio:** No sound (silent DOOM is still DOOM)

## Troubleshooting

**Widget doesn't appear:**
- Check that the WAD file path is correct
- Make sure the WAD file is not corrupted
- Check the application logs for errors

**Performance issues:**
- Increase `refresh_rate_ms` (e.g., 100ms for slower refresh)
- DOOM is CPU-intensive even at small resolutions

**WAD not found:**
- Use filename only: `"wad_name": "doom1.wad"` (place WAD in working directory)
- Do not include paths, only filename (e.g., "doom1.wad" not "path/doom1.wad")
- Ensure file has `.wad` extension
- Check internet connection (for auto-download)

## Why?

Because we can. And because DOOM runs on everything.

## Credits

- Original DOOM by id Software
- Gore engine by [Andre Renaud](https://github.com/AndreRenaud/gore)
- Integration by your truly dedicated SteelClock developers

---

*"Can it run DOOM?" - Yes. Yes it can.*
