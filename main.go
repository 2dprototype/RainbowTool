package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/2dprototype/wui"
)

// ---------------------------------------------------------------------
// Data structures for Open-Meteo API
// ---------------------------------------------------------------------
type WeatherResponse struct {
	Latitude           float64 `json:"latitude"`
	Longitude          float64 `json:"longitude"`
	UTC_Offset_Seconds int     `json:"utc_offset_seconds"`
	Current            Current `json:"current"`
	Hourly             Hourly  `json:"hourly"`
	Daily              Daily   `json:"daily"`
}

type Current struct {
	Time                string  `json:"time"`
	Temperature2m       float64 `json:"temperature_2m"`
	RelativeHumidity2m  float64 `json:"relative_humidity_2m"`
	ApparentTemperature float64 `json:"apparent_temperature"`
	Precipitation       float64 `json:"precipitation"`
	WeatherCode         int     `json:"weather_code"`
	CloudCover          float64 `json:"cloud_cover"`
	WindSpeed10m        float64 `json:"wind_speed_10m"`
	WindDirection10m    float64 `json:"wind_direction_10m"`
	UVIndex             float64 `json:"uv_index"`
	DewPoint2m          float64 `json:"dew_point_2m"`
	SurfacePressure     float64 `json:"surface_pressure"`
}

type Hourly struct {
	Time                []string  `json:"time"`
	PrecipitationProb   []float64 `json:"precipitation_probability"`
	CloudCover          []float64 `json:"cloud_cover"`
	Temperature2m       []float64 `json:"temperature_2m"`
	DewPoint2m          []float64 `json:"dew_point_2m"`
	WeatherCode         []int     `json:"weather_code"`
	RelativeHumidity2m  []float64 `json:"relative_humidity_2m"`
	WindSpeed10m        []float64 `json:"wind_speed_10m"`
}

type Daily struct {
	Time                 []string  `json:"time"`
	Temperature2mMax     []float64 `json:"temperature_2m_max"`
	Temperature2mMin     []float64 `json:"temperature_2m_min"`
	PrecipitationProbMax []float64 `json:"precipitation_probability_max"`
	WeatherCode          []int     `json:"weather_code"`
}

type GeocodingResult struct {
	Results []struct {
		Name      string  `json:"name"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		Country   string  `json:"country"`
		Admin1    string  `json:"admin1"`
	} `json:"results"`
}

type RGBColor [3]uint8

type AppConfig struct {
	Colors struct {
		SkyDayDay           RGBColor `json:"sky_day_day"`
		SkyNight            RGBColor `json:"sky_night"`
		SkySunset           RGBColor `json:"sky_sunset"`
		GroundGreen1        RGBColor `json:"ground_green_1"`
		GroundGreen2        RGBColor `json:"ground_green_2"`
		GroundSnow1         RGBColor `json:"ground_snow_1"`
		GroundSnow2         RGBColor `json:"ground_snow_2"`
		GroundDesert1       RGBColor `json:"ground_desert_1"`
		GroundDesert2       RGBColor `json:"ground_desert_2"`
		GroundStorm1        RGBColor `json:"ground_storm_1"`
		GroundStorm2        RGBColor `json:"ground_storm_2"`
		CloudWhite          RGBColor `json:"cloud_white"`
		CloudLightGray      RGBColor `json:"cloud_light_gray"`
		CloudDarkGray       RGBColor `json:"cloud_dark_gray"`
		CloudStorm          RGBColor `json:"cloud_storm"`
		RainDrop            RGBColor `json:"rain_drop"`
		SnowDrop            RGBColor `json:"snow_drop"`
		Lightning           RGBColor `json:"lightning"`
		WindLine            RGBColor `json:"wind_line"`
		SunInner            RGBColor `json:"sun_inner"`
		SunOuter            RGBColor `json:"sun_outer"`
		Moon                RGBColor `json:"moon"`
		MoonCrater          RGBColor `json:"moon_crater"`
		Star                RGBColor `json:"star"`
		CanvasBackground    RGBColor `json:"canvas_background"`
		RainbowColors       []RGBColor `json:"rainbow_colors"`
	} `json:"colors"`
	Verifications map[string]bool `json:"verifications"`
	ThemeName     string          `json:"theme_name"`
}

// ---------------------------------------------------------------------
// Global UI elements
// ---------------------------------------------------------------------
var (
	mainWindow *wui.Window

	// header
	searchEdit   *wui.EditLine
	searchCombo  *wui.ComboBox
	searchItems  []geocodingItem
	btnGeo       *wui.Button

	// current weather
	labelMainTemp  *wui.Label
	labelMainFeels *wui.Label
	labelMainIcon  *wui.Label
	labelLocation  *wui.Label
	labelDesc      *wui.Label
	labelDate      *wui.Label // Added to show selected date

	// feature cards
	featureLabels map[string]*wui.Label

	// hourly & daily tables
	hourlyTable *wui.StringTable
	dailyTable  *wui.StringTable

	// canvases
	mainCanvas  *wui.PaintBox
	colorCanvas *wui.PaintBox

	// state
	currentLat          float64 = 40.71
	currentLon          float64 = -74.00
	locationName                = "New York"
	countryName                 = "US"
	tempUnit            string  = "celsius"
	weatherCache        *WeatherResponse
	timezoneOffsetHours float64 = -4
	
	// Selected day data (for viewing past/future days)
	selectedDayData    *DailyDayData
	selectedHourlyData []HourlyData
	selectedDate       string
	selectedHourData   *HourlyData
	
	// logging
	logFile *os.File
	logger  *log.Logger

	// config
	appConfig  AppConfig
	configPath string

	// animation
	animFrame int64

	// new ui elements
	editLat       *wui.EditLine
	editLon       *wui.EditLine
	btnUpdateLoc  *wui.Button
	checkVerified *wui.CheckBox

	// themes
	themeCombo *wui.ComboBox
	themeItems []string
)

type geocodingItem struct {
	Name    string
	Lat     float64
	Lon     float64
	Country string
	Admin1  string
}

type DailyDayData struct {
	Date            time.Time
	TemperatureMax  float64
	TemperatureMin  float64
	PrecipitationMax float64
	WeatherCode     int
}

type HourlyData struct {
	Time               time.Time
	Temperature2m      float64
	WeatherCode        int
	PrecipitationProb  float64
	CloudCover         float64
	RelativeHumidity2m float64
	WindSpeed10m       float64
}

func initDefaultConfig() {
	appConfig = AppConfig{
		Verifications: make(map[string]bool),
	}
	appConfig.Colors.SkyDayDay = RGBColor{135, 206, 235}
	appConfig.Colors.SkyNight = RGBColor{5, 5, 20}
	appConfig.Colors.SkySunset = RGBColor{255, 140, 0}
	appConfig.Colors.GroundGreen1 = RGBColor{34, 139, 34}
	appConfig.Colors.GroundGreen2 = RGBColor{46, 139, 87}
	appConfig.Colors.GroundSnow1 = RGBColor{240, 248, 255}
	appConfig.Colors.GroundSnow2 = RGBColor{255, 250, 250}
	appConfig.Colors.GroundDesert1 = RGBColor{237, 201, 175}
	appConfig.Colors.GroundDesert2 = RGBColor{210, 180, 140}
	appConfig.Colors.GroundStorm1 = RGBColor{25, 100, 25}
	appConfig.Colors.GroundStorm2 = RGBColor{35, 100, 50}
	appConfig.Colors.CloudWhite = RGBColor{255, 255, 255}
	appConfig.Colors.CloudLightGray = RGBColor{200, 200, 200}
	appConfig.Colors.CloudDarkGray = RGBColor{100, 100, 100}
	appConfig.Colors.CloudStorm = RGBColor{50, 50, 60}
	appConfig.Colors.RainDrop = RGBColor{150, 150, 200}
	appConfig.Colors.SnowDrop = RGBColor{255, 255, 255}
	appConfig.Colors.Lightning = RGBColor{255, 255, 0}
	appConfig.Colors.WindLine = RGBColor{200, 200, 200}
	appConfig.Colors.SunInner = RGBColor{255, 255, 0}
	appConfig.Colors.SunOuter = RGBColor{255, 255, 150}
	appConfig.Colors.Moon = RGBColor{200, 200, 220}
	appConfig.Colors.MoonCrater = RGBColor{170, 170, 190}
	appConfig.Colors.Star = RGBColor{255, 255, 255}
	appConfig.Colors.CanvasBackground = RGBColor{200, 200, 200}
	appConfig.Colors.RainbowColors = []RGBColor{
		{255, 0, 0}, {255, 127, 0}, {255, 255, 0},
		{0, 255, 0}, {0, 0, 255}, {75, 0, 130}, {148, 0, 211},
	}
}

func loadConfig() {
	exePath, err := os.Executable()
	if err != nil {
		initDefaultConfig()
		return
	}
	
	exeDir := filepath.Dir(exePath)
	exeName := strings.TrimSuffix(filepath.Base(exePath), filepath.Ext(exePath))
	configPath = filepath.Join(exeDir, exeName+".json")
	
	file, err := os.Open(configPath)
	if err != nil {
		initDefaultConfig()
		saveConfig()
		return
	}
	defer file.Close()
	
	if err := json.NewDecoder(file).Decode(&appConfig); err != nil {
		initDefaultConfig()
	}
	
	if appConfig.Verifications == nil {
		appConfig.Verifications = make(map[string]bool)
	}
	if len(appConfig.Colors.RainbowColors) == 0 {
		initDefaultConfig()
		saveConfig()
	}
}

func saveConfig() {
	if configPath == "" {
		return
	}
	
	data, err := json.MarshalIndent(appConfig, "", "  ")
	if err != nil {
		return
	}
	
	os.WriteFile(configPath, data, 0644)
}

func wuiColor(c RGBColor) wui.Color {
	return wui.RGB(c[0], c[1], c[2])
}

// ---------------------------------------------------------------------
// Logging setup
// ---------------------------------------------------------------------
func setupLogging() error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	
	exeDir := filepath.Dir(exePath)
	exeName := strings.TrimSuffix(filepath.Base(exePath), filepath.Ext(exePath))
	
	logDir := filepath.Join(exeDir, exeName)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}
	
	today := time.Now().Format("2006-01-02")
	logPath := filepath.Join(logDir, today+".log")
	
	logFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	
	logger = log.New(logFile, "", log.LstdFlags)
	logger.Println("=== Weather App Started ===")
	return nil
}

func logWeatherData(w *WeatherResponse, lat, lon float64, location string) {
	if logger == nil {
		return
	}
	
	logger.Printf("Location: %s (%.4f, %.4f)", location, lat, lon)
	if w != nil {
		logger.Printf("Temperature: %.1f°C, Humidity: %.1f%%, Weather Code: %d", 
			w.Current.Temperature2m, w.Current.RelativeHumidity2m, w.Current.WeatherCode)
		logger.Printf("Wind: %.1f km/h, Pressure: %.1f hPa, UV: %.1f", 
			w.Current.WindSpeed10m, w.Current.SurfacePressure, w.Current.UVIndex)
	}
	logger.Println("---")
}

// ---------------------------------------------------------------------
// Weather helpers
// ---------------------------------------------------------------------
func weatherInfo(code int) (icon string, desc string) {
	switch code {
	case 0:
		return "☀️", "Clear sky"
	case 1:
		return "☀️", "Mainly clear"
	case 2:
		return "⛅", "Partly cloudy"
	case 3:
		return "☁️", "Overcast"
	case 45, 48:
		return "🌫️", "Fog"
	case 51:
		return "🌧️", "Light drizzle"
	case 53:
		return "🌧️", "Moderate drizzle"
	case 55:
		return "🌧️", "Dense drizzle"
	case 61:
		return "🌦️", "Slight rain"
	case 63:
		return "🌧️", "Moderate rain"
	case 65:
		return "🌧️", "Heavy rain"
	case 71, 73, 75:
		return "❄️", "Snow"
	case 80:
		return "🌦️", "Rain showers"
	case 81:
		return "🌧️", "Moderate showers"
	case 82:
		return "🌧️", "Violent showers"
	case 95:
		return "⛈️", "Thunderstorm"
	case 96, 99:
		return "⛈️", "T-storm w/ hail"
	default:
		return "☁️", "Unknown"
	}
}

func formatTemp(c float64) string {
	if tempUnit == "fahrenheit" {
		return fmt.Sprintf("%.0f°F", c*9/5+32)
	}
	return fmt.Sprintf("%.0f°C", c)
}

// Solar elevation in degrees given UTC date-time
func getSolarElevationUTC(lat, lon float64, utc time.Time) float64 {
	dayOfYear := float64(utc.YearDay())
	declination := 23.44 * math.Sin((2*math.Pi/365)*(dayOfYear-81))
	declRad := declination * math.Pi / 180
	latRad := lat * math.Pi / 180
	hourUTC := float64(utc.Hour()) + float64(utc.Minute())/60
	solarNoonUTC := 12 - (lon / 15)
	hourAngle := (hourUTC - solarNoonUTC) * 15 * math.Pi / 180
	sinAlt := math.Sin(latRad)*math.Sin(declRad) + math.Cos(latRad)*math.Cos(declRad)*math.Cos(hourAngle)
	return math.Asin(math.Max(-1, math.Min(1, sinAlt))) * 180 / math.Pi
}

type rainbowPred struct {
	Time       time.Time
	Score      int
	SunElev    float64
	PrecipProb float64
}

func predictRainbow(hourlyData []HourlyData, lat, lon float64, tzOffsetHours float64, includePast bool) []rainbowPred {
	var results []rainbowPred
	now := time.Now()
	
	for i := 0; i < len(hourlyData) && i < 48; i++ {
		localTime := hourlyData[i].Time
		
		// Skip future times if we only want past, or skip past if includePast is false
		if !includePast && localTime.Before(now) {
			continue
		}
		
		utcTime := localTime.Add(-time.Duration(tzOffsetHours) * time.Hour)
		sunElev := getSolarElevationUTC(lat, lon, utcTime)
		
		// Rainbows are physically impossible if the sun is below the horizon or above 42°
		if sunElev <= 0 || sunElev >= 42 {
			continue
		}

		precipProb := hourlyData[i].PrecipitationProb
		cloud := hourlyData[i].CloudCover

		score := 0.0

		// 1. Sun Elevation Score (Max 30)
		if sunElev > 5 && sunElev < 35 {
			score += 30.0 - math.Abs(sunElev-20)*0.5 // peaks near 20 degrees
		} else {
			score += 15.0
		}

		// 2. Precipitation Score (Max 40)
		if precipProb >= 30 && precipProb <= 70 {
			score += 40.0
		} else if precipProb > 70 {
			score += 30.0
		} else {
			score += precipProb * 0.5
		}

		// 3. Cloud Cover Score (Max 30)
		if cloud >= 30 && cloud <= 70 {
			score += 30.0
		} else if cloud > 70 && cloud < 90 {
			score += 15.0
		} else if cloud >= 90 {
			score += 0.0
		} else {
			score += cloud * 0.5
		}

		// Strict Multipliers (Penalties for impossible combinations)
		if cloud > 95 {
			score *= 0.1 // Sun is completely blocked
		}
		if precipProb < 10 {
			score *= 0.1 // Not enough moisture
		}
		if hourlyData[i].Temperature2m < 0 {
			score *= 0.0 // Snow instead of rain
		}

		finalScore := int(math.Min(98, math.Round(score)))
		if finalScore > 10 {
			results = append(results, rainbowPred{localTime, finalScore, math.Round(sunElev), precipProb})
		}
	}
	
	sort.Slice(results, func(i, j int) bool { return results[i].Score > results[j].Score })
	return results
}

// ---------------------------------------------------------------------
// API helpers (async versions)
// ---------------------------------------------------------------------
func fetchWeatherAsync(lat, lon float64, callback func(*WeatherResponse, error)) {
	go func() {
		url := fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=%f&longitude=%f&current=temperature_2m,relative_humidity_2m,apparent_temperature,precipitation,weather_code,cloud_cover,wind_speed_10m,wind_direction_10m,uv_index,dew_point_2m,surface_pressure&hourly=precipitation_probability,cloud_cover,temperature_2m,dew_point_2m,weather_code,relative_humidity_2m,wind_speed_10m&daily=temperature_2m_max,temperature_2m_min,precipitation_probability_max,weather_code&timezone=auto&forecast_days=7&past_days=7",
			lat, lon)
		
		resp, err := http.Get(url)
		if err != nil {
			callback(nil, err)
			return
		}
		defer resp.Body.Close()
		
		var w WeatherResponse
		if err := json.NewDecoder(resp.Body).Decode(&w); err != nil {
			callback(nil, err)
			return
		}
		
		logWeatherData(&w, lat, lon, fmt.Sprintf("%.4f,%.4f", lat, lon))
		callback(&w, nil)
	}()
}

func searchCityAsync(q string, callback func([]geocodingItem, error)) {
	go func() {
		url := fmt.Sprintf("https://geocoding-api.open-meteo.com/v1/search?name=%s&count=5", q)
		resp, err := http.Get(url)
		if err != nil {
			callback(nil, err)
			return
		}
		defer resp.Body.Close()
		
		var res GeocodingResult
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			callback(nil, err)
			return
		}
		
		items := make([]geocodingItem, len(res.Results))
		for i, r := range res.Results {
			items[i] = geocodingItem{r.Name, r.Latitude, r.Longitude, r.Country, r.Admin1}
		}
		callback(items, nil)
	}()
}

func getUserLocationAsync(callback func(float64, float64, string, string, error)) {
	go func() {
		resp, err := http.Get("http://ip-api.com/json/?fields=lat,lon,city,country")
		if err != nil {
			callback(0, 0, "", "", err)
			return
		}
		defer resp.Body.Close()
		
		var data struct {
			Lat     float64 `json:"lat"`
			Lon     float64 `json:"lon"`
			City    string  `json:"city"`
			Country string  `json:"country"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			callback(0, 0, "", "", err)
			return
		}
		callback(data.Lat, data.Lon, data.City, data.Country, nil)
	}()
}

func reverseGeocodeAsync(lat, lon float64, callback func(string, error)) {
	go func() {
		client := &http.Client{}
		url := fmt.Sprintf("https://nominatim.openstreetmap.org/reverse?format=json&lat=%f&lon=%f", lat, lon)
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("User-Agent", "RainbowTool/1.0")

		resp, err := client.Do(req)
		if err != nil {
			callback("", err)
			return
		}
		defer resp.Body.Close()

		var data struct {
			DisplayName string `json:"display_name"`
			Address     struct {
				City    string `json:"city"`
				Town    string `json:"town"`
				Village string `json:"village"`
				Country string `json:"country"`
			} `json:"address"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			callback("", err)
			return
		}

		name := data.Address.City
		if name == "" {
			name = data.Address.Town
		}
		if name == "" {
			name = data.Address.Village
		}
		if name == "" {
			name = data.DisplayName
		}
		callback(name, nil)
	}()
}

// ---------------------------------------------------------------------
// Data extraction helpers
// ---------------------------------------------------------------------
func extractDailyData(w *WeatherResponse) []DailyDayData {
	if w == nil {
		return nil
	}
	
	var dailyData []DailyDayData
	for i, t := range w.Daily.Time {
		date, _ := time.Parse("2006-01-02", t)
		dailyData = append(dailyData, DailyDayData{
			Date:             date,
			TemperatureMax:   w.Daily.Temperature2mMax[i],
			TemperatureMin:   w.Daily.Temperature2mMin[i],
			PrecipitationMax: w.Daily.PrecipitationProbMax[i],
			WeatherCode:      w.Daily.WeatherCode[i],
		})
	}
	return dailyData
}

func extractHourlyDataForDay(w *WeatherResponse, targetDate time.Time) []HourlyData {
	if w == nil {
		return nil
	}
	
	var hourlyData []HourlyData
	targetDateStr := targetDate.Format("2006-01-02")
	
	for i, t := range w.Hourly.Time {
		hourTime, err := time.Parse("2006-01-02T15:04", t)
		if err != nil {
			continue
		}
		
		if hourTime.Format("2006-01-02") == targetDateStr {
			hourlyData = append(hourlyData, HourlyData{
				Time:               hourTime,
				Temperature2m:      w.Hourly.Temperature2m[i],
				WeatherCode:        w.Hourly.WeatherCode[i],
				PrecipitationProb:  w.Hourly.PrecipitationProb[i],
				CloudCover:         w.Hourly.CloudCover[i],
				RelativeHumidity2m: w.Hourly.RelativeHumidity2m[i],
				WindSpeed10m:       w.Hourly.WindSpeed10m[i],
			})
		}
	}
	return hourlyData
}

// ---------------------------------------------------------------------
// UI update functions
// ---------------------------------------------------------------------
func setLabelText(l *wui.Label, text string) {
	if l != nil {
		l.SetText(text)
	}
}

func updateCurrentWeather() {
	if selectedDayData != nil && selectedDate != "today" {
		// Show selected day's data (not current)
		icon, desc := weatherInfo(selectedDayData.WeatherCode)
		setLabelText(labelMainTemp, formatTemp(selectedDayData.TemperatureMax))
		setLabelText(labelMainFeels, fmt.Sprintf("High / Low: %s / %s", formatTemp(selectedDayData.TemperatureMax), formatTemp(selectedDayData.TemperatureMin)))
		setLabelText(labelMainIcon, icon)
		setLabelText(labelDesc, desc)
		setLabelText(labelDate, selectedDayData.Date.Format("Mon, Jan 2, 2006"))
	} else if weatherCache != nil {
		// Show current weather
		c := weatherCache.Current
		icon, desc := weatherInfo(c.WeatherCode)
		setLabelText(labelMainTemp, formatTemp(c.Temperature2m))
		setLabelText(labelMainFeels, fmt.Sprintf("Feels %s", formatTemp(c.ApparentTemperature)))
		setLabelText(labelMainIcon, icon)
		setLabelText(labelDesc, desc)
		setLabelText(labelDate, time.Now().Format("Mon, Jan 2, 2006"))
	}
	
	loc := locationName
	if countryName != "" {
		loc += ", " + countryName
	}
	setLabelText(labelLocation, loc)
}

func updateFeatures() {
	if weatherCache == nil {
		return
	}
	
	// Use selected day's hourly data or current
	var hourlyData []HourlyData
	if selectedHourlyData != nil && len(selectedHourlyData) > 0 {
		hourlyData = selectedHourlyData
	} else {
		hourlyData = extractHourlyDataForDay(weatherCache, time.Now())
	}
	
	c := weatherCache.Current
	dailyData := extractDailyData(weatherCache)
	
	lat := weatherCache.Latitude
	lon := weatherCache.Longitude
	tzOff := float64(weatherCache.UTC_Offset_Seconds) / 3600

	includePast := (selectedDayData != nil && selectedDate != "today")
	rainbowPreds := predictRainbow(hourlyData, lat, lon, tzOff, includePast)
	bestRainbow := "None"
	bestRainbowDetail := "No rain/sun angle"
	bestRainbowBadge := "Unlikely"
	if len(rainbowPreds) > 0 {
		rp := rainbowPreds[0]
		bestRainbow = fmt.Sprintf("%d%%", rp.Score)
		bestRainbowDetail = fmt.Sprintf("Best %s · sun %.0f°", rp.Time.Format("3:04 PM"), rp.SunElev)
		bestRainbowBadge = "View window"
	}

	// Rainbow-specific metrics
	setLabelText(featureLabels["rainbow"], bestRainbow)
	setLabelText(featureLabels["rainbowDetail"], bestRainbowDetail)
	setLabelText(featureLabels["rainbowBadge"], bestRainbowBadge)

	sunElevVal := "--"
	sunElevDetail := "Night"
	if len(rainbowPreds) > 0 {
		sunElevVal = fmt.Sprintf("%.1f°", rainbowPreds[0].SunElev)
		if rainbowPreds[0].SunElev > 42 {
			sunElevDetail = "Too high (>42°)"
		} else if rainbowPreds[0].SunElev <= 0 {
			sunElevDetail = "Below horizon"
		} else {
			sunElevDetail = "Optimal angle"
		}
	}
	setLabelText(featureLabels["sun"], sunElevVal)
	setLabelText(featureLabels["sunDetail"], sunElevDetail)
	setLabelText(featureLabels["sunBadge"], "Angle")

	precipProb := c.Precipitation
	if precipProb == 0 && len(dailyData) > 0 {
		precipProb = dailyData[0].PrecipitationMax
	}
	setLabelText(featureLabels["precip"], fmt.Sprintf("%.0f%%", precipProb))
	setLabelText(featureLabels["precipDetail"], "Required for bows")
	setLabelText(featureLabels["precipBadge"], "Moisture")

	setLabelText(featureLabels["cloud"], fmt.Sprintf("%.0f%%", c.CloudCover))
	setLabelText(featureLabels["cloudDetail"], "Need < 95%")
	setLabelText(featureLabels["cloudBadge"], "Blockage")

	if mainCanvas != nil {
		mainCanvas.Paint()
	}
	if colorCanvas != nil {
		colorCanvas.Paint()
	}
}

func updateHourlyTable() {
	if weatherCache == nil || hourlyTable == nil {
		return
	}
	
	// Use selected day's hourly data or current day's data
	var hourlyData []HourlyData
	if selectedHourlyData != nil && len(selectedHourlyData) > 0 {
		hourlyData = selectedHourlyData
	} else {
		hourlyData = extractHourlyDataForDay(weatherCache, time.Now())
	}
	
	tzOff := float64(weatherCache.UTC_Offset_Seconds) / 3600
	lat := weatherCache.Latitude
	lon := weatherCache.Longitude
	
	// Calculate rainbow predictions and sun angles
	rainbowPreds := predictRainbow(hourlyData, lat, lon, tzOff, true)
	
	rainbowScores := make(map[int64]int)
	for _, rp := range rainbowPreds {
		rainbowScores[rp.Time.Unix()] = rp.Score
	}
	
	// Pre-calculate sun angles for all hours
	sunAngles := make(map[int64]float64)
	for _, hour := range hourlyData {
		utcTime := hour.Time.Add(-time.Duration(tzOff) * time.Hour)
		sunElev := getSolarElevationUTC(lat, lon, utcTime)
		sunAngles[hour.Time.Unix()] = math.Round(sunElev*10) / 10 // Round to 1 decimal
	}

	// Only clear if row count changed to avoid flickering/selection loss
	if hourlyTable.RowCount() != len(hourlyData) {
		hourlyTable.Clear()
	}
	
	for i, hour := range hourlyData {
		icon, _ := weatherInfo(hour.WeatherCode)
		temp := formatTemp(hour.Temperature2m)
		
		// Get sun angle
		sunAngle := sunAngles[hour.Time.Unix()]
		sunAngleStr := fmt.Sprintf("%.1f°", sunAngle)

		switch {
		case sunAngle <= 0:
			sunAngleStr = fmt.Sprintf("🌙 %.1f°", sunAngle)      // Night
		case sunAngle < 8:
			sunAngleStr = fmt.Sprintf("🌤️ %.1f°", sunAngle)     // Dawn/day but low angle
		default:
			sunAngleStr = fmt.Sprintf("☀️ %.1f°", sunAngle)      // Full day
		}
		
		// Get rainbow score if applicable
		rainStr := "-"
		if score, exists := rainbowScores[hour.Time.Unix()]; exists {
			rainStr = fmt.Sprintf("%d%% 🌈", score)
		}
		
		ts := fmt.Sprintf("%d", hour.Time.Unix())
		if appConfig.Verifications[ts] {
			rainStr += " ✅"
		}

		hourlyTable.SetCell(0, i, hour.Time.Format("03:04 PM"))
		hourlyTable.SetCell(1, i, icon)
		hourlyTable.SetCell(2, i, temp)
		hourlyTable.SetCell(3, i, rainStr)
		hourlyTable.SetCell(4, i, sunAngleStr)
	}
}

func updateDailyTable() {
	if weatherCache == nil || dailyTable == nil {
		return
	}
	
	dailyData := extractDailyData(weatherCache)
	tzOff := float64(weatherCache.UTC_Offset_Seconds) / 3600
	lat := weatherCache.Latitude
	lon := weatherCache.Longitude

	dailyTable.Clear()
	now := time.Now()
	
	for _, day := range dailyData {
		// 1. Get hourly data for THIS specific day
		dayHourly := extractHourlyDataForDay(weatherCache, day.Date)
		
		// 2. Predict rainbows for those 24 hours
		// Set includePast to true so we get scores for the whole day
		rainbows := predictRainbow(dayHourly, lat, lon, tzOff, true)
		
		rainbowStr := "-"
		if len(rainbows) > 0 {
			// predictRainbow returns results sorted by Score descending
			// so the first element is the daily best.
			rainbowStr = fmt.Sprintf("%d%% 🌈", rainbows[0].Score)
		}

		// UI formatting
		var dayStr, dateStr string
		isPast := day.Date.Before(now.AddDate(0, 0, -1))
		isToday := day.Date.Format("2006-01-02") == now.Format("2006-01-02")
		
		if isToday {
			dayStr = "Today"
			dateStr = day.Date.Format("02/01/06")
		} else if isPast {
			dayStr = day.Date.Format("Mon")
			dateStr = day.Date.Format("02/01/06")
		} else {
			dayStr = day.Date.Format("Mon")
			dateStr = day.Date.Format("02/01/06")
		}
		icon, _ := weatherInfo(day.WeatherCode)
		high := formatTemp(day.TemperatureMax)
		low := formatTemp(day.TemperatureMin)

		row := dailyTable.RowCount()
		dailyTable.SetCell(0, row, dateStr)
		dailyTable.SetCell(1, row, dayStr)
		dailyTable.SetCell(2, row, icon)
		dailyTable.SetCell(3, row, fmt.Sprintf("%s / %s", high, low))
		dailyTable.SetCell(4, row, rainbowStr) // Now displays Rainbow score instead of Precip
	}
}

func onDailyTableSelection() {
	selectedRow := dailyTable.SelectedRow()
	if selectedRow < 0 || weatherCache == nil {
		return
	}
	
	dailyData := extractDailyData(weatherCache)
	if selectedRow >= len(dailyData) {
		return
	}
	
	selectedDay := dailyData[selectedRow]
	selectedDate = selectedDay.Date.Format("2006-01-02")
	
	// Check if selected day is today
	todayStr := time.Now().Format("2006-01-02")
	if selectedDate == todayStr {
		// Reset to current weather
		selectedDayData = nil
		selectedHourlyData = nil
		selectedDate = "today"
	} else {
		// Load selected day's data
		selectedDayData = &selectedDay
		selectedHourlyData = extractHourlyDataForDay(weatherCache, selectedDay.Date)
	}
	
	selectedHourData = nil
	
	// Update all displays
	updateCurrentWeather()
	updateFeatures()
	updateHourlyTable()
}

func onHourlyTableSelection() {
	selectedRow := hourlyTable.SelectedRow()
	if selectedRow < 0 || weatherCache == nil {
		return
	}
	
	var hourlyData []HourlyData
	if selectedHourlyData != nil && len(selectedHourlyData) > 0 {
		hourlyData = selectedHourlyData
	} else {
		hourlyData = extractHourlyDataForDay(weatherCache, time.Now())
	}
	
	if selectedRow >= len(hourlyData) {
		return
	}
	
	selectedHourData = &hourlyData[selectedRow]
	
	ts := fmt.Sprintf("%d", selectedHourData.Time.Unix())
	if checkVerified != nil {
		checkVerified.SetChecked(appConfig.Verifications[ts])
	}
	
	if mainCanvas != nil {
		mainCanvas.Paint()
	}
}

func updateSearchCombo() {
	searchCombo.Clear()
	for _, item := range searchItems {
		label := item.Name
		if item.Admin1 != "" {
			label += ", " + item.Admin1
		}
		if item.Country != "" {
			label += " (" + item.Country + ")"
		}
		searchCombo.AddItem(label)
	}
}

// ---------------------------------------------------------------------
// Event handlers
// ---------------------------------------------------------------------
func onSearchEditChange() {
	q := strings.TrimSpace(searchEdit.Text())
	if len(q) < 2 {
		searchItems = nil
		updateSearchCombo()
		searchCombo.SetVisible(false)
		return
	}
	
	searchCityAsync(q, func(results []geocodingItem, err error) {
		if err != nil {
			searchItems = nil
			searchCombo.SetVisible(false)
			return
		}
		
		searchItems = results
		if len(results) > 0 {
			updateSearchCombo()
			searchCombo.SetVisible(true)
			x, y := searchEdit.Position()
			searchCombo.SetPosition(x, y+24)
		} else {
			searchCombo.SetVisible(false)
		}
		searchCombo.SetSelectedIndex(0)
	})
}

func onSearchComboChange(index int) {
    if index < 0 || index >= len(searchItems) {
        return
    }
    selected := searchItems[index]
    searchItems = nil
    updateSearchCombo()
    searchCombo.SetVisible(false)
    currentLat = selected.Lat
    currentLon = selected.Lon
    locationName = selected.Name
    countryName = selected.Country
    
    // UPDATE: Set lat/lon input fields with the searched coordinates
    if editLat != nil {
        editLat.SetText(fmt.Sprintf("%.4f", currentLat))
    }
    if editLon != nil {
        editLon.SetText(fmt.Sprintf("%.4f", currentLon))
    }
    
    searchEdit.SetText(locationName + ", " + countryName)
    
    // Reset selected day when changing location
    selectedDayData = nil
    selectedHourlyData = nil
    selectedDate = "today"
    selectedHourData = nil
    
    updateData()
}

func onGeoClick() {
    btnGeo.SetEnabled(false)
    btnGeo.SetText("⏳")
    
    getUserLocationAsync(func(lat, lon float64, city, country string, err error) {
        if err != nil {
            lat, lon, city, country = 40.71, -74.00, "New York", "US"
        }
        currentLat = lat
        currentLon = lon
        locationName = city
        countryName = country
        
        // UPDATE: Set lat/lon input fields with geo coordinates
        if editLat != nil {
            editLat.SetText(fmt.Sprintf("%.4f", currentLat))
        }
        if editLon != nil {
            editLon.SetText(fmt.Sprintf("%.4f", currentLon))
        }
        
        // Reset selected day when changing location
        selectedDayData = nil
        selectedHourlyData = nil
        selectedDate = "today"
        selectedHourData = nil
        
        updateData()
        
        btnGeo.SetEnabled(true)
        btnGeo.SetText("📍")
    })
}

func updateData() {
	if labelLocation != nil {
		labelLocation.SetText("Loading weather data...")
	}
	
	if editLat != nil {
		editLat.SetText(fmt.Sprintf("%.4f", currentLat))
	}
	if editLon != nil {
		editLon.SetText(fmt.Sprintf("%.4f", currentLon))
	}
	
	// Reset selected day when fetching new data
	selectedDayData = nil
	selectedHourlyData = nil
	selectedDate = "today"
	selectedHourData = nil
	
	fetchWeatherAsync(currentLat, currentLon, func(w *WeatherResponse, err error) {
		if err == nil && w != nil {
			weatherCache = w
			timezoneOffsetHours = float64(w.UTC_Offset_Seconds) / 3600
			
			updateCurrentWeather()
			updateFeatures()
			updateHourlyTable()
			updateDailyTable()
			
			logWeatherData(w, currentLat, currentLon, locationName+", "+countryName)
		} else if err != nil {
			if labelLocation != nil {
				labelLocation.SetText("Error loading weather data")
			}
			if logger != nil {
				logger.Printf("Error fetching weather: %v", err)
			}
		}
	})
}

func showRainbowFormula() {
	
text := `A rainbow requires three conditions:
   1. Sun behind you (angle 0°-42°)
   2. Rain in front of you
   3. Sunlight not completely blocked

SUN ELEVATION SCORE (max 30)
   - Optimal range: 5°-35° (peaks at ~20°)
   - Outside optimal: reduced score
   - Sun below 0° or above 42° → impossible

PRECIPITATION SCORE (max 40)
   - 30-70% chance: full 40 points
   - >70% chance: 30 points (too heavy blocks sun)
   - <30% chance: scales linearly 0-20 points

CLOUD COVER SCORE (max 30)
   - 30-70% cover: full 30 points
   - 70-90% cover: 15 points
   - >90% cover: 0 points (sun blocked)
   - <30% cover: scales linearly 0-15 points

STRICT PENALTIES:
   - Cloud cover >95% → score × 0.1 (sun completely blocked)
   - Precipitation <10% → score × 0.1 (not enough moisture)

SCALE:
   - 90-98% : Excellent chance
   - 70-89% : Good chance
   - 40-69% : Possible
   - 10-39% : Unlikely
   - <10%    : Not shown`

		wui.MessageBox("Rainbow Formula Calculation", text)	
}

func onColorCanvasPaint(c *wui.Canvas) {
	colors := []wui.Color{}
	for _, col := range appConfig.Colors.RainbowColors {
		colors = append(colors, wuiColor(col))
	}
	
	w, h := c.Size()
	if len(colors) > 0 {
		rectWidth := w / len(colors)
		for i, col := range colors {
			x := i * rectWidth
			c.FillRect(x, 0, rectWidth, h, col)
		}
	}
}

func onMainCanvasPaint(c *wui.Canvas) {
	w, h := c.Size()
	
	if weatherCache == nil {
		c.FillRect(0, 0, w, h, wuiColor(appConfig.Colors.CanvasBackground))
		c.TextOut(10, 10, "Loading...", wui.RGB(0,0,0))
		return
	}
	
	lat := weatherCache.Latitude
	lon := weatherCache.Longitude
	tzOff := float64(weatherCache.UTC_Offset_Seconds) / 3600
	
	var targetHour HourlyData
	if selectedHourData != nil {
		targetHour = *selectedHourData
	} else {
		var hourlyData []HourlyData
		if selectedHourlyData != nil && len(selectedHourlyData) > 0 {
			hourlyData = selectedHourlyData
		} else {
			hourlyData = extractHourlyDataForDay(weatherCache, time.Now())
		}

		// DEFAULT: Show current hour if it's today, otherwise show best rainbow
		isToday := selectedDate == "today" || selectedDate == "" || (selectedDayData != nil && selectedDayData.Date.Format("2006-01-02") == time.Now().Format("2006-01-02"))

		if isToday {
			now := time.Now()
			found := false
			for _, hd := range hourlyData {
				if hd.Time.Hour() == now.Hour() {
					targetHour = hd
					found = true
					break
				}
			}
			if !found && len(hourlyData) > 0 {
				targetHour = hourlyData[0]
			}
		} else {
			rainbowPreds := predictRainbow(hourlyData, lat, lon, tzOff, true)
			bestTime := time.Now()
			if len(rainbowPreds) > 0 {
				bestTime = rainbowPreds[0].Time
			}

			found := false
			for _, hd := range hourlyData {
				if hd.Time.Equal(bestTime) {
					targetHour = hd
					found = true
					break
				}
			}
			if !found && len(hourlyData) > 0 {
				targetHour = hourlyData[0]
			}
		}
	}
	
	cloudCover := targetHour.CloudCover
	precipProb := targetHour.PrecipitationProb
	temp := targetHour.Temperature2m
	humidity := targetHour.RelativeHumidity2m
	wind := targetHour.WindSpeed10m
	code := targetHour.WeatherCode
	
	sunElev := getSolarElevationUTC(lat, lon, targetHour.Time.Add(-time.Duration(tzOff) * time.Hour))
	
	// --- NEW REALISTIC WEATHER CLASSIFICATION ---
	isFog := code == 45 || code == 48
	isDrizzle := (code >= 51 && code <= 57)
	isRain := (code >= 61 && code <= 67) || (code >= 80 && code <= 82)
	isSnow := (code >= 71 && code <= 77) || (code >= 85 && code <= 86)
	isHail := code == 96 || code == 99
	isThunderstorm := code >= 95
	isPrecipitating := isDrizzle || isRain || isSnow || isHail
	isBlizzard := isSnow && wind > 30 // Heavy wind + Snow
	
	rainbowScore := 0
	hourlyDataArr := []HourlyData{targetHour}
	preds := predictRainbow(hourlyDataArr, lat, lon, tzOff, true)
	if len(preds) > 0 {
		rainbowScore = preds[0].Score
	}
	
	// --- GROUND COLORS ---
	groundColor1 := wuiColor(appConfig.Colors.GroundGreen1)
	groundColor2 := wuiColor(appConfig.Colors.GroundGreen2)
	
	if temp < 0 || isSnow {
		groundColor1 = wuiColor(appConfig.Colors.GroundSnow1)
		groundColor2 = wuiColor(appConfig.Colors.GroundSnow2)
	} else if temp > 30 && humidity < 30 {
		groundColor1 = wuiColor(appConfig.Colors.GroundDesert1)
		groundColor2 = wuiColor(appConfig.Colors.GroundDesert2)
	} else if isRain || isThunderstorm {
		groundColor1 = wuiColor(appConfig.Colors.GroundStorm1)
		groundColor2 = wuiColor(appConfig.Colors.GroundStorm2)
	}

	// --- SKY COLORS ---
	skyColor := appConfig.Colors.SkyDayDay
	if sunElev < -5 {
		skyColor = appConfig.Colors.SkyNight
	} else if sunElev < 10 {
		skyColor = appConfig.Colors.SkySunset
	}
	
	cloudDarken := float64(cloudCover * 0.8) // Max darken scaling
	if isThunderstorm || isBlizzard {
		cloudDarken = 100 // Heavily darken sky for severe weather
	}
	
	skyR := math.Max(0, float64(skyColor[0])-cloudDarken)
	skyG := math.Max(0, float64(skyColor[1])-cloudDarken)
	skyB := math.Max(0, float64(skyColor[2])-cloudDarken)
	
	if isThunderstorm && (animFrame%30 == 0 || animFrame%30 == 1) {
		skyR, skyG, skyB = 255, 255, 255 // Lightning flash triggers sky flash
	}

	horizonR := math.Min(255, skyR+50)
	horizonG := math.Min(255, skyG+50)
	horizonB := math.Min(255, skyB+50)

	// Draw Vertical Gradient Sky
	for y := 0; y < h; y += 4 {
		ratio := float64(y) / float64(h)
		r := uint8(skyR*(1-ratio) + horizonR*ratio)
		g := uint8(skyG*(1-ratio) + horizonG*ratio)
		b := uint8(skyB*(1-ratio) + horizonB*ratio)
		c.FillRect(0, y, w, 4, wui.RGB(r, g, b))
	}
	
	// --- STARS ---
	if sunElev < -5 && cloudCover < 50 && !isFog {
		starColor := wuiColor(appConfig.Colors.Star)
		c.FillRect(20, 20, 2, 2, starColor)
		c.FillRect(120, 30, 2, 2, starColor)
		c.FillRect(200, 15, 2, 2, starColor)
		c.FillRect(350, 40, 2, 2, starColor)
	}
	
	// --- SUN / MOON ---
	if !isBlizzard { // Blizzards fully obscure celestial bodies
		sunX := 40
		sunY := h - 40 - int(sunElev*2) // Map elevation
		if sunY > h { sunY = h }
		if sunY < 20 { sunY = 20 }
		
		if sunElev >= -5 {
			pulse := int(math.Sin(float64(animFrame)*0.2) * 5)
			c.FillEllipse(sunX-5-pulse, sunY-5-pulse, 50+pulse*2, 50+pulse*2, wuiColor(appConfig.Colors.SunOuter))
			c.FillEllipse(sunX, sunY, 40, 40, wuiColor(appConfig.Colors.SunInner))
		} else {
			c.FillEllipse(sunX, 20, 30, 30, wuiColor(appConfig.Colors.Moon))
			c.FillEllipse(sunX+5, 25, 8, 8, wuiColor(appConfig.Colors.MoonCrater))
			c.FillEllipse(sunX+15, 35, 10, 10, wuiColor(appConfig.Colors.MoonCrater))
		}
	}

	// --- CLOUDS ---
	if cloudCover > 10 {
		cloudColor := wuiColor(appConfig.Colors.CloudWhite)
		if cloudCover > 50 { cloudColor = wuiColor(appConfig.Colors.CloudLightGray) }
		if cloudCover > 80 { cloudColor = wuiColor(appConfig.Colors.CloudDarkGray) }
		if isThunderstorm || isBlizzard { cloudColor = wuiColor(appConfig.Colors.CloudStorm) }
		
		numClouds := int(cloudCover / 10)
		if isBlizzard { numClouds = 10 } // Force heavy cloud cover for blizzard
		
		for i := 0; i < numClouds; i++ {
			speed := float64(i%3+1) * 0.5
			if wind > 20 { speed *= 2 } // Clouds move faster in wind
			offset := int(float64(animFrame) * speed)
			cx := (i * 45 + offset) % (w + 100) - 50
			cy := 10 + (i*10)%40
			
			c.FillEllipse(cx, cy, 70, 35, cloudColor)
			c.FillEllipse(cx+15, cy-15, 60, 45, cloudColor)
			c.FillEllipse(cx-10, cy-5, 50, 40, cloudColor)
		}
	}
	
	// --- RAINBOW ---
	if rainbowScore > 10 && sunElev > 0 {
		rainbowColors := []wui.Color{}
		for _, col := range appConfig.Colors.RainbowColors {
			rainbowColors = append(rainbowColors, wuiColor(col))
		}
		
		cx := w / 2 + 50
		cy := h - 20
		radius := 110 + int(sunElev)
		if radius > w/2 { radius = w/2 }
		
		for i, col := range rainbowColors {
			r := radius - (i * 5)
			c.Arc(cx-r, cy-r, r*2, r*2, 270, 180, col)
			c.Arc(cx-r+1, cy-r+1, r*2-2, r*2-2, 270, 180, col)
			c.Arc(cx-r+2, cy-r+2, r*2-4, r*2-4, 270, 180, col)
			c.Arc(cx-r+3, cy-r+3, r*2-6, r*2-6, 270, 180, col)
		}
	}

	// --- GROUND ---
	c.FillEllipse(-50, h-40, w/2+100, 100, groundColor1)
	c.FillEllipse(w/2-50, h-60, w/2+100, 150, groundColor2)

	// --- FOG ---
	if isFog {
		fogColor := wuiColor(appConfig.Colors.CloudLightGray)
		for i := 0; i < 6; i++ {
			drift := int(float64(animFrame) * 0.5)
			cx := (i * 70 + drift) % (w + 150) - 75
			cy := h - 70 + (i*10)%30
			c.FillEllipse(cx, cy, 180, 50, fogColor)
		}
	}

	// --- PRECIPITATION (Rain, Snow, Hail, Drizzle) ---
	windOffset := int(wind / 3) // Dynamic horizontal drift based on wind speed
	
	if isPrecipitating {
		dropColor := wuiColor(appConfig.Colors.RainDrop)
		numDrops := 40
		
		// Map intensity based on strict WMO codes
		if isDrizzle {
			numDrops = 20
			dropColor = wuiColor(appConfig.Colors.CloudLightGray) // Fine, light mist
		} else if isRain {
			if code == 65 || code == 82 { numDrops = 150 } // Heavy rain
		} else if isSnow {
			dropColor = wuiColor(appConfig.Colors.SnowDrop)
			numDrops = 80
			if code == 75 || code == 86 { numDrops = 200 } // Heavy snow
			if isBlizzard { numDrops = 350 } // Blizzard whiteout conditions
		} else if isHail {
			dropColor = wuiColor(appConfig.Colors.SnowDrop)
			numDrops = 60
		}

		for i := 0; i < numDrops; i++ {
			x := (i * 67) % w // Pseudo-random spread
			
			if isSnow {
				fallSpeed := (i%2 + 1) * 2
				if isBlizzard { fallSpeed = (i%3 + 3) * 3 } // Faster lateral movement in storms
				
				y := (i*17 + int(animFrame)*fallSpeed) % h
				
				// Apply sine wave drift for snow fluttering + standard wind
				drift := int(math.Sin(float64(animFrame)*0.05+float64(i))*10) + int(wind)
				x = (x + drift + w) % w
				
				flakeSize := (i%2) + 2
				if isBlizzard { flakeSize = 1 + (i%2) } // Smaller, violently blowing chunks
				
				c.FillRect(x, y, flakeSize, flakeSize, dropColor)
				
			} else if isHail {
				fallSpeed := (i%2 + 4) * 5
				y := (i*17 + int(animFrame)*fallSpeed) % h
				x = (x + windOffset + w) % w
				
				c.FillEllipse(x, y, 4, 4, dropColor) // Round hail pellets
				
			} else {
				// Rain & Drizzle Logic
				fallSpeed := (i%3 + 3) * 5
				length := fallSpeed
				
				if isDrizzle { 
					fallSpeed = (i%2 + 1) * 3
					length = 3
				}
				if code == 65 || code == 82 { fallSpeed += 5 } // Heavy rain falls faster
				
				y := (i*23 + int(animFrame)*fallSpeed) % h
				x = (x + windOffset + w) % w
				
				c.Line(x, y, x-windOffset, y+length, dropColor)
			}
		}
	}
	
	// --- THUNDERSTORM / LIGHTNING ---
	if isThunderstorm {
		if animFrame%30 == 0 || animFrame%30 == 1 {
			lightningColor := wuiColor(appConfig.Colors.Lightning)
			lx := w/2 + (int(animFrame*7) % 100) - 50 // Randomize lightning strike position
			c.Line(lx, 20, lx-15, 60, lightningColor)
			c.Line(lx-15, 60, lx+10, 75, lightningColor)
			c.Line(lx+10, 75, lx-30, h-40, lightningColor)
			
			// Secondary branches
			if i := int(animFrame) % 2; i == 0 {
				c.Line(lx-15, 60, lx-40, 80, lightningColor)
			}
		}
	}
	
	// --- WIND DIRT/LINES (Visualizing high wind when clear) ---
	if wind > 20 && !isSnow {
		windColor := wuiColor(appConfig.Colors.WindLine)
		offset1 := (int(animFrame) * int(wind/4)) % w
		offset2 := (int(animFrame) * int(wind/3)) % w
		c.Line(offset1, h-50, offset1+30, h-50, windColor)
		c.Line((offset2+w/2)%w, h-80, (offset2+w/2)%w+40, h-80, windColor)
	}
	
	// --- INFO OVERLAY ---
	info := fmt.Sprintf("%s | Temp: %.1f° | Wind: %.1f | Rain Prob: %.0f%%", targetHour.Time.Format("Mon 15:04"), temp, wind, precipProb)
	c.TextOut(5, c.Height() - 15, info, wui.RGB(255,255,255))
}

func createUI() {
	windowFont, _ := wui.NewFont(wui.FontDesc{Name: "Tahoma", Height: -11})
	mainWindow = wui.NewWindow()
	mainWindow.SetFont(windowFont)
	
	mainWindow.SetInnerSize(655, 425)
	mainWindow.SetPosition(200, 70)
	mainWindow.SetResizable(false)
	mainWindow.SetHasMaxButton(false)
	mainWindow.SetTitle("Rainbow Tool")

	loadThemes := func() {
		exePath, _ := os.Executable()
		exeDir := filepath.Dir(exePath)
		schemesDir := filepath.Join(exeDir, "schemes")
		files, err := os.ReadDir(schemesDir)
		if err != nil {
			return
		}

		themeCombo.Clear()
		themeItems = nil
		selectedIndex := -1
		for _, f := range files {
			if !f.IsDir() && strings.HasSuffix(strings.ToLower(f.Name()), ".json") {
				name := strings.TrimSuffix(f.Name(), filepath.Ext(f.Name()))
				themeItems = append(themeItems, f.Name())
				themeCombo.AddItem(name)
				if name == appConfig.ThemeName {
					selectedIndex = len(themeItems) - 1
				}
			}
		}
		if selectedIndex >= 0 {
			themeCombo.SetSelectedIndex(selectedIndex)
		}
	}

	applyTheme := func(index int) {
		if index < 0 || index >= len(themeItems) {
			return
		}
		exePath, _ := os.Executable()
		exeDir := filepath.Dir(exePath)
		themePath := filepath.Join(exeDir, "schemes", themeItems[index])

		data, err := os.ReadFile(themePath)
		if err != nil {
			return
		}

		var newConfig AppConfig
		if err := json.Unmarshal(data, &newConfig); err != nil {
			return
		}

		// Keep existing verifications
		oldVerifications := appConfig.Verifications
		appConfig = newConfig
		appConfig.Verifications = oldVerifications
		appConfig.ThemeName = strings.TrimSuffix(themeItems[index], filepath.Ext(themeItems[index]))

		saveConfig()
		if mainCanvas != nil {
			mainCanvas.Paint()
		}
		if colorCanvas != nil {
			colorCanvas.Paint()
		}
	}

	// Header row
	logo := wui.NewLabel()
	logo.SetBounds(10, 10, 150, 24)
	logo.SetText("🌈 Rainbow Tool")
	logoFont, _ := wui.NewFont(wui.FontDesc{Name: "Tahoma", Height: -14, Bold: true})
	logo.SetFont(logoFont)
	mainWindow.Add(logo)

	// LAT/LON inputs (moved to left side, where search used to be)

	editLat = wui.NewEditLine()
	editLat.SetBounds(285, 12, 50, 20)
	editLat.SetText("40.71")
	mainWindow.Add(editLat)

	editLon = wui.NewEditLine()
	editLon.SetBounds(340, 12, 50, 20)
	editLon.SetText("-74.00")
	mainWindow.Add(editLon)

	btnUpdateLoc = wui.NewButton()
	btnUpdateLoc.SetBounds(395, 12, 40, 22)
	btnUpdateLoc.SetText("Go")
	btnUpdateLoc.SetOnClick(func() {
		latStr := editLat.Text()
		lonStr := editLon.Text()
		lat, err1 := strconv.ParseFloat(latStr, 64)
		lon, err2 := strconv.ParseFloat(lonStr, 64)
		if err1 == nil && err2 == nil {
			currentLat = lat
			currentLon = lon
			locationName = fmt.Sprintf("%.2f, %.2f", lat, lon)
			countryName = ""
			searchEdit.SetText(locationName)
			
			reverseGeocodeAsync(lat, lon, func(name string, err error) {
				if err == nil && name != "" {
					locationName = name
					searchEdit.SetText(name)
				}
			})
			
			selectedDayData = nil
			selectedHourlyData = nil
			selectedDate = "today"
			selectedHourData = nil
			updateData()
		}
	})
	mainWindow.Add(btnUpdateLoc)

	// Search section (moved to right side, where lat/lon used to be)
	searchEdit = wui.NewEditLine()
	searchEdit.SetBounds(440, 12, 120, 20)
	searchEdit.SetText("")
	searchEdit.SetOnTextChange(onSearchEditChange)
	mainWindow.Add(searchEdit)

	searchCombo = wui.NewComboBox()
	searchCombo.SetBounds(440, 38, 120, 100)
	searchCombo.SetVisible(false)
	searchCombo.SetOnChange(onSearchComboChange)
	mainWindow.Add(searchCombo)

	buttonSearch := wui.NewButton()
	buttonSearch.SetBounds(565, 12, 50, 22)
	buttonSearch.SetText("Search")
	buttonSearch.SetOnClick(func() { onSearchEditChange() })
	mainWindow.Add(buttonSearch)

	btnGeo = wui.NewButton()
	btnGeo.SetBounds(620, 12, 28, 24)
	btnGeo.SetText("📍")
	btnGeo.SetOnClick(onGeoClick)
	mainWindow.Add(btnGeo)

	// Theme selection
	// themeLabel := wui.NewLabel()
	// themeLabel.SetBounds(240, 48, 12, 24)
	// themeLabel.SetText("Theme:")
	// mainWindow.Add(themeLabel)

	themeCombo = wui.NewComboBox()
	themeCombo.SetBounds(180, 12, 100, 24)
	themeCombo.SetOnChange(applyTheme)
	mainWindow.Add(themeCombo)
	loadThemes()

	// Current weather section
	labelMainIcon = wui.NewLabel()
	labelMainIcon.SetBounds(10, 45, 40, 50)
	fontIcon, _ := wui.NewFont(wui.FontDesc{Name: "Segoe UI Emoji", Height: 30})
	labelMainIcon.SetFont(fontIcon)
	labelMainIcon.SetText("☀️")
	mainWindow.Add(labelMainIcon)

	labelMainTemp = wui.NewLabel()
	labelMainTemp.SetBounds(50, 45, 90, 50)
	fontTemp, _ := wui.NewFont(wui.FontDesc{Name: "Tahoma", Height: -28, Bold: true})
	labelMainTemp.SetFont(fontTemp)
	labelMainTemp.SetText("  °")
	mainWindow.Add(labelMainTemp)

	labelMainFeels = wui.NewLabel()
	labelMainFeels.SetBounds(10, 95, 130, 16) // Moved down slightly to prevent Temp overlap
	labelMainFeels.SetText("Feels --°")
	mainWindow.Add(labelMainFeels)

	labelLocation = wui.NewLabel()
	labelLocation.SetBounds(150, 45, 200, 20)
	fontLoc, _ := wui.NewFont(wui.FontDesc{Name: "Tahoma", Height: -13, Bold: true})
	labelLocation.SetFont(fontLoc)
	labelLocation.SetText("--")
	mainWindow.Add(labelLocation)

	labelDate = wui.NewLabel()
	labelDate.SetBounds(150, 65, 100, 16)
	labelDate.SetText("")
	mainWindow.Add(labelDate)

	labelDesc = wui.NewLabel()
	labelDesc.SetBounds(150, 85, 100, 16)
	labelDesc.SetText("--")
	mainWindow.Add(labelDesc)

	// Feature cards
	featureLabels = make(map[string]*wui.Label)
	cardDefs := []struct {
		name, icon, title string
		x, y0             int
	}{
		{"rainbow", "🌈", "Rainbow Prob", 10, 120},
		{"sun", "☀️", "Sun Elevation", 135, 120},
		{"precip", "☂️", "Rain Prob", 10, 185},
		{"cloud", "☁️", "Cloud Cover", 135, 185},
	}
	
	for _, def := range cardDefs {
		x, y0 := def.x, def.y0
		
		title := wui.NewLabel()
		title.SetBounds(x, y0, 115, 14)
		title.SetText(def.icon + " " + def.title)
		fontTitle, _ := wui.NewFont(wui.FontDesc{Name: "Tahoma", Height: -11, Bold: true})
		title.SetFont(fontTitle)
		mainWindow.Add(title)

		value := wui.NewLabel()
		value.SetBounds(x, y0+16, 115, 18)
		fontVal, _ := wui.NewFont(wui.FontDesc{Name: "Tahoma", Height: -14, Bold: true})
		value.SetFont(fontVal)
		value.SetText("--")
		featureLabels[def.name] = value
		mainWindow.Add(value)

		detail := wui.NewLabel()
		detail.SetBounds(x, y0+34, 115, 14)
		fontDetail, _ := wui.NewFont(wui.FontDesc{Name: "Tahoma", Height: -10})
		detail.SetFont(fontDetail)
		detail.SetText("")
		featureLabels[def.name+"Detail"] = detail
		mainWindow.Add(detail)

		badge := wui.NewLabel()
		badge.SetBounds(x, y0+48, 115, 14)
		fontBadge, _ := wui.NewFont(wui.FontDesc{Name: "Tahoma", Height: -10, Italic: true})
		badge.SetFont(fontBadge)
		badge.SetText("")
		featureLabels[def.name+"Badge"] = badge
		mainWindow.Add(badge)
	}
	
	btnHelp := wui.NewButton()
	btnHelp.SetBounds(110, 120, 14, 14)
	font, _ := wui.NewFont(wui.FontDesc{Name: "Tahoma", Height: -10, Bold: false})
	btnHelp.SetFont(font)
	btnHelp.SetText("?")
	btnHelp.SetOnClick(func() {
		showRainbowFormula()
	})
	mainWindow.Add(btnHelp)

	// Hourly table
	labelHourly := wui.NewLabel()
	labelHourly.SetBounds(10, 255, 120, 16)
	labelHourly.SetText("⌛ Hourly forecast")
	fontTable, _ := wui.NewFont(wui.FontDesc{Name: "Tahoma", Height: -12, Bold: true})
	labelHourly.SetFont(fontTable)
	mainWindow.Add(labelHourly)
	
	checkVerified = wui.NewCheckBox()
	checkVerified.SetBounds(140, 255, 120, 16)
	checkVerified.SetText("Verified Rainbow")
	checkVerified.SetOnChange(func(checked bool) {
		if selectedHourData != nil {
			ts := fmt.Sprintf("%d", selectedHourData.Time.Unix())
			if checked {
				appConfig.Verifications[ts] = true
			} else {
				delete(appConfig.Verifications, ts)
			}
			saveConfig()
			updateHourlyTable()
		}
	})
	mainWindow.Add(checkVerified)

	hourlyTable = wui.NewStringTable("🕒 Hour", "Icon", "Temp", "Rainbow", "Sun Angle")
	hourlyTable.SetBounds(10, 275, 310, 135)
	hourlyTable.SetOnSelectionChange(onHourlyTableSelection)
	mainWindow.Add(hourlyTable)

	// Daily table
	labelDaily := wui.NewLabel()
	labelDaily.SetBounds(330, 255, 180, 16)
	labelDaily.SetText("📅 14-day rainbow forecast")
	labelDaily.SetFont(fontTable)
	mainWindow.Add(labelDaily)

	// Separated Day and Date into two columns
	dailyTable = wui.NewStringTable("📅 Date", "✅ Day", "Icon", "High / Low", "Rainbow")
	dailyTable.SetBounds(330, 275, 315, 135)
	dailyTable.SetOnSelectionChange(onDailyTableSelection)
	mainWindow.Add(dailyTable)


	// labelCanvas := wui.NewLabel()
	// labelCanvas.SetBounds(360, 60, 200, 16)
	// labelCanvas.SetText("🎨 Rainbow Visualization")
	// labelCanvas.SetFont(fontTable)
	// mainWindow.Add(labelCanvas)

	mainCanvas = wui.NewPaintBox()
	// mainCanvas.SetBounds(260, 80, 385, 170)
	mainCanvas.SetBounds(330, 80, 315, 170)
	mainCanvas.SetOnPaint(onMainCanvasPaint)
	mainWindow.Add(mainCanvas)

	colorCanvas = wui.NewPaintBox()
	colorCanvas.SetBounds(526, 60, 119, 15)
	colorCanvas.SetOnPaint(onColorCanvasPaint)
	mainWindow.Add(colorCanvas)
}

func runCLI(query string) {
	searchCityAsync(query, func(items []geocodingItem, err error) {
		if err != nil || len(items) == 0 {
			fmt.Printf("Error: Could not find location '%s'\n", query)
			os.Exit(1)
		}
		item := items[0]
		fmt.Printf("Location: %s, %s (%f, %f)\n", item.Name, item.Country, item.Lat, item.Lon)
		fetchWeatherAsync(item.Lat, item.Lon, func(w *WeatherResponse, err error) {
			if err != nil {
				fmt.Printf("Error: Could not fetch weather: %v\n", err)
				os.Exit(1)
			}
			c := w.Current
			_, desc := weatherInfo(c.WeatherCode)
			fmt.Printf("Temperature: %.1f°C (Feels %.1f°C)\n", c.Temperature2m, c.ApparentTemperature)
			fmt.Printf("Condition: %s\n", desc)
			fmt.Printf("Humidity: %.0f%%\n", c.RelativeHumidity2m)
			fmt.Printf("Wind: %.1f km/h\n", c.WindSpeed10m)
			hourlyData := extractHourlyDataForDay(w, time.Now())
			tzOff := float64(w.UTC_Offset_Seconds) / 3600
			preds := predictRainbow(hourlyData, w.Latitude, w.Longitude, tzOff, false)
			if len(preds) > 0 {
				best := preds[0]
				fmt.Printf("Rainbow Probability: %d%% at %s (Sun Angle: %.1f°)\n", best.Score, best.Time.Format("15:04"), best.SunElev)
			} else {
				fmt.Println("Rainbow Probability: Low / None")
			}
			os.Exit(0)
		})
	})
	select {}
}

func createQuickUI() {
	windowFont, _ := wui.NewFont(wui.FontDesc{Name: "Tahoma", Height: -11})
	mainWindow = wui.NewWindow()
	mainWindow.SetFont(windowFont)
	mainWindow.SetInnerSize(340, 240)
	mainWindow.SetResizable(false)
	mainWindow.SetHasMaxButton(false)
	mainWindow.SetTitle("Rainbow Tool Quick")

	labelMainTemp = wui.NewLabel()
	labelMainTemp.SetBounds(10, 10, 100, 40)
	fontTemp, _ := wui.NewFont(wui.FontDesc{Name: "Tahoma", Height: -24, Bold: true})
	labelMainTemp.SetFont(fontTemp)
	labelMainTemp.SetText("--°C")
	mainWindow.Add(labelMainTemp)

	labelQuickRainbow := wui.NewLabel()
	labelQuickRainbow.SetBounds(120, 15, 200, 30)
	fontRainbow, _ := wui.NewFont(wui.FontDesc{Name: "Tahoma", Height: -18, Bold: true})
	labelQuickRainbow.SetFont(fontRainbow)
	labelQuickRainbow.SetText("🌈 --%")
	featureLabels = make(map[string]*wui.Label)
	featureLabels["rainbow"] = labelQuickRainbow
	mainWindow.Add(labelQuickRainbow)

	mainCanvas = wui.NewPaintBox()
	mainCanvas.SetBounds(10, 55, 320, 175)
	mainCanvas.SetOnPaint(onMainCanvasPaint)
	mainWindow.Add(mainCanvas)
}

func runQuickMode() {
	createQuickUI()
	go func() {
		ticker := time.NewTicker(50 * time.Millisecond)
		for range ticker.C {
			animFrame++
			if mainCanvas != nil {
				mainCanvas.Paint()
			}
		}
	}()
	getUserLocationAsync(func(lat, lon float64, city, country string, err error) {
		if err == nil {
			currentLat = lat
			currentLon = lon
			locationName = city
			countryName = country
		}
		updateData()
	})
	mainWindow.Show()
}

func main() {
	loadConfig()

	if len(os.Args) > 1 {
		arg := os.Args[1]
		if arg == "quick" {
			runQuickMode()
			return
		} else if !strings.HasPrefix(arg, "-") {
			runCLI(arg)
			return
		}
	}

	if err := setupLogging(); err != nil {
		fmt.Printf("Warning: Could not setup logging: %v\n", err)
	}
	
	createUI()
	
	go func() {
		ticker := time.NewTicker(50 * time.Millisecond)
		for range ticker.C {
			animFrame++
			if mainCanvas != nil {
				mainCanvas.Paint()
			}
		}
	}()
	
	getUserLocationAsync(func(lat, lon float64, city, country string, err error) {
		if err == nil {
			currentLat = lat
			currentLon = lon
			locationName = city
			countryName = country
		}
		updateData()
	})
	
	mainWindow.Show()
}