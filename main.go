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
func updateCurrentWeather() {
	if selectedDayData != nil && selectedDate != "today" {
		// Show selected day's data (not current)
		icon, desc := weatherInfo(selectedDayData.WeatherCode)
		labelMainTemp.SetText(formatTemp(selectedDayData.TemperatureMax))
		labelMainFeels.SetText(fmt.Sprintf("High / Low: %s / %s", formatTemp(selectedDayData.TemperatureMax), formatTemp(selectedDayData.TemperatureMin)))
		labelMainIcon.SetText(icon)
		labelDesc.SetText(desc)
		labelDate.SetText(selectedDayData.Date.Format("Mon, Jan 2, 2006"))
	} else if weatherCache != nil {
		// Show current weather
		c := weatherCache.Current
		icon, desc := weatherInfo(c.WeatherCode)
		labelMainTemp.SetText(formatTemp(c.Temperature2m))
		labelMainFeels.SetText(fmt.Sprintf("Feels %s", formatTemp(c.ApparentTemperature)))
		labelMainIcon.SetText(icon)
		labelDesc.SetText(desc)
		labelDate.SetText(time.Now().Format("Mon, Jan 2, 2006"))
		labelDate.SetText(time.Now().Format("Mon, Jan 2, 2006"))
	}
	
	loc := locationName
	if countryName != "" {
		loc += ", " + countryName
	}
	labelLocation.SetText(loc)
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
	dailyData := extractDailyData(weatherCache) // Unused, keeping if needed elsewhere
	
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
	featureLabels["rainbow"].SetText(bestRainbow)
	featureLabels["rainbowDetail"].SetText(bestRainbowDetail)
	featureLabels["rainbowBadge"].SetText(bestRainbowBadge)

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
	featureLabels["sun"].SetText(sunElevVal)
	featureLabels["sunDetail"].SetText(sunElevDetail)
	featureLabels["sunBadge"].SetText("Angle")

	precipProb := c.Precipitation
	if precipProb == 0 && len(dailyData) > 0 {
		precipProb = dailyData[0].PrecipitationMax
	}
	featureLabels["precip"].SetText(fmt.Sprintf("%.0f%%", precipProb))
	featureLabels["precipDetail"].SetText("Required for bows")
	featureLabels["precipBadge"].SetText("Moisture")

	featureLabels["cloud"].SetText(fmt.Sprintf("%.0f%%", c.CloudCover))
	featureLabels["cloudDetail"].SetText("Need < 95%")
	featureLabels["cloudBadge"].SetText("Blockage")

	// Remove older cards from updates

	if mainCanvas != nil {
		mainCanvas.Paint()
	}
	if colorCanvas != nil {
		colorCanvas.Paint()
	}
}

func updateHourlyTable() {
	if weatherCache == nil {
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

	hourlyTable.Clear()
	
	for _, hour := range hourlyData {
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

		row := hourlyTable.RowCount()
		hourlyTable.SetCell(0, row, hour.Time.Format("03:04 PM"))
		hourlyTable.SetCell(1, row, icon)
		hourlyTable.SetCell(2, row, temp)
		hourlyTable.SetCell(3, row, rainStr)
		hourlyTable.SetCell(4, row, sunAngleStr)
	}
}

func updateDailyTable() {
	if weatherCache == nil {
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
	// searchEdit.SetText("")
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
	labelLocation.SetText("Loading weather data...")
	
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
			labelLocation.SetText("Error loading weather data")
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
	colors := []wui.Color{
		wui.RGB(255, 0, 0),    // Red
		wui.RGB(255, 127, 0),  // Orange
		wui.RGB(255, 255, 0),  // Yellow
		wui.RGB(0, 255, 0),    // Green
		wui.RGB(0, 0, 255),    // Blue
		wui.RGB(75, 0, 130),   // Indigo
		wui.RGB(148, 0, 211),  // Violet
	}
	
	w, h := c.Size()
	rectWidth := w / len(colors)
	
	for i, col := range colors {
		x := i * rectWidth
		c.FillRect(x, 0, rectWidth, h, col)
	}
}

func onMainCanvasPaint(c *wui.Canvas) {
	w, h := c.Size()
	
	if weatherCache == nil {
		c.FillRect(0, 0, w, h, wui.RGB(200, 200, 200))
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
	
	cloudCover := targetHour.CloudCover
	precip := targetHour.PrecipitationProb
	temp := targetHour.Temperature2m
	humidity := targetHour.RelativeHumidity2m
	wind := targetHour.WindSpeed10m
	code := targetHour.WeatherCode
	
	sunElev := getSolarElevationUTC(lat, lon, targetHour.Time.Add(-time.Duration(tzOff) * time.Hour))
	
	rainbowScore := 0
	hourlyDataArr := []HourlyData{targetHour}
	preds := predictRainbow(hourlyDataArr, lat, lon, tzOff, true)
	if len(preds) > 0 {
		rainbowScore = preds[0].Score
	}
	
	groundColor1 := wui.RGB(34, 139, 34) // Default Green
	groundColor2 := wui.RGB(46, 139, 87)
	
	if temp < 0 || (code >= 71 && code <= 86) {
		groundColor1 = wui.RGB(240, 248, 255) // Alice Blue
		groundColor2 = wui.RGB(255, 250, 250) // Snow
	} else if temp > 30 && humidity < 30 {
		groundColor1 = wui.RGB(237, 201, 175) // Desert Sand
		groundColor2 = wui.RGB(210, 180, 140) // Tan
	} else if precip > 80 || code >= 95 {
		groundColor1 = wui.RGB(25, 100, 25) 
		groundColor2 = wui.RGB(35, 100, 50)
	}

	skyBaseR, skyBaseG, skyBaseB := uint8(135), uint8(206), uint8(235) // Light sky blue
	if sunElev < -5 {
		skyBaseR, skyBaseG, skyBaseB = 5, 5, 20 // Night
	} else if sunElev < 10 {
		skyBaseR, skyBaseG, skyBaseB = 255, 140, 0 // Sunset/Sunrise orange
	}
	
	cloudDarken := uint8(cloudCover * 0.8) // max darken
	skyR := uint8(math.Max(0, float64(skyBaseR)-float64(cloudDarken)))
	skyG := uint8(math.Max(0, float64(skyBaseG)-float64(cloudDarken)))
	skyB := uint8(math.Max(0, float64(skyBaseB)-float64(cloudDarken)))
	
	c.FillRect(0, 0, w, h, wui.RGB(skyR, skyG, skyB))
	
	if sunElev < -5 && cloudCover < 50 {
		c.FillRect(20, 20, 2, 2, wui.RGB(255,255,255))
		c.FillRect(120, 30, 2, 2, wui.RGB(255,255,255))
		c.FillRect(200, 15, 2, 2, wui.RGB(255,255,255))
		c.FillRect(350, 40, 2, 2, wui.RGB(255,255,255))
	}
	
	sunX := 80
	sunY := h - 40 - int(sunElev*2) // Map elevation
	if sunY > h { sunY = h }
	if sunY < 20 { sunY = 20 }
	
	if sunElev >= -5 {
		c.FillEllipse(sunX-5, sunY-5, 50, 50, wui.RGB(255, 255, 150))
		c.FillEllipse(sunX, sunY, 40, 40, wui.RGB(255, 255, 0))
	} else {
		c.FillEllipse(sunX, 20, 30, 30, wui.RGB(200, 200, 220))
		c.FillEllipse(sunX+5, 25, 8, 8, wui.RGB(170, 170, 190))
		c.FillEllipse(sunX+15, 35, 10, 10, wui.RGB(170, 170, 190))
	}

	if cloudCover > 10 {
		cloudColor := wui.RGB(255, 255, 255)
		if cloudCover > 50 { cloudColor = wui.RGB(200, 200, 200) }
		if cloudCover > 80 { cloudColor = wui.RGB(100, 100, 100) }
		if code >= 95 { cloudColor = wui.RGB(50, 50, 60) }
		
		numClouds := int(cloudCover / 10)
		for i := 0; i < numClouds; i++ {
			cx := (i * 45) % w
			cy := 10 + (i*10)%40
			c.FillEllipse(cx, cy, 60, 30, cloudColor)
			c.FillEllipse(cx+20, cy-10, 50, 40, cloudColor)
		}
	}

	c.FillEllipse(-50, h-40, w/2+100, 100, groundColor1)
	c.FillEllipse(w/2-50, h-60, w/2+100, 150, groundColor2)

	isSnow := (code >= 71 && code <= 75) || (code >= 85 && code <= 86)
	isStorm := code >= 95

	windOffset := int(wind / 5)
	
	if precip > 0 {
		dropColor := wui.RGB(150, 150, 200)
		if isSnow {
			dropColor = wui.RGB(255, 255, 255)
		}
		
		numDrops := int(precip)
		if numDrops > 80 { numDrops = 80 }
		
		for i := 0; i < numDrops; i++ {
			x := (i * 37) % w
			y := (i * 23) % (h - 30)
			
			if isSnow {
				c.FillRect(x, y, 3, 3, dropColor)
			} else {
				c.Line(x, y, x-windOffset, y+15, dropColor)
			}
		}
	}
	
	if isStorm {
		c.Line(w/2, 20, w/2-10, 50, wui.RGB(255, 255, 0))
		c.Line(w/2-10, 50, w/2+5, 60, wui.RGB(255, 255, 0))
		c.Line(w/2+5, 60, w/2-20, h-40, wui.RGB(255, 255, 0))
	}
	
	if wind > 20 {
		windColor := wui.RGB(200, 200, 200)
		c.Line(10, h-50, 40, h-50, windColor)
		c.Line(w/2, h-80, w/2+50, h-80, windColor)
	}

	if rainbowScore > 10 && sunElev > 0 {
		colors := []wui.Color{
			wui.RGB(255, 0, 0), wui.RGB(255, 127, 0), wui.RGB(255, 255, 0),
			wui.RGB(0, 255, 0), wui.RGB(0, 0, 255), wui.RGB(75, 0, 130), wui.RGB(148, 0, 211),
		}
		
		cx := w / 2 + 50
		cy := h - 20
		radius := 110 + int(sunElev)
		if radius > w/2 { radius = w/2 }
		
		for i, col := range colors {
			r := radius - (i * 5)
			c.Arc(cx-r, cy-r, r*2, r*2, 270, 180, col)
			c.Arc(cx-r+1, cy-r+1, r*2-2, r*2-2, 270, 180, col)
			c.Arc(cx-r+2, cy-r+2, r*2-4, r*2-4, 270, 180, col)
			c.Arc(cx-r+3, cy-r+3, r*2-6, r*2-6, 270, 180, col)
		}
	}
	
	info := fmt.Sprintf("%s | Temp: %.1f° | Wind: %.1f | Rain: %.0f%% | Bow: %d%%", targetHour.Time.Format("Mon 15:04"), temp, wind, precip, rainbowScore)
	c.TextOut(5, 5, info, wui.RGB(255,255,255))
}

func createUI() {
	windowFont, _ := wui.NewFont(wui.FontDesc{Name: "Tahoma", Height: -11})
	mainWindow = wui.NewWindow()
	mainWindow.SetFont(windowFont)
	
	mainWindow.SetInnerSize(655, 425)
	mainWindow.SetPosition(180, 70)
	mainWindow.SetResizable(false)
	mainWindow.SetHasMaxButton(false)
	mainWindow.SetTitle("Rainbow Tool")

	// Header row
	logo := wui.NewLabel()
	logo.SetBounds(10, 10, 150, 24)
	logo.SetText("🌈 Rainbow Tool")
	logoFont, _ := wui.NewFont(wui.FontDesc{Name: "Tahoma", Height: -14, Bold: true})
	logo.SetFont(logoFont)
	mainWindow.Add(logo)

	searchEdit = wui.NewEditLine()
	searchEdit.SetBounds(405, 12, 145, 24)
	searchEdit.SetText("")
	searchEdit.SetOnTextChange(onSearchEditChange)
	mainWindow.Add(searchEdit)

	searchCombo = wui.NewComboBox()
	searchCombo.SetBounds(405, 38, 145, 100)
	searchCombo.SetVisible(false)
	searchCombo.SetOnChange(onSearchComboChange)
	mainWindow.Add(searchCombo)

	buttonSearch := wui.NewButton()
	buttonSearch.SetBounds(555, 12, 55, 24)
	buttonSearch.SetText("Search")
	buttonSearch.SetOnClick(func() { onSearchEditChange() })
	mainWindow.Add(buttonSearch)

	btnGeo = wui.NewButton()
	btnGeo.SetBounds(615, 12, 28, 24)
	btnGeo.SetText("📍")
	btnGeo.SetOnClick(onGeoClick)
	mainWindow.Add(btnGeo)

	btnGeo = wui.NewButton()
	btnGeo.SetBounds(615, 12, 28, 24)
	btnGeo.SetText("📍")
	btnGeo.SetOnClick(onGeoClick)
	mainWindow.Add(btnGeo)

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
	btnHelp.SetBounds(80, 120, 14, 14)
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

func main() {
	if err := setupLogging(); err != nil {
		fmt.Printf("Warning: Could not setup logging: %v\n", err)
	}
	
	createUI()
	
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