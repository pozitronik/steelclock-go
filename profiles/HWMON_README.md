# Hardware Monitor Widget (hwmon)

The **hwmon** widget displays hardware sensor data from [LibreHardwareMonitor](https://github.com/LibreHardwareMonitor/LibreHardwareMonitor) (LHM) or [Open Hardware Monitor](https://github.com/openhardwaremonitor/openhardwaremonitor) (OHM) on your SteelSeries OLED screen. It can show temperatures, voltages, fan speeds, clock frequencies, power consumption, load percentages, and any other sensor these tools expose.

## Requirements

- **Windows** with LibreHardwareMonitor or Open Hardware Monitor installed
- The application's **web server** must be enabled (see setup below)

## Setup

### LibreHardwareMonitor

1. Download and run [LibreHardwareMonitor](https://github.com/LibreHardwareMonitor/LibreHardwareMonitor/releases)
2. Run it **as Administrator** (required for full sensor access)
3. Enable the web server: **Options > Remote Web Server > Run**
4. The default port is **8085**. You can verify it works by opening `http://localhost:8085` in your browser

To start LHM automatically with Windows, enable **Options > Run On Windows Startup**.

### Open Hardware Monitor

1. Download and run [Open Hardware Monitor](https://openhardwaremonitor.org/downloads/)
2. Run it **as Administrator**
3. Enable the web server: **Options > Remote Web Server > Run**
4. Default port is also **8085**

### Verifying the Connection

Open `http://localhost:8085/data.json` in your browser. You should see a JSON tree with your hardware sensors. If you see an error or empty page, make sure the application is running with the web server enabled.

## Finding Sensor IDs and Types

To configure the widget, you need to know what sensors are available. Open `http://localhost:8085/data.json` and look for sensor entries. Each sensor has:

- **SensorId** - a unique path like `/amdcpu/0/temperature/2` or `/gpu-nvidia/0/load/0`
- **Text** - a human-readable name like "Core (Tctl/Tdie)" or "GPU Core"
- **Type** - the sensor category: Temperature, Load, Voltage, Power, Clock, Fan, Throughput, etc.
- **Value** - the current reading like "65,0 °C" or "15,0 %"

### Common Sensor Types

| Type        | Description           | Typical Unit | Suggested Max |
|-------------|-----------------------|--------------|---------------|
| Temperature | Hardware temperatures | °C           | 100           |
| Load        | Usage percentages     | %            | 100           |
| Voltage     | Voltage readings      | V            | varies        |
| Power       | Power consumption     | W            | varies        |
| Clock       | Clock frequencies     | MHz          | varies        |
| Fan         | Fan speeds            | RPM          | varies        |

## Configuration

### Basic Example

A single widget showing CPU temperature as text:

```json
{
  "type": "hwmon",
  "enabled": true,
  "position": { "x": 0, "y": 0, "w": 128, "h": 40 },
  "mode": "text",
  "hwmon": {
    "sensor_type": "Temperature",
    "sensor_filter": "cpu"
  },
  "update_interval": 2
}
```

### hwmon Configuration Options

| Option          | Type   | Default                 | Description                                                                  |
|-----------------|--------|-------------------------|------------------------------------------------------------------------------|
| `url`           | string | `http://localhost:8085` | LHM/OHM web server URL. Change if using a non-default port or remote machine |
| `sensor_id`     | string | -                       | Exact sensor path (e.g., `/amdcpu/0/temperature/2`). Highest priority filter |
| `sensor_type`   | string | -                       | Filter by type: `Temperature`, `Load`, `Voltage`, `Power`, `Clock`, etc.     |
| `sensor_filter` | string | -                       | Case-insensitive substring match on sensor ID or display name                |
| `min`           | number | 0                       | Minimum value for normalization (bar/gauge/graph modes)                      |
| `max`           | number | 100                     | Maximum value for normalization                                              |

### Sensor Selection Priority

Filters are applied in this order:

1. **`sensor_id`** - if set, selects exactly one sensor. Other filters are ignored.
2. **`sensor_type` + `sensor_filter`** - if both set, sensors must match both criteria.
3. **`sensor_type`** alone - matches all sensors of that type.
4. **`sensor_filter`** alone - matches any sensor whose ID or name contains the substring.
5. **No filters** - returns all sensors from the server.

When multiple sensors match and `per_core` is not enabled, their values are **averaged**.

### Display Modes

The widget supports the same display modes as CPU/memory widgets:

| Mode    | Description                                   |
|---------|-----------------------------------------------|
| `text`  | Numeric value with unit (e.g., "65°C", "15%") |
| `bar`   | Horizontal or vertical progress bar           |
| `graph` | Scrolling history graph                       |
| `gauge` | Circular gauge with needle                    |

### Text Formatting

In `text` mode, the unit determines the default format:

| Unit  | Format       | Example |
|-------|--------------|---------|
| °C    | `%.0f°C`     | 65°C    |
| %     | `%.0f%%`     | 15%     |
| W     | `%.1fW`      | 13.9W   |
| MHz   | `%.0fMHz`    | 3924MHz |
| V     | `%.2fV`      | 1.05V   |
| other | `%.1f<unit>` | 42.5GB  |

You can override the format using the `text.format` field. This is a printf-style format string with one `%f` verb for the sensor value. You can add prefixes and suffixes:

```json
"text": {
  "format": "CPU: %.0f°C"
}
```

This would display "CPU: 65°C" instead of just "65°C".

### Normalization (min/max)

For bar, gauge, and graph modes, the raw sensor value is normalized to a 0-100% scale using the `min` and `max` settings:

```
normalized = ((value - min) / (max - min)) * 100
```

This is useful for sensors with non-zero baselines. For example, to display CPU clock speed where idle is 800 MHz and boost is 5000 MHz:

```json
"hwmon": {
  "sensor_type": "Clock",
  "sensor_filter": "cpu",
  "min": 800,
  "max": 5000
}
```

### Per-Sensor Grid Mode

Enable `per_core` to display each matching sensor in its own grid cell (similar to the CPU widget's per-core mode):

```json
{
  "type": "hwmon",
  "position": { "x": 0, "y": 0, "w": 128, "h": 40 },
  "mode": "bar",
  "hwmon": {
    "sensor_type": "Temperature"
  },
  "per_core": {
    "enabled": true,
    "border": false,
    "margin": 1
  }
}
```

This shows all temperature sensors as individual bars in an auto-sized grid.

## Configuration Examples

### CPU and GPU Dashboard (Default Profile)

Four text widgets showing CPU/GPU temperature and load:

```json
{
  "widgets": [
    {
      "type": "hwmon",
      "position": { "x": 0, "y": 0, "w": 64, "h": 20 },
      "mode": "text",
      "hwmon": { "sensor_type": "Temperature", "sensor_filter": "cpu" },
      "update_interval": 2
    },
    {
      "type": "hwmon",
      "position": { "x": 64, "y": 0, "w": 64, "h": 20 },
      "mode": "text",
      "hwmon": { "sensor_type": "Temperature", "sensor_filter": "gpu" },
      "update_interval": 2
    },
    {
      "type": "hwmon",
      "position": { "x": 0, "y": 20, "w": 64, "h": 20 },
      "mode": "text",
      "hwmon": { "sensor_type": "Load", "sensor_filter": "CPU Total" },
      "update_interval": 1
    },
    {
      "type": "hwmon",
      "position": { "x": 64, "y": 20, "w": 64, "h": 20 },
      "mode": "text",
      "hwmon": { "sensor_type": "Load", "sensor_filter": "GPU Core" },
      "update_interval": 1
    }
  ]
}
```

### GPU Power and Temperature Gauges

```json
{
  "widgets": [
    {
      "type": "hwmon",
      "position": { "x": 0, "y": 0, "w": 64, "h": 40 },
      "mode": "gauge",
      "hwmon": {
        "sensor_type": "Temperature",
        "sensor_filter": "gpu",
        "max": 90
      },
      "update_interval": 2
    },
    {
      "type": "hwmon",
      "position": { "x": 64, "y": 0, "w": 64, "h": 40 },
      "mode": "gauge",
      "hwmon": {
        "sensor_type": "Power",
        "sensor_filter": "gpu",
        "max": 350
      },
      "update_interval": 2
    }
  ]
}
```

### CPU Load History Graph

```json
{
  "type": "hwmon",
  "position": { "x": 0, "y": 0, "w": 128, "h": 40 },
  "mode": "graph",
  "hwmon": {
    "sensor_type": "Load",
    "sensor_filter": "CPU Total"
  },
  "graph": { "history": 128 },
  "update_interval": 1
}
```

### Exact Sensor by ID

If filtering gives unexpected results, use the exact sensor path from `/data.json`:

```json
"hwmon": {
  "sensor_id": "/amdcpu/0/temperature/2"
}
```

### Remote Machine Monitoring

Monitor a different machine running LHM with its web server accessible over the network:

```json
"hwmon": {
  "url": "http://192.168.1.100:8085",
  "sensor_type": "Temperature",
  "sensor_filter": "cpu"
}
```

## Troubleshooting

### Widget shows "No sensors"

1. **Is LHM/OHM running?** Check that the application is open and showing sensor readings.
2. **Is the web server enabled?** Open `http://localhost:8085` in your browser. You should see the sensor tree.
3. **Is LHM running as Administrator?** Some sensors require elevated privileges.
4. **Wrong URL or port?** If you changed the port in LHM, update the `url` field in the widget config.

### Widget shows a value but it looks wrong

1. **Check your filters.** Open `http://localhost:8085/data.json` and search for the sensor you want. Verify the `sensor_type` and `sensor_filter` match.
2. **Multiple sensors matching.** When several sensors match your filter, their values are averaged. Use `sensor_id` for an exact match, or make `sensor_filter` more specific.
3. **Wrong min/max range.** For bar/gauge/graph modes, check that `min` and `max` cover the expected value range.

### LHM shows sensors but the widget does not

The widget fetches from the HTTP API, not from WMI. Make sure the **web server** is enabled in LHM, not just the WMI provider.

## Links

- [LibreHardwareMonitor](https://github.com/LibreHardwareMonitor/LibreHardwareMonitor) - recommended, actively maintained
- [Open Hardware Monitor](https://openhardwaremonitor.org/) - compatible alternative
- [CONFIG_GUIDE.md](CONFIG_GUIDE.md) - full SteelClock configuration reference
