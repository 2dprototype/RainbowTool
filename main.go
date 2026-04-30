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
	labelHumidity  *wui.Label
	labelWind      *wui.Label
	labelPressure  *wui.Label
	labelUV        *wui.Label
	labelDate      *wui.Label // Added to show selected date

	// feature cards
	featureLabels map[string]*wui.Label

	// hourly & daily tables
	hourlyTable *wui.StringTable
	dailyTable  *wui.StringTable

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
	Time              time.Time
	Temperature2m     float64
	WeatherCode       int
	PrecipitationProb float64
	CloudCover        float64
	RelativeHumidity2m float64
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
		url := fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=%f&longitude=%f&current=temperature_2m,relative_humidity_2m,apparent_temperature,precipitation,weather_code,cloud_cover,wind_speed_10m,wind_direction_10m,uv_index,dew_point_2m,surface_pressure&hourly=precipitation_probability,cloud_cover,temperature_2m,dew_point_2m,weather_code,relative_humidity_2m&daily=temperature_2m_max,temperature_2m_min,precipitation_probability_max,weather_code&timezone=auto&forecast_days=7&past_days=7",
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
		
		// For selected days, hide some current-specific data
		labelHumidity.SetText("--")
		labelWind.SetText("--")
		labelPressure.SetText("--")
		labelUV.SetText("--")
	} else if weatherCache != nil {
		// Show current weather
		c := weatherCache.Current
		icon, desc := weatherInfo(c.WeatherCode)
		labelMainTemp.SetText(formatTemp(c.Temperature2m))
		labelMainFeels.SetText(fmt.Sprintf("Feels %s", formatTemp(c.ApparentTemperature)))
		labelMainIcon.SetText(icon)
		labelDesc.SetText(desc)
		labelDate.SetText(time.Now().Format("Mon, Jan 2, 2006"))
		
		labelHumidity.SetText(fmt.Sprintf("%.0f%%", c.RelativeHumidity2m))
		labelWind.SetText(fmt.Sprintf("%.0f km/h", c.WindSpeed10m))
		labelPressure.SetText(fmt.Sprintf("%.0f hPa", c.SurfacePressure))
		labelUV.SetText(fmt.Sprintf("%.1f", c.UVIndex))
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

	// Moon Phase Calculation
	moonPhase := (float64(time.Now().UnixMilli()-time.Date(2000, 1, 6, 18, 14, 0, 0, time.UTC).UnixMilli()) / (1000 * 3600 * 24)) / 29.53
	moonIllum := 0.5 * (1 - math.Cos(2*math.Pi*moonPhase))

	// Aurora Score
	latScore := func() int {
		absLat := math.Abs(lat)
		if absLat > 55 { return 7 }
		if absLat > 48 { return 4 }
		return 1
	}()

	cloudScore := 1
	if c.CloudCover < 40 {
		cloudScore = 3
	}

	moonScore := 0
	if moonIllum > 0.6 {
		moonScore = 1
	}
	auroraScore := int(math.Min(10, math.Round(float64(latScore+cloudScore-moonScore))))

	// Stars Rating (Fixed missing parenthesis)
	starsRating := int(math.Min(5, math.Round(float64(
		int((100-c.CloudCover)/20) +
			func() int { if c.RelativeHumidity2m < 50 { return 1 }; return 0 }() +
			func() int { if moonIllum < 0.3 { return 1 }; return 0 }(), // Added missing paren here
	))))

	// Golden Hour Score
	goldenScore := int(math.Min(100, math.Round(float64(
		func() int {
			if c.CloudCover > 10 && c.CloudCover < 60 { return 50 }
			return 20
		}() +
		func() int {
			if c.RelativeHumidity2m < 55 { return 25 }
			return 15
		}() +
		func() int {
			if c.UVIndex > 0 { return 15 }
			return 0
		}(),
	))))

	// Fog Probability
	fogProb := int(math.Min(98, math.Round(float64(
		func() int {
			diff := c.Temperature2m - c.DewPoint2m
			if diff < 2 { return 45 }
			if diff < 4 { return 25 }
			return 5
		}() +
		func() int {
			if c.RelativeHumidity2m > 80 { return 35 }
			return 15
		}() +
		func() int {
			if c.WindSpeed10m < 8 { return 20 }
			return 5
		}(),
	))))

	// Mirage Score
	mirageScore := 10
	if c.Temperature2m > 30 && c.CloudCover < 30 {
		mirageScore = int(math.Min(95, 60+(c.Temperature2m-30)*3))
	} else if c.Temperature2m < 5 {
		mirageScore = int(math.Min(50, 20+math.Abs(c.Temperature2m)*3))
	}

	// Comfort Rating
	heatIndexC := c.Temperature2m + 0.5555*(6.11*math.Exp(5417.753*(1/273.16-1/(c.DewPoint2m+273.15)))-10)
	comfortRating := "Comfortable"
	switch {
	case heatIndexC < 10:
		comfortRating = "Cool"
	case heatIndexC < 22:
		comfortRating = "Comfortable"
	case heatIndexC < 28:
		comfortRating = "Warm"
	case heatIndexC < 35:
		comfortRating = "Hot"
	default:
		comfortRating = "Dangerous"
	}

	// Lightning Risk
	lightningRisk := 0
	if c.WeatherCode >= 95 && c.CloudCover > 60 {
		lightningRisk = int(math.Min(90, (c.RelativeHumidity2m-50)*1.5+c.WindSpeed10m*0.5))
	}

	// Pollen Index
	pollenIndex := int(math.Round(func() float64 {
		base := 10.0
		if c.Temperature2m > 15 { base += 30 }
		if c.RelativeHumidity2m < 60 { base += 20 }
		if c.WindSpeed10m < 15 { base += 15 }
		return base
	}()))

	precipProb := c.Precipitation
	if precipProb == 0 && len(dailyData) > 0 {
		precipProb = dailyData[0].PrecipitationMax
	}

	featureLabels["rainbow"].SetText(bestRainbow)
	featureLabels["rainbowDetail"].SetText(bestRainbowDetail)
	featureLabels["rainbowBadge"].SetText(bestRainbowBadge)

	featureLabels["aurora"].SetText(fmt.Sprintf("%d/10", auroraScore))
	featureLabels["auroraDetail"].SetText(fmt.Sprintf("Lat %.1f° · cloud %.0f%%", math.Abs(lat), c.CloudCover))
	featureLabels["auroraBadge"].SetText(func() string {
		if auroraScore > 6 {
			return "Good chance"
		}
		return "Low"
	}())

	featureLabels["stars"].SetText(fmt.Sprintf("%d/5", starsRating))
	featureLabels["starsDetail"].SetText(fmt.Sprintf("Cloud %.0f%% · moon %.0f%%", c.CloudCover, moonIllum*100))
	featureLabels["starsBadge"].SetText(func() string {
		if starsRating > 3 {
			return "Great"
		}
		return "Poor"
	}())

	featureLabels["golden"].SetText(fmt.Sprintf("%d%%", goldenScore))
	featureLabels["goldenDetail"].SetText("Cloud pattern optimal")
	featureLabels["goldenBadge"].SetText("Photo score")

	featureLabels["fog"].SetText(fmt.Sprintf("%d%%", fogProb))
	featureLabels["fogDetail"].SetText(fmt.Sprintf("Spread %.1f°C", c.Temperature2m-c.DewPoint2m))
	featureLabels["fogBadge"].SetText("Dew point")

	featureLabels["mirage"].SetText(fmt.Sprintf("%d%%", mirageScore))
	featureLabels["mirageDetail"].SetText(func() string {
		if c.Temperature2m > 30 {
			return "Hot surface"
		}
		if c.Temperature2m < 5 {
			return "Cold inversion"
		}
		return "Low gradient"
	}())
	featureLabels["mirageBadge"].SetText(func() string {
		if mirageScore > 40 {
			return "Visible"
		}
		return ""
	}())

	featureLabels["lightning"].SetText(func() string {
		if lightningRisk > 0 {
			return fmt.Sprintf("%d%%", lightningRisk)
		}
		return "None"
	}())
	featureLabels["lightningDetail"].SetText(func() string {
		if lightningRisk > 30 {
			return "Seek shelter"
		}
		return "Safe"
	}())
	featureLabels["lightningBadge"].SetText(func() string {
		if lightningRisk > 40 {
			return "Alert"
		}
		return ""
	}())

	featureLabels["heat"].SetText(fmt.Sprintf("%.0f°C", heatIndexC))
	featureLabels["heatDetail"].SetText(fmt.Sprintf("Feels %s", comfortRating))
	featureLabels["heatBadge"].SetText(comfortRating)

	featureLabels["pollen"].SetText(fmt.Sprintf("%d%%", pollenIndex))
	featureLabels["pollenDetail"].SetText("Temp & wind based")
	featureLabels["pollenBadge"].SetText("Allergy")

	featureLabels["precip"].SetText(fmt.Sprintf("%.0f%%", precipProb))
	featureLabels["precipDetail"].SetText("Next hour")
	featureLabels["precipBadge"].SetText("Rain")
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
			sunAngleStr = "🌙 Night"
		case sunAngle < 8:
			sunAngleStr = fmt.Sprintf("%.1f° 🌄", sunAngle)
		case sunAngle < 20:
			sunAngleStr = fmt.Sprintf("%.1f° 📸", sunAngle)
		case sunAngle < 35:
			sunAngleStr = fmt.Sprintf("%.1f° 🌈", sunAngle)
		case sunAngle < 42:
			sunAngleStr = fmt.Sprintf("%.1f° ⚠️", sunAngle)
		default:
			sunAngleStr = fmt.Sprintf("%.1f° 🔥", sunAngle)
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
	
	dailyTable.Clear()
	now := time.Now()
	
	for _, day := range dailyData {
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
		dailyTable.SetCell(4, row, fmt.Sprintf("%.0f%%", day.PrecipitationMax))
		
		// Store the date string to be used for selection parsing
		_ = day.Date.Format("2006-01-02") 
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
	
	// Update all displays
	updateCurrentWeather()
	updateFeatures()
	updateHourlyTable()
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
	searchEdit.SetText("")
	searchCombo.SetVisible(false)
	currentLat = selected.Lat
	currentLon = selected.Lon
	locationName = selected.Name
	countryName = selected.Country
	
	// Reset selected day when changing location
	selectedDayData = nil
	selectedHourlyData = nil
	selectedDate = "today"
	
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

func createUI() {
	windowFont, _ := wui.NewFont(wui.FontDesc{Name: "Tahoma", Height: -11})
	mainWindow = wui.NewWindow()
	mainWindow.SetFont(windowFont)
	
	mainWindow.SetInnerSize(650, 420)
	mainWindow.SetPosition(200, 70)
	mainWindow.SetResizable(false)
	mainWindow.SetHasMaxButton(false)
	mainWindow.SetTitle("Weather Pro")

	// Header row
	logo := wui.NewLabel()
	logo.SetBounds(10, 10, 120, 24)
	logo.SetText("🌤️ Weather Pro")
	logoFont, _ := wui.NewFont(wui.FontDesc{Name: "Tahoma", Height: -14, Bold: true})
	logo.SetFont(logoFont)
	mainWindow.Add(logo)

	searchEdit = wui.NewEditLine()
	searchEdit.SetBounds(405, 12, 140, 24)
	searchEdit.SetText("")
	searchEdit.SetOnTextChange(onSearchEditChange)
	mainWindow.Add(searchEdit)

	searchCombo = wui.NewComboBox()
	searchCombo.SetBounds(405, 38, 140, 100)
	searchCombo.SetVisible(false)
	searchCombo.SetOnChange(onSearchComboChange)
	mainWindow.Add(searchCombo)

	buttonSearch := wui.NewButton()
	buttonSearch.SetBounds(550, 12, 55, 24)
	buttonSearch.SetText("Search")
	buttonSearch.SetOnClick(func() { onSearchEditChange() })
	mainWindow.Add(buttonSearch)

	btnGeo = wui.NewButton()
	btnGeo.SetBounds(610, 12, 28, 24)
	btnGeo.SetText("📍")
	btnGeo.SetOnClick(onGeoClick)
	mainWindow.Add(btnGeo)

	// Current weather section
	labelMainIcon = wui.NewLabel()
	labelMainIcon.SetBounds(10, 45, 50, 50)
	fontIcon, _ := wui.NewFont(wui.FontDesc{Name: "Segoe UI Emoji", Height: 30})
	labelMainIcon.SetFont(fontIcon)
	labelMainIcon.SetText("☀️")
	mainWindow.Add(labelMainIcon)

	labelMainTemp = wui.NewLabel()
	labelMainTemp.SetBounds(60, 45, 80, 50)
	fontTemp, _ := wui.NewFont(wui.FontDesc{Name: "Tahoma", Height: -28, Bold: true})
	labelMainTemp.SetFont(fontTemp)
	labelMainTemp.SetText("--°")
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
	labelDate.SetBounds(150, 65, 200, 16)
	labelDate.SetText("")
	mainWindow.Add(labelDate)

	labelDesc = wui.NewLabel()
	labelDesc.SetBounds(150, 85, 200, 16)
	labelDesc.SetText("--")
	mainWindow.Add(labelDesc)

	// Meta items (Moved to the top right to save vertical space)
	metaItems := []struct{ cap, id string }{
		{"💧 Hum", "hum"},
		{"💨 Wind", "wind"},
		{"🔽 Pres", "pres"},
		{"☀️ UV", "uv"},
	}
	
	for i, m := range metaItems {
		col := i % 2
		row := i / 2
		cx := 405 + col*110 // Moved X coordinate to the right side
		cy := 60 + row*25   // Tucked under the search bar

		l := wui.NewLabel()
		l.SetBounds(cx, cy, 50, 18)
		l.SetText(m.cap)
		mainWindow.Add(l)

		val := wui.NewLabel()
		val.SetBounds(cx+50, cy, 60, 18)
		val.SetText("--")
		mainWindow.Add(val)

		switch m.id {
		case "hum":
			labelHumidity = val
		case "wind":
			labelWind = val
		case "pres":
			labelPressure = val
		case "uv":
			labelUV = val
		}
	}

	// Feature cards
	featureLabels = make(map[string]*wui.Label)
	cardDefs := []struct {
		name, icon, title string
		x, y0             int
	}{
		{"rainbow", "🌈", "Rainbow", 10, 120}, // Adjusted Y start
		{"aurora", "🌌", "Aurora", 135, 120},
		{"stars", "⭐", "Stargazing", 260, 120},
		{"golden", "🌅", "Golden hr", 385, 120},
		{"fog", "🌫️", "Fog risk", 510, 120},
		{"mirage", "💧", "Mirage", 10, 185}, // Adjusted Y start
		{"lightning", "⚡", "Lightning", 135, 185},
		{"heat", "🌡️", "Heat idx", 260, 185},
		{"pollen", "🌻", "Pollen", 385, 185},
		{"precip", "☂️", "Precip", 510, 185},
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

	// Hourly table
	labelHourly := wui.NewLabel()
	labelHourly.SetBounds(10, 255, 120, 16)
	labelHourly.SetText("⌛ Hourly forecast")
	fontTable, _ := wui.NewFont(wui.FontDesc{Name: "Tahoma", Height: -12, Bold: true})
	labelHourly.SetFont(fontTable)
	mainWindow.Add(labelHourly)

	hourlyTable = wui.NewStringTable("🕒 Hour", "Icon", "Temp", "Rainbow", "Sun Angle")
	hourlyTable.SetBounds(10, 275, 310, 135)
	mainWindow.Add(hourlyTable)

	// Daily table
	labelDaily := wui.NewLabel()
	labelDaily.SetBounds(330, 255, 150, 16)
	labelDaily.SetText("📅 14-day weather")
	labelDaily.SetFont(fontTable)
	mainWindow.Add(labelDaily)

	// Separated Day and Date into two columns
	dailyTable = wui.NewStringTable("📅 Date", "✅ Day", "Icon", "High / Low", "Precip")
	dailyTable.SetBounds(330, 275, 310, 135)
	dailyTable.SetOnSelectionChange(onDailyTableSelection)
	mainWindow.Add(dailyTable)
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