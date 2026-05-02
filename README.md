# Rainbow Tool

A sophisticated Windows desktop weather application built with Go that predicts rainbow probabilities and provides realistic weather visualizations.

## Features

### Core Weather
- **Current Weather**: Temperature, feels-like, humidity, wind, pressure, UV index
- **7-Day Forecast**: Daily highs/lows with rainbow probability predictions
- **Hourly Forecast**: 24-hour breakdown with temperature, weather icons, sun angles, and rainbow scores

### Smart Rainbow Prediction
- **Dynamic Scoring** (0-98%):
  - Sun elevation angle (optimal 5°-35°)
  - Precipitation probability (30-70% ideal)
  - Cloud cover analysis (30-70% optimal)
  - Temperature verification (above freezing)
- **Real-time Visualization**: Animated rainbow rendering when conditions align
- **Verification System**: Mark and save confirmed rainbow sightings with location data

### Realistic Weather Visualization
- **Dynamic Sky**: Smooth gradients based on time of day and weather
- **Animated Weather Effects**:
  - Rain with wind-driven angle
  - Snow with drifting/fluttering effects
  - Hail with bouncing pellets
  - Thunderstorms with lightning flashes
  - Fog with drifting mist
  - Blizzard conditions with heavy snow
- **Cloud Systems**: Animated clouds with wind-driven movement
- **Celestial Bodies**: Sun with pulsing glow, moon with craters, stars at night

### Location Support
- **Search**: Find any city worldwide with auto-complete suggestions
- **Manual Entry**: Enter custom latitude/longitude coordinates
- **Auto-Detect**: One-click location detection via IP geolocation
- **Persistent Storage**: Saved rainbow verifications retain location data

### Data Management
- **Rainbow Verification**: Mark confirmed rainbows with timestamp and location
- **Configuration File**: Saves color schemes and verification history
- **Detailed Logging**: Automatic weather data logging to local files
- **Color Customization**: Modern flat/neubrutalism color scheme

## Weather Classification System

The application intelligently classifies weather conditions for accurate visualization:

| Condition | Visual Effects |
|-----------|----------------|
| Clear Sky | Sun/Moon with optimal visibility |
| Fog | Layered drifting fog effects |
| Drizzle | Light, fine mist particles |
| Rain | Dynamic angled raindrops |
| Heavy Rain | Increased droplet density |
| Snow | Fluttering snowflakes |
| Blizzard | High-density snow with drift |
| Hail | Bouncing pellet effects |
| Thunderstorm | Sky flashes, lightning branches |
| Extreme Heat | Desert ground textures |
| Freezing | Snow-covered ground |

## Installation

### Prerequisites
- Windows operating system (7/8/10/11)
- Internet connection for weather data

### Download
Download the latest `rainbowtool.exe` from the [Releases](https://github.com/2dprototype/WeatherPro/releases) page.

### Build from Source

```bash
# Clone the repository
git clone https://github.com/2dprototype/RainbowTool.git
cd RainbowTool

# Install dependencies
go mod tidy

# Build with Windows GUI (no console window)
go build -ldflags="-H windowsgui" -o rainbowtool.exe

# Or build with console for debugging
go build -o rainbowtool.exe
```

### Command Line Usage

Rainbow Tool supports CLI commands for quick weather checks:

```bash
# Check weather for a specific city
rainbowtool "New York"

# Open in Quick Mode (mini window)
rainbowtool quick
```

### Quick Mode
Quick Mode opens a compact 340x240 window showing only the vital information:
- Current Temperature
- Rainbow Probability %
- Animated Weather Visualizer

### Quick Start

1. Launch `rainbowtool.exe` for the full experience.
2. Run `rainbowtool quick` for the mini visualizer.
3. The app will auto-detect your location via IP.
4. View current weather and rainbow predictions.
5. Use the interface:

```
[🌈 Rainbow Tool]  [Lat: 40.71] [Lon: -74.00] [Go]  [Search...] [🔍] [📍]
```

### Finding Locations

- **Search**: Type city name → Select from dropdown → Auto-fills lat/lon
- **Manual**: Enter coordinates directly → Click "Go"
- **Auto**: Click 📍 to detect your current location

### Reading Predictions

The main canvas shows the most promising rainbow window with:
- Current time and weather conditions
- Animated weather effects
- Rainbow arc when probability > 10% and sun angle > 0°

### Verifying Rainbows

1. Click any hour in the hourly table
2. Check "Verified Rainbow" if you actually saw a rainbow
3. Verification saves with location data for future reference

### Location Indicators

- ✅ in hourly table = Verified rainbow at that time/location
- Rainbow score % = Mathematical probability based on conditions

## Configuration

The app creates `rainbowtool.json` in the same directory:

```json
{
  "colors": {
    "sky_day_day": [135, 206, 235],
    "sky_night": [5, 5, 20],
    ...
  },
  "verifications": {
    "1704182400": true
  }
}
```

## Logging

Weather data and verifications are automatically logged to:

```
%EXEDIR%/rainbowtool/YYYY-MM-DD.log
```

Example log entry:
```
2024/01/15 14:30:45 Location: New York, US (40.7128, -74.0060)
2024/01/15 14:30:45 Temperature: 22.5°C, Humidity: 65%, Weather Code: 3
2024/01/15 14:30:45 Wind: 15.2 km/h, Pressure: 1013.2 hPa, UV: 4.5
```

## Rainbow Formula Details

The prediction algorithm uses weighted scoring:

| Factor | Max Score | Optimal Range |
|--------|-----------|---------------|
| Sun Elevation | 30 | 5°-35° (peaks at 20°) |
| Precipitation | 40 | 30-70% chance |
| Cloud Cover | 30 | 30-70% coverage |

**Penalties:**
- Cloud cover >95%: 90% reduction
- Precipitation <10%: 90% reduction
- Temperature <0°C: Impossible (snow instead of rain)

**Scale:**
- 90-98%: Excellent chance
- 70-89%: Good chance  
- 40-69%: Possible
- 10-39%: Unlikely
- <10%: Not displayed

## Data Sources

- **Weather Data**: [Open-Meteo API](https://open-meteo.com/) - Free, no API key required
- **Geocoding**: [Open-Meteo Geocoding API](https://open-meteo.com/en/docs/geocoding-api)
- **Location Detection**: [ip-api.com](http://ip-api.com/) - No API key required