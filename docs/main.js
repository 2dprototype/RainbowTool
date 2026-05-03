// ============ APP STATE ============
const state = {
	currentLat: 40.71,
	currentLon: -74.00,
	locationName: 'New York',
	countryName: 'US',
	tempUnit: 'celsius',
	weatherCache: null,
	timezoneOffsetHours: -4,
	selectedDayData: null,
	selectedHourlyData: null,
	selectedDate: 'today',
	selectedHourData: null,
	selectedHourlyRow: -1,
	selectedDailyRow: -1,
	extraDailyData: [],
	extraHourlyData: {},
	verifications: {},
	animFrame: 0,
	themeName: 'default',
	config: {
		colors: {
			skyDayDay: [135, 206, 235],
			skyNight: [5, 5, 20],
			skySunset: [255, 140, 0],
			groundGreen1: [34, 139, 34],
			groundGreen2: [46, 139, 87],
			groundSnow1: [240, 248, 255],
			groundSnow2: [255, 250, 250],
			groundDesert1: [237, 201, 175],
			groundDesert2: [210, 180, 140],
			groundStorm1: [25, 100, 25],
			groundStorm2: [35, 100, 50],
			cloudWhite: [255, 255, 255],
			cloudLightGray: [200, 200, 200],
			cloudDarkGray: [100, 100, 100],
			cloudStorm: [50, 50, 60],
			rainDrop: [150, 150, 200],
			snowDrop: [255, 255, 255],
			lightning: [255, 255, 0],
			windLine: [200, 200, 200],
			sunInner: [255, 255, 0],
			sunOuter: [255, 255, 150],
			moon: [200, 200, 220],
			moonCrater: [170, 170, 190],
			star: [255, 255, 255],
			canvasBackground: [200, 200, 200],
			rainbowColors: [
				[255, 0, 0], [255, 127, 0], [255, 255, 0],
				[0, 255, 0], [0, 0, 255], [75, 0, 130], [148, 0, 211]
			]
		}
	}
};

// ============ UTILITY FUNCTIONS ============
function rgbStr(c, alpha) {
	if (alpha) return `rgba(${c[0]},${c[1]},${c[2]},${alpha})`;
	return `rgb(${c[0]},${c[1]},${c[2]})`;
}

function weatherInfo(code) {
	const map = {
		0: ['☀️', 'Clear sky'],
		1: ['☀️', 'Mainly clear'],
		2: ['⛅', 'Partly cloudy'],
		3: ['☁️', 'Overcast'],
		45: ['🌫️', 'Fog'],
		48: ['🌫️', 'Fog'],
		51: ['🌧️', 'Light drizzle'],
		53: ['🌧️', 'Moderate drizzle'],
		55: ['🌧️', 'Dense drizzle'],
		61: ['🌦️', 'Slight rain'],
		63: ['🌧️', 'Moderate rain'],
		65: ['🌧️', 'Heavy rain'],
		71: ['❄️', 'Snow'],
		73: ['❄️', 'Snow'],
		75: ['❄️', 'Snow'],
		80: ['🌦️', 'Rain showers'],
		81: ['🌧️', 'Moderate showers'],
		82: ['🌧️', 'Violent showers'],
		95: ['⛈️', 'Thunderstorm'],
		96: ['⛈️', 'T-storm w/ hail'],
		99: ['⛈️', 'T-storm w/ hail']
	};
	return map[code] || ['☁️', 'Unknown'];
}

function formatTemp(c) {
	if (state.tempUnit === 'fahrenheit') {
		return `${Math.round(c * 9/5 + 32)}°F`;
	}
	return `${Math.round(c)}°C`;
}

function getSolarElevationUTC(lat, lon, utc) {
	const jd = utc.getTime() / 86400000 + 2440587.5;
	const jc = (jd - 2451545.0) / 36525.0;
	const rad = Math.PI / 180.0;

	const geomMeanLongSun = ((280.46646 + jc * (36000.76983 + jc * 0.0003032)) % 360 + 360) % 360;
	const geomMeanAnomSun = 357.52911 + jc * (35999.05029 - 0.0001537 * jc);
	const eccentEarthOrbit = 0.016708634 - jc * (0.000042037 + 0.0000001267 * jc);

	const sunEqOfCtr = Math.sin(geomMeanAnomSun * rad) * (1.914602 - jc * (0.004817 + 0.000014 * jc)) +
		Math.sin(2 * geomMeanAnomSun * rad) * (0.019993 - 0.000101 * jc) +
		Math.sin(3 * geomMeanAnomSun * rad) * 0.000289;

	const sunTrueLong = geomMeanLongSun + sunEqOfCtr;
	const sunAppLong = sunTrueLong - 0.00569 - 0.00478 * Math.sin((125.04 - 1934.136 * jc) * rad);
	const meanObliqEcliptic = 23.439291 - jc * (0.0130042 + jc * (0.00000016 - jc * 0.000000504));
	const obliqCorr = meanObliqEcliptic + 0.00256 * Math.cos((125.04 - 1934.136 * jc) * rad);

	const sunDeclin = Math.asin(Math.sin(obliqCorr * rad) * Math.sin(sunAppLong * rad)) * (180.0 / Math.PI);

	const y = Math.tan(obliqCorr / 2 * rad) * Math.tan(obliqCorr / 2 * rad);
	const eqOfTime = 4.0 * (y * Math.sin(2 * geomMeanLongSun * rad) -
		2 * eccentEarthOrbit * Math.sin(geomMeanAnomSun * rad) +
		4 * eccentEarthOrbit * y * Math.sin(geomMeanAnomSun * rad) * Math.cos(2 * geomMeanLongSun * rad) -
		0.5 * y * y * Math.sin(4 * geomMeanLongSun * rad) -
		1.25 * eccentEarthOrbit * eccentEarthOrbit * Math.sin(2 * geomMeanAnomSun * rad)) * (180.0 / Math.PI);

	const trueSolarTime = ((utc.getUTCHours() * 60 + utc.getUTCMinutes() + utc.getUTCSeconds() / 60 + eqOfTime + 4.0 * lon) % 1440 + 1440) % 1440;
	let hourAngle = trueSolarTime / 4.0 - 180.0;
	if (hourAngle < -180) hourAngle += 360;

	const sinAlt = Math.sin(lat * rad) * Math.sin(sunDeclin * rad) +
		Math.cos(lat * rad) * Math.cos(sunDeclin * rad) * Math.cos(hourAngle * rad);
	return Math.asin(sinAlt) * (180.0 / Math.PI);
}

function predictRainbow(hourlyData, lat, lon, tzOffsetHours, includePast) {
	const results = [];
	const now = new Date();

	for (let i = 0; i < Math.min(hourlyData.length, 48); i++) {
		const localTime = hourlyData[i].Time;
		if (!includePast && localTime < now) continue;

		const utcTime = new Date(localTime.getTime() - tzOffsetHours * 3600000);
		const sunElev = getSolarElevationUTC(lat, lon, utcTime);

		if (sunElev <= 0 || sunElev >= 42) continue;
		if (hourlyData[i].Temperature2m <= 0) continue;

		const code = hourlyData[i].WeatherCode;
		if ((code >= 71 && code <= 77) || (code >= 85 && code <= 86)) continue;

		let score = 0;
		const precip = hourlyData[i].Precipitation || 0;
		const precipProb = hourlyData[i].PrecipitationProb || 0;
		const radiation = hourlyData[i].DirectRadiation || 0;
		const cloud = hourlyData[i].CloudCover || 0;
		const vis = hourlyData[i].Visibility || 10000;
		const wind = hourlyData[i].WindSpeed10m || 0;
		const humidity = hourlyData[i].RelativeHumidity2m || 0;

		// Sun angle score
		if (sunElev >= 10 && sunElev <= 30) score += 25;
		else if (sunElev > 0 && sunElev < 10) score += 15 + sunElev;
		else score += 25 - ((sunElev - 30) * 1.5);

		// Precipitation score
		if (precip >= 0.1 && precip <= 5.0) score += 25;
		else if (precip > 5.0) score += 15;
		else if (precipProb > 30) score += 10;

		// Direct sunlight score
		if (radiation > 100) score += 30;
		else if (radiation > 20) score += 15;
		else if (cloud >= 30 && cloud <= 70) score += 10;
		else if (cloud < 30) score += 5;

		// Visibility
		if (vis > 10000) score += 10;
		else if (vis > 5000) score += 5;

		// Stability
		if (wind < 15) score += 5;
		if (humidity > 65) score += 5;

		// Penalties
		if (cloud > 95 && radiation < 50) score *= 0.1;
		if (precip === 0 && precipProb < 10) score *= 0;
		if (vis < 2000 || code === 45 || code === 48) score *= 0.1;

		const finalScore = Math.min(98, Math.round(score));
		if (finalScore > 10) {
			results.push({
				Time: localTime,
				Score: finalScore,
				SunElev: Math.round(sunElev),
				PrecipProb: precipProb
			});
		}
	}

	results.sort((a, b) => b.Score - a.Score);
	return results;
}

// ============ API CALLS ============
async function fetchWeatherData(lat, lon) {
	const url = `https://api.open-meteo.com/v1/forecast?latitude=${lat}&longitude=${lon}&current=temperature_2m,relative_humidity_2m,apparent_temperature,precipitation,weather_code,cloud_cover,wind_speed_10m,wind_direction_10m,uv_index,dew_point_2m,surface_pressure&hourly=precipitation_probability,precipitation,cloud_cover,temperature_2m,dew_point_2m,weather_code,relative_humidity_2m,wind_speed_10m,visibility,direct_radiation&daily=temperature_2m_max,temperature_2m_min,precipitation_probability_max,weather_code&timezone=auto&forecast_days=7&past_days=7`;
	const resp = await fetch(url);
	return await resp.json();
}

async function searchCity(query) {
	const url = `https://geocoding-api.open-meteo.com/v1/search?name=${encodeURIComponent(query)}&count=5`;
	const resp = await fetch(url);
	return await resp.json();
}

async function getUserLocation() {
	try {
		const resp = await fetch('http://ip-api.com/json/?fields=lat,lon,city,country');
		return await resp.json();
	} catch (e) {
		return { lat: 40.71, lon: -74.00, city: 'New York', country: 'US' };
	}
}

async function reverseGeocode(lat, lon) {
	try {
		const url = `https://nominatim.openstreetmap.org/reverse?format=json&lat=${lat}&lon=${lon}`;
		const resp = await fetch(url, { headers: { 'User-Agent': 'RainbowTool/1.0' } });
		const data = await resp.json();
		return data.address?.city || data.address?.town || data.address?.village || data.display_name || '';
	} catch (e) {
		return '';
	}
}

async function fetchDateData(lat, lon, dateStr, isHistorical) {
	const apiUrl = isHistorical ?
		'https://archive-api.open-meteo.com/v1/archive' :
		'https://api.open-meteo.com/v1/forecast';
	const dailyParams = isHistorical ?
		'temperature_2m_max,temperature_2m_min,precipitation_sum,weather_code' :
		'temperature_2m_max,temperature_2m_min,precipitation_probability_max,weather_code';
	const hourlyParams = isHistorical ?
		'precipitation,cloud_cover,temperature_2m,dew_point_2m,weather_code,relative_humidity_2m,wind_speed_10m,direct_radiation' :
		'precipitation_probability,precipitation,cloud_cover,temperature_2m,dew_point_2m,weather_code,relative_humidity_2m,wind_speed_10m,visibility,direct_radiation';

	const url = `${apiUrl}?latitude=${lat}&longitude=${lon}&start_date=${dateStr}&end_date=${dateStr}&daily=${dailyParams}&hourly=${hourlyParams}&timezone=auto`;
	const resp = await fetch(url);
	return await resp.json();
}

// ============ DATA EXTRACTION ============
function extractDailyData(w) {
	if (!w) return [];
	const data = [];
	for (let i = 0; i < w.daily.time.length; i++) {
		const date = new Date(w.daily.time[i] + 'T00:00:00');
		let precipMax = 0;
		if (i < (w.daily.precipitation_probability_max || []).length) {
			precipMax = w.daily.precipitation_probability_max[i];
		} else if (i < (w.daily.precipitation_sum || []).length && w.daily.precipitation_sum[i] > 0) {
			precipMax = 100;
		}
		data.push({
			Date: date,
			TemperatureMax: w.daily.temperature_2m_max[i],
			TemperatureMin: w.daily.temperature_2m_min[i],
			PrecipitationMax: precipMax,
			WeatherCode: w.daily.weather_code[i]
		});
	}
	return data;
}

function extractHourlyDataForDay(w, targetDate) {
	if (!w) return [];
	const data = [];
	const targetStr = targetDate.toISOString().split('T')[0];
	for (let i = 0; i < w.hourly.time.length; i++) {
		const hourTime = new Date(w.hourly.time[i]);
		if (hourTime.toISOString().split('T')[0] === targetStr) {
			const precipProb = i < (w.hourly.precipitation_probability || []).length ?
				w.hourly.precipitation_probability[i] : 0;
			const visibility = i < (w.hourly.visibility || []).length ?
				w.hourly.visibility[i] : 10000;
			data.push({
				Time: hourTime,
				Temperature2m: w.hourly.temperature_2m[i],
				WeatherCode: w.hourly.weather_code[i],
				PrecipitationProb: precipProb,
				Precipitation: w.hourly.precipitation ? w.hourly.precipitation[i] : 0,
				CloudCover: w.hourly.cloud_cover ? w.hourly.cloud_cover[i] : 0,
				RelativeHumidity2m: w.hourly.relative_humidity_2m ? w.hourly.relative_humidity_2m[i] : 0,
				WindSpeed10m: w.hourly.wind_speed_10m ? w.hourly.wind_speed_10m[i] : 0,
				Visibility: visibility,
				DirectRadiation: w.hourly.direct_radiation ? w.hourly.direct_radiation[i] : 0
			});
		}
	}
	return data;
}

function getAllDailyData() {
	const data = extractDailyData(state.weatherCache);
	data.push(...state.extraDailyData);
	data.sort((a, b) => a.Date - b.Date);
	return data;
}

function getHourlyDataForDay(d) {
	const dateStr = d.toISOString().split('T')[0];
	if (state.extraHourlyData[dateStr]) return state.extraHourlyData[dateStr];
	return extractHourlyDataForDay(state.weatherCache, d);
}

// ============ UI UPDATES ============
function updateCurrentWeather() {
	const now = new Date();

	if (state.selectedDayData && state.selectedDate !== 'today') {
		const [, desc] = weatherInfo(state.selectedDayData.WeatherCode);
		document.getElementById('labelMainIcon').textContent = weatherInfo(state.selectedDayData.WeatherCode)[0];
		document.getElementById('labelMainTemp').textContent = formatTemp(state.selectedDayData.TemperatureMax);
		document.getElementById('labelMainFeels').textContent =
			`High / Low: ${formatTemp(state.selectedDayData.TemperatureMax)} / ${formatTemp(state.selectedDayData.TemperatureMin)}`;
		document.getElementById('labelDesc').textContent = desc;
		document.getElementById('labelDate').textContent = state.selectedDayData.Date.toLocaleDateString('en-US', {
			weekday: 'short',
			year: 'numeric',
			month: 'short',
			day: 'numeric'
		});
	} else if (state.weatherCache) {
		const c = state.weatherCache.current;
		const [, desc] = weatherInfo(c.weather_code);
		document.getElementById('labelMainIcon').textContent = weatherInfo(c.weather_code)[0];
		document.getElementById('labelMainTemp').textContent = formatTemp(c.temperature_2m);
		document.getElementById('labelMainFeels').textContent = `Feels ${formatTemp(c.apparent_temperature)}`;
		document.getElementById('labelDesc').textContent = desc;
		document.getElementById('labelDate').textContent = now.toLocaleDateString('en-US', {
			weekday: 'short',
			year: 'numeric',
			month: 'short',
			day: 'numeric'
		});
	}

	let loc = state.locationName;
	if (state.countryName) loc += ', ' + state.countryName;
	document.getElementById('labelLocation').textContent = loc;
}

function updateFeatures() {
	if (!state.weatherCache) return;

	let hourlyData;
	if (state.selectedHourlyData && state.selectedHourlyData.length > 0) {
		hourlyData = state.selectedHourlyData;
	} else {
		hourlyData = getHourlyDataForDay(new Date());
	}

	const lat = state.weatherCache.latitude;
	const lon = state.weatherCache.longitude;
	const tzOff = (state.weatherCache.utc_offset_seconds || 0) / 3600;
	const includePast = !!(state.selectedDayData && state.selectedDate !== 'today');

	const rainbowPreds = predictRainbow(hourlyData, lat, lon, tzOff, includePast);

	let bestRainbow = 'None';
	let bestRainbowDetail = 'No rain/sun angle';
	let bestRainbowBadge = 'Unlikely';

	if (rainbowPreds.length > 0) {
		bestRainbow = `${rainbowPreds[0].Score}%`;
		bestRainbowDetail = `Best ${rainbowPreds[0].Time.toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit' })} · sun ${rainbowPreds[0].SunElev}°`;
		bestRainbowBadge = 'View window';
	}

	document.getElementById('valRainbow').textContent = bestRainbow;
	document.getElementById('valRainbowDetail').textContent = bestRainbowDetail;
	document.getElementById('valRainbowBadge').textContent = bestRainbowBadge;

	let sunElevVal = '--';
	let sunElevDetail = 'Night';
	if (rainbowPreds.length > 0) {
		sunElevVal = `${rainbowPreds[0].SunElev}°`;
		if (rainbowPreds[0].SunElev > 42) sunElevDetail = 'Too high (>42°)';
		else if (rainbowPreds[0].SunElev <= 0) sunElevDetail = 'Below horizon';
		else sunElevDetail = 'Optimal angle';
	}
	document.getElementById('valSun').textContent = sunElevVal;
	document.getElementById('valSunDetail').textContent = sunElevDetail;
	document.getElementById('valSunBadge').textContent = 'Angle';

	const c = state.weatherCache.current;
	document.getElementById('valPrecip').textContent = `${Math.round(c.precipitation || 0)}%`;
	document.getElementById('valPrecipDetail').textContent = 'Required for bows';
	document.getElementById('valPrecipBadge').textContent = 'Moisture';
	document.getElementById('valCloud').textContent = `${Math.round(c.cloud_cover || 0)}%`;
	document.getElementById('valCloudDetail').textContent = 'Need < 95%';
	document.getElementById('valCloudBadge').textContent = 'Blockage';
}

function updateHourlyTable() {
	if (!state.weatherCache) return;

	let hourlyData;
	if (state.selectedHourlyData && state.selectedHourlyData.length > 0) {
		hourlyData = state.selectedHourlyData;
	} else {
		hourlyData = extractHourlyDataForDay(state.weatherCache, new Date());
	}

	const tzOff = (state.weatherCache.utc_offset_seconds || 0) / 3600;
	const lat = state.weatherCache.latitude;
	const lon = state.weatherCache.longitude;

	const rainbowPreds = predictRainbow(hourlyData, lat, lon, tzOff, true);
	const rainbowScores = {};
	rainbowPreds.forEach(rp => { rainbowScores[rp.Time.getTime()] = rp.Score; });

	const sunnyEmoji = '☀️';
	const dawnEmoji = '🌤️';
	const nightEmoji = '🌙';
	const tbody = document.querySelector('#hourlyTable tbody');
	const checkVerified = document.getElementById('checkVerified');

	// Store current verification state for selected row
	const currentSelectedTs = state.selectedHourData ? state.selectedHourData.Time.getTime() : null;

	tbody.innerHTML = '';

	hourlyData.forEach((hour, i) => {
		const utcTime = new Date(hour.Time.getTime() - tzOff * 3600000);
		const sunElev = Math.round(getSolarElevationUTC(lat, lon, utcTime) * 10) / 10;

		let sunAngleStr = `${sunElev.toFixed(1)}°`;
		if (sunElev <= 0) sunAngleStr = `🌙 ${sunElev.toFixed(1)}°`;
		else if (sunElev < 8) sunAngleStr = `🌤️ ${sunElev.toFixed(1)}°`;
		else sunAngleStr = `☀️ ${sunElev.toFixed(1)}°`;

		const score = rainbowScores[hour.Time.getTime()];
		let rainStr = '-';
		if (score) rainStr = `${score}% 🌈`;

		const ts = hour.Time.getTime().toString();
		if (state.verifications[ts]) rainStr += ' ✅';

		const row = tbody.insertRow();
		row.dataset.index = i;
		row.dataset.ts = ts;
		if (i === state.selectedHourlyRow) {
			row.classList.add('selected');
			if (state.verifications[ts]) {
				checkVerified.checked = true;
			} else {
				checkVerified.checked = false;
			}
		}

		row.innerHTML = `
			<td>${hour.Time.toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit' })}</td>
			<td>${weatherInfo(hour.WeatherCode)[0]}</td>
			<td>${formatTemp(hour.Temperature2m)}</td>
			<td>${rainStr}</td>
			<td>${sunAngleStr}</td>
		`;

		row.addEventListener('click', () => selectHourlyRow(i, hourlyData));
	});
}

function selectHourlyRow(index, hourlyData) {
	state.selectedHourlyRow = index;
	if (index >= 0 && index < hourlyData.length) {
		state.selectedHourData = hourlyData[index];
		const ts = state.selectedHourData.Time.getTime().toString();
		document.getElementById('checkVerified').checked = !!state.verifications[ts];
	}
	updateHourlyTable();
	drawMainCanvas();
	drawAnalysisCanvas();
}

function updateDailyTable() {
	const dailyData = getAllDailyData();
	const tbody = document.querySelector('#dailyTable tbody');
	const now = new Date();
	const todayStr = now.toISOString().split('T')[0];

	tbody.innerHTML = '';

	dailyData.forEach((day, i) => {
		const dayHourly = getHourlyDataForDay(day.Date);
		const rainbows = predictRainbow(dayHourly, state.currentLat, state.currentLon, state.timezoneOffsetHours, true);

		let rainbowStr = '-';
		if (rainbows.length > 0) rainbowStr = `${rainbows[0].Score}% 🌈`;

		const isToday = day.Date.toISOString().split('T')[0] === todayStr;
		let dayStr = day.Date.toLocaleDateString('en-US', { weekday: 'short' });
		if (isToday) dayStr = 'Today';

		const row = tbody.insertRow();
		row.dataset.index = i;
		if (i === state.selectedDailyRow) row.classList.add('selected');

		row.innerHTML = `
			<td>${day.Date.toLocaleDateString('en-US', { day: '2-digit', month: '2-digit', year: '2-digit' })}</td>
			<td>${dayStr}</td>
			<td>${weatherInfo(day.WeatherCode)[0]}</td>
			<td>${formatTemp(day.TemperatureMax)} / ${formatTemp(day.TemperatureMin)}</td>
			<td>${rainbowStr}</td>
		`;

		row.addEventListener('click', () => selectDailyRow(i, day, dailyData));
	});
}

function selectDailyRow(index, day, dailyData) {
	state.selectedDailyRow = index;
	state.selectedDate = day.Date.toISOString().split('T')[0];
	const todayStr = new Date().toISOString().split('T')[0];

	if (state.selectedDate === todayStr) {
		state.selectedDayData = null;
		state.selectedHourlyData = null;
		state.selectedDate = 'today';
	} else {
		state.selectedDayData = day;
		state.selectedHourlyData = getHourlyDataForDay(day.Date);
	}

	state.selectedHourData = null;
	state.selectedHourlyRow = -1;

	updateCurrentWeather();
	updateFeatures();
	updateHourlyTable();
	updateDailyTable();
	drawMainCanvas();
	drawAnalysisCanvas();
}

function updateVerifiedTable() {
	const tbody = document.querySelector('#verifiedTable tbody');
	tbody.innerHTML = '';

	const list = Object.values(state.verifications);
	list.sort((a, b) => b.Timestamp - a.Timestamp);

	list.forEach(v => {
		const row = tbody.insertRow();
		row.innerHTML = `
			<td>${v.Date}</td>
			<td>${v.Location}</td>
			<td>${v.Lat.toFixed(2)}</td>
			<td>${v.Lon.toFixed(2)}</td>
			<td>${v.Icon}</td>
			<td>${v.Temp.toFixed(1)}°C</td>
		`;
		row.addEventListener('click', () => selectVerifiedRow(v));
	});
}

function selectVerifiedRow(v) {
	state.currentLat = v.Lat;
	state.currentLon = v.Lon;
	state.locationName = v.Location;
	state.countryName = '';
	document.getElementById('editLat').value = v.Lat.toFixed(4);
	document.getElementById('editLon').value = v.Lon.toFixed(4);

	const t = new Date(v.Date.replace(/(\d{2})-(\d{2})-(\d{4})/, '$3-$2-$1'));
	updateData();
}

async function updateData() {
	document.getElementById('labelLocation').textContent = 'Loading weather data...';
	document.getElementById('editLat').value = state.currentLat.toFixed(4);
	document.getElementById('editLon').value = state.currentLon.toFixed(4);

	state.selectedDayData = null;
	state.selectedHourlyData = null;
	state.selectedDate = 'today';
	state.selectedHourData = null;
	state.selectedHourlyRow = -1;
	state.selectedDailyRow = -1;
	state.extraDailyData = [];
	state.extraHourlyData = {};

	try {
		state.weatherCache = await fetchWeatherData(state.currentLat, state.currentLon);
		state.timezoneOffsetHours = (state.weatherCache.utc_offset_seconds || 0) / 3600;

		updateCurrentWeather();
		updateFeatures();
		updateHourlyTable();
		updateDailyTable();
		updateVerifiedTable();
		drawMainCanvas();
		drawAnalysisCanvas();
	} catch (e) {
		document.getElementById('labelLocation').textContent = 'Error loading weather data';
		console.error(e);
	}
}

// ============ CANVAS DRAWING ============
function drawMainCanvas() {
	const canvas = document.getElementById('mainCanvas');
	const ctx = canvas.getContext('2d');
	const w = canvas.width;
	const h = canvas.height;
	const colors = state.config.colors;

	// Clear
	ctx.fillStyle = rgbStr(colors.canvasBackground);
	ctx.fillRect(0, 0, w, h);

	if (!state.weatherCache) {
		ctx.fillStyle = '#fff';
		ctx.font = '14px Tahoma';
		ctx.fillText('Loading...', 10, 20);
		return;
	}

	const lat = state.weatherCache.latitude;
	const lon = state.weatherCache.longitude;
	const tzOff = (state.weatherCache.utc_offset_seconds || 0) / 3600;

	let targetHour;
	if (state.selectedHourData) {
		targetHour = state.selectedHourData;
	} else {
		let hourlyData;
		if (state.selectedHourlyData && state.selectedHourlyData.length > 0) {
			hourlyData = state.selectedHourlyData;
		} else {
			hourlyData = extractHourlyDataForDay(state.weatherCache, new Date());
		}

		const isToday = state.selectedDate === 'today' || !state.selectedDayData ||
			state.selectedDayData.Date.toISOString().split('T')[0] === new Date().toISOString().split('T')[0];

		if (isToday && hourlyData.length > 0) {
			const nowHour = new Date().getHours();
			targetHour = hourlyData.find(h => h.Time.getHours() === nowHour) || hourlyData[0];
		} else if (hourlyData.length > 0) {
			const rainbowPreds = predictRainbow(hourlyData, lat, lon, tzOff, true);
			const bestTime = rainbowPreds.length > 0 ? rainbowPreds[0].Time : hourlyData[0].Time;
			targetHour = hourlyData.find(h => h.Time.getTime() === bestTime.getTime()) || hourlyData[0];
		} else {
			return;
		}
	}

	const cloudCover = targetHour.CloudCover || 0;
	const precipProb = targetHour.PrecipitationProb || 0;
	const temp = targetHour.Temperature2m;
	const humidity = targetHour.RelativeHumidity2m || 0;
	const wind = targetHour.WindSpeed10m || 0;
	const code = targetHour.WeatherCode;

	const utcTime = new Date(targetHour.Time.getTime() - tzOff * 3600000);
	const sunElev = getSolarElevationUTC(lat, lon, utcTime);

	const isFog = code === 45 || code === 48;
	const isRain = (code >= 61 && code <= 67) || (code >= 80 && code <= 82);
	const isSnow = (code >= 71 && code <= 77) || (code >= 85 && code <= 86);
	const isHail = code === 96 || code === 99;
	const isThunderstorm = code >= 95;
	const isPrecipitating = (code >= 51 && code <= 57) || isRain || isSnow || isHail;
	const isBlizzard = isSnow && wind > 30;

	// Rainbow score
	let rainbowScore = 0;
	const preds = predictRainbow([targetHour], lat, lon, tzOff, true);
	if (preds.length > 0) rainbowScore = preds[0].Score;

	// Ground colors
	let groundColor1 = colors.groundGreen1;
	let groundColor2 = colors.groundGreen2;
	if (temp < 0 || isSnow) {
		groundColor1 = colors.groundSnow1;
		groundColor2 = colors.groundSnow2;
	} else if (temp > 30 && humidity < 30) {
		groundColor1 = colors.groundDesert1;
		groundColor2 = colors.groundDesert2;
	} else if (isRain || isThunderstorm) {
		groundColor1 = colors.groundStorm1;
		groundColor2 = colors.groundStorm2;
	}

	// Sky color
	let skyColor = colors.skyDayDay;
	if (sunElev < -5) skyColor = colors.skyNight;
	else if (sunElev < 10) skyColor = colors.skySunset;

	let cloudDarken = cloudCover * 0.8;
	if (isThunderstorm || isBlizzard) cloudDarken = 100;

	let skyR = Math.max(0, skyColor[0] - cloudDarken);
	let skyG = Math.max(0, skyColor[1] - cloudDarken);
	let skyB = Math.max(0, skyColor[2] - cloudDarken);

	if (isThunderstorm && (state.animFrame % 30 < 2)) {
		skyR = 255;
		skyG = 255;
		skyB = 255;
	}

	// Draw sky
	const horizonR = Math.min(255, skyR + 50);
	const horizonG = Math.min(255, skyG + 50);
	const horizonB = Math.min(255, skyB + 50);

	for (let y = 0; y < h; y += 4) {
		const ratio = y / h;
		const r = Math.floor(skyR * (1 - ratio) + horizonR * ratio);
		const g = Math.floor(skyG * (1 - ratio) + horizonG * ratio);
		const b = Math.floor(skyB * (1 - ratio) + horizonB * ratio);
		ctx.fillStyle = `rgb(${r},${g},${b})`;
		ctx.fillRect(0, y, w, 4);
	}

	// Stars
	if (sunElev < -5 && cloudCover < 50 && !isFog) {
		ctx.fillStyle = 'white';
		[[20, 20], [120, 30], [200, 15], [350, 40]].forEach(([sx, sy]) => {
			ctx.fillRect(sx, sy, 2, 2);
		});
	}

	// Sun/Moon
	if (!isBlizzard) {
		const sunX = 40;
		let sunY = h - 40 - sunElev * 8;
		sunY = Math.max(20, Math.min(h, sunY));

		if (sunElev >= -5) {
			const pulse = Math.sin(state.animFrame * 0.2) * 5;
			ctx.fillStyle = rgbStr(colors.sunOuter);
			ctx.beginPath();
			ctx.ellipse(sunX, sunY, 25 + pulse, 25 + pulse, 0, 0, Math.PI * 2);
			ctx.fill();
			ctx.fillStyle = rgbStr(colors.sunInner);
			ctx.beginPath();
			ctx.ellipse(sunX, sunY, 18, 18, 0, 0, Math.PI * 2);
			ctx.fill();
		} else {
			ctx.fillStyle = rgbStr(colors.moon);
			ctx.beginPath();
			ctx.ellipse(sunX, 20, 15, 15, 0, 0, Math.PI * 2);
			ctx.fill();
			ctx.fillStyle = rgbStr(colors.moonCrater);
			ctx.beginPath();
			ctx.ellipse(sunX + 3, 25, 4, 4, 0, 0, Math.PI * 2);
			ctx.fill();
			ctx.beginPath();
			ctx.ellipse(sunX + 8, 32, 5, 5, 0, 0, Math.PI * 2);
			ctx.fill();
		}
	}

	// Clouds
	if (cloudCover > 10) {
		let cloudColor = colors.cloudWhite;
		if (cloudCover > 50) cloudColor = colors.cloudLightGray;
		if (cloudCover > 80) cloudColor = colors.cloudDarkGray;
		if (isThunderstorm || isBlizzard) cloudColor = colors.cloudStorm;

		const numClouds = isBlizzard ? 10 : Math.floor(cloudCover / 10);
		ctx.fillStyle = rgbStr(cloudColor);

		for (let i = 0; i < numClouds; i++) {
			const speed = (i % 3 + 1) * 0.5 * (wind > 20 ? 2 : 1);
			const offset = Math.floor(state.animFrame * speed) % (w + 100) - 50;
			const cx = (i * 45 + offset + w + 100) % (w + 100) - 50;
			const cy = 10 + (i * 10) % 40;

			ctx.beginPath();
			ctx.ellipse(cx, cy, 35, 18, 0, 0, Math.PI * 2);
			ctx.fill();
			ctx.beginPath();
			ctx.ellipse(cx + 15, cy - 8, 30, 22, 0, 0, Math.PI * 2);
			ctx.fill();
			ctx.beginPath();
			ctx.ellipse(cx - 10, cy - 3, 25, 20, 0, 0, Math.PI * 2);
			ctx.fill();
		}
	}

	// Rainbow
	if (rainbowScore > 10 && sunElev > 0) {
		const cx = w / 2 + 50;
		const cy = h - 30;
		let radius = 150 + sunElev;
		if (radius > w / 2) radius = w / 2;

		colors.rainbowColors.forEach((col, i) => {
			const r = radius - i * 5;
			ctx.strokeStyle = rgbStr(col,100);
			ctx.lineWidth = 4;
			// Draw full half-circle arc from left (π) to right (0 or 2π)
			ctx.beginPath();
			ctx.arc(cx, cy, r, Math.PI, 2 * Math.PI);
			ctx.stroke();
		});
	}

	// Ground
	ctx.fillStyle = rgbStr(groundColor1);
	ctx.beginPath();
	ctx.ellipse(-50, h - 20, w / 2 + 100, 50, 0, 0, Math.PI * 2);
	ctx.fill();
	ctx.fillStyle = rgbStr(groundColor2);
	ctx.beginPath();
	ctx.ellipse(w / 2 - 50, h - 30, w / 2 + 100, 75, 0, 0, Math.PI * 2);
	ctx.fill();

	// Fog
	if (isFog) {
		ctx.fillStyle = rgbStr(colors.cloudLightGray);
		for (let i = 0; i < 6; i++) {
			const drift = Math.floor(state.animFrame * 0.5) % (w + 150) - 75;
			const fx = (i * 70 + drift + w + 150) % (w + 150) - 75;
			const fy = h - 70 + (i * 10) % 30;
			ctx.beginPath();
			ctx.ellipse(fx, fy, 90, 25, 0, 0, Math.PI * 2);
			ctx.fill();
		}
	}

	// Precipitation
	const windOffset = Math.floor(wind / 3);
	if (isPrecipitating) {
		let dropColor = colors.rainDrop;
		let numDrops = 40;

		if ((code >= 51 && code <= 57)) { numDrops = 20;
			dropColor = colors.cloudLightGray; } else if (isRain && (code === 65 || code === 82)) { numDrops =
			150; } else if (isSnow) {
			dropColor = colors.snowDrop;
			numDrops = 80;
			if (code === 75 || code === 86) numDrops = 200;
			if (isBlizzard) numDrops = 350;
		} else if (isHail) {
			numDrops = 60;
		}

		ctx.fillStyle = rgbStr(dropColor);
		ctx.strokeStyle = rgbStr(dropColor);

		for (let i = 0; i < numDrops; i++) {
			let x = (i * 67) % w;

			if (isSnow) {
				const fallSpeed = isBlizzard ? (i % 3 + 3) * 3 : (i % 2 + 1) * 2;
				const sy = (i * 17 + state.animFrame * fallSpeed) % h;
				const drift = Math.sin(state.animFrame * 0.05 + i) * 10 + wind;
				x = ((x + Math.floor(drift) + w) % w + w) % w;
				const size = isBlizzard ? 1 + (i % 2) : (i % 2) + 2;
				ctx.fillRect(x, sy, size, size);
			} else if (isHail) {
				const fallSpeed = (i % 2 + 4) * 5;
				const hy = (i * 17 + state.animFrame * fallSpeed) % h;
				x = ((x + windOffset + w) % w + w) % w;
				ctx.beginPath();
				ctx.ellipse(x, hy, 2, 2, 0, 0, Math.PI * 2);
				ctx.fill();
			} else {
				const fallSpeed = (code === 65 || code === 82) ? (i % 3 + 3) * 5 + 5 : (i % 3 + 3) * 5;
				const length = (code >= 51 && code <= 57) ? 3 : fallSpeed;
				const ry = (i * 23 + state.animFrame * fallSpeed) % h;
				x = ((x + windOffset + w) % w + w) % w;

				const splashY = h - 30 + (i % 15);
				if (ry + length >= splashY) {
					ctx.strokeStyle = rgbStr(colors.cloudLightGray);
					ctx.beginPath();
					ctx.moveTo(x, splashY);
					ctx.lineTo(x - 3, splashY - 4);
					ctx.stroke();
					ctx.beginPath();
					ctx.moveTo(x, splashY);
					ctx.lineTo(x + 2, splashY - 3);
					ctx.stroke();
					ctx.strokeStyle = rgbStr(dropColor);
				} else {
					ctx.beginPath();
					ctx.moveTo(x, ry);
					ctx.lineTo(x - windOffset, ry + length);
					ctx.stroke();
				}
			}
		}
	}

	// Lightning
	if (isThunderstorm && state.animFrame % 30 < 2) {
		ctx.strokeStyle = rgbStr(colors.lightning);
		ctx.lineWidth = 2;
		const lx = w / 2 + ((state.animFrame * 7) % 100) - 50;
		ctx.beginPath();
		ctx.moveTo(lx, 20);
		ctx.lineTo(lx - 15, 60);
		ctx.lineTo(lx + 10, 75);
		ctx.lineTo(lx - 30, h - 40);
		ctx.stroke();
	}

	// Wind lines
	if (wind > 20 && !isSnow) {
		ctx.strokeStyle = rgbStr(colors.windLine);
		ctx.lineWidth = 1;
		const o1 = (state.animFrame * Math.floor(wind / 4)) % w;
		const o2 = (state.animFrame * Math.floor(wind / 3)) % w;
		ctx.beginPath();
		ctx.moveTo(o1, h - 50);
		ctx.lineTo(o1 + 30, h - 50);
		ctx.stroke();
		ctx.beginPath();
		ctx.moveTo((o2 + w / 2) % w, h - 80);
		ctx.lineTo((o2 + w / 2) % w + 40, h - 80);
		ctx.stroke();
	}

	// Grain effect
	ctx.fillStyle = 'rgba(20,20,25,0.3)';
	for (let i = 0; i < 300; i++) {
		const nx = (i * 73 + state.animFrame * 13) % w;
		const ny = (i * 97 + state.animFrame * 29) % h;
		ctx.fillRect(nx, ny, 1, 1);
	}

	// Info text
	ctx.fillStyle = '#fff';
	ctx.font = '11px Tahoma';
	ctx.fillText(
		`${targetHour.Time.toLocaleDateString('en-US', { weekday: 'short' })} ${targetHour.Time.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit' })} | Temp: ${temp.toFixed(1)}° | Wind: ${wind.toFixed(1)} | Rain Prob: ${precipProb.toFixed(0)}%`,
		5, h - 8
	);
}

function drawAnalysisCanvas() {
	const canvas = document.getElementById('analysisCanvas');
	const ctx = canvas.getContext('2d');
	const w = canvas.width;
	const h = canvas.height;

	ctx.fillStyle = 'rgb(30,30,40)';
	ctx.fillRect(0, 0, w, h);

	let hourlyData;
	if (state.selectedHourlyData && state.selectedHourlyData.length > 0) {
		hourlyData = state.selectedHourlyData;
	} else if (state.weatherCache) {
		hourlyData = extractHourlyDataForDay(state.weatherCache, new Date());
	}

	if (!hourlyData || hourlyData.length === 0) {
		ctx.fillStyle = '#ccc';
		ctx.font = '12px Tahoma';
		ctx.fillText('No data', 10, 15);
		return;
	}

	const graphType = parseInt(document.getElementById('analysisCombo').value);
	const lat = state.weatherCache ? state.weatherCache.latitude : state.currentLat;
	const lon = state.weatherCache ? state.weatherCache.longitude : state.currentLon;
	const tzOff = state.timezoneOffsetHours;

	// Grid
	ctx.strokeStyle = 'rgb(60,60,70)';
	ctx.lineWidth = 0.5;
	for (let i = 0; i <= 4; i++) {
		const y = i * (h - 20) / 4 + 10;
		ctx.beginPath();
		ctx.moveTo(0, y);
		ctx.lineTo(w, y);
		ctx.stroke();
	}

	const plotH = h - 20;

	switch (graphType) {
		case 0: { // Temp & Rain %
			let minT = 100,
				maxT = -100;
			hourlyData.forEach(d => {
				if (d.Temperature2m < minT) minT = d.Temperature2m;
				if (d.Temperature2m > maxT) maxT = d.Temperature2m;
			});
			if (maxT - minT < 5) { maxT = minT + 5; }
			minT -= 2;
			maxT += 2;

			ctx.strokeStyle = 'rgb(255,100,100)';
			ctx.lineWidth = 2;
			ctx.beginPath();
			hourlyData.forEach((d, i) => {
				const x = i * w / (hourlyData.length - 1);
				const y = plotH - (d.Temperature2m - minT) / (maxT - minT) * plotH + 10;
				if (i === 0) ctx.moveTo(x, y);
				else ctx.lineTo(x, y);
			});
			ctx.stroke();

			ctx.strokeStyle = 'rgb(100,150,255)';
			ctx.lineWidth = 2;
			ctx.beginPath();
			hourlyData.forEach((d, i) => {
				const x = i * w / (hourlyData.length - 1);
				const y = plotH - (d.PrecipitationProb / 100) * plotH + 10;
				if (i === 0) ctx.moveTo(x, y);
				else ctx.lineTo(x, y);
			});
			ctx.stroke();

			ctx.fillStyle = '#ccc';
			ctx.font = '11px Tahoma';
			ctx.fillText('Environment: Temp (Red) / Rain% (Blue)', 5, 8);
			break;
		}

		case 1: { // Rainbow Score Trend
			const rainbows = predictRainbow(hourlyData, lat, lon, tzOff, true);
			const scores = {};
			rainbows.forEach(r => { scores[r.Time.getHours()] = r.Score; });

			ctx.strokeStyle = 'rgb(255,255,100)';
			ctx.lineWidth = 3;
			ctx.beginPath();
			hourlyData.forEach((d, i) => {
				const x = i * w / (hourlyData.length - 1);
				const score = scores[d.Time.getHours()] || 0;
				const y = plotH - (score / 100) * plotH + 10;
				if (i === 0) ctx.moveTo(x, y);
				else ctx.lineTo(x, y);
			});
			ctx.stroke();
			ctx.fillStyle = '#fff';
			ctx.font = '11px Tahoma';
			ctx.fillText('Rainbow Analysis: Probability Score (0-100%)', 5, 8);
			break;
		}

		case 2: { // Sun Elevation
			ctx.strokeStyle = 'rgb(255,180,50)';
			ctx.lineWidth = 2;
			ctx.beginPath();
			hourlyData.forEach((d, i) => {
				const utc = new Date(d.Time.getTime() - tzOff * 3600000);
				const elev = getSolarElevationUTC(lat, lon, utc);
				const x = i * w / (hourlyData.length - 1);
				const y = Math.max(10, plotH - (elev / 90) * plotH + 10);
				if (i === 0) ctx.moveTo(x, y);
				else ctx.lineTo(x, y);

				if (elev > 0 && elev < 42) {
					ctx.fillStyle = 'rgba(0,255,0,0.5)';
					ctx.fillRect(x - 2, h - 5, 4, 5);
				}
			});
			ctx.stroke();
			ctx.fillStyle = '#ccc';
			ctx.font = '11px Tahoma';
			ctx.fillText('Sun Analysis: Elevation Angle (Green = Potential Window)', 5, 8);
			break;
		}

		case 3: { // Cloud vs Precipitation
			ctx.strokeStyle = 'rgb(200,200,200)';
			ctx.lineWidth = 2;
			ctx.beginPath();
			hourlyData.forEach((d, i) => {
				const x = i * w / (hourlyData.length - 1);
				const y = plotH - (d.CloudCover / 100) * plotH + 10;
				if (i === 0) ctx.moveTo(x, y);
				else ctx.lineTo(x, y);
			});
			ctx.stroke();

			ctx.strokeStyle = 'rgb(0,120,255)';
			ctx.lineWidth = 2;
			ctx.beginPath();
			hourlyData.forEach((d, i) => {
				const x = i * w / (hourlyData.length - 1);
				const y = plotH - (d.PrecipitationProb / 100) * plotH + 10;
				if (i === 0) ctx.moveTo(x, y);
				else ctx.lineTo(x, y);
			});
			ctx.stroke();
			ctx.fillStyle = '#ccc';
			ctx.font = '11px Tahoma';
			ctx.fillText('Detailed: Cloud Cover (Gray) / Precipitation Prob (Blue)', 5, 8);
			break;
		}

		case 4: { // Wind Speed & Visibility
			let maxWind = 1;
			hourlyData.forEach(d => { if (d.WindSpeed10m > maxWind) maxWind = d.WindSpeed10m; });

			ctx.strokeStyle = 'rgb(150,255,150)';
			ctx.lineWidth = 2;
			ctx.beginPath();
			hourlyData.forEach((d, i) => {
				const x = i * w / (hourlyData.length - 1);
				const y = plotH - (d.WindSpeed10m / maxWind) * plotH + 10;
				if (i === 0) ctx.moveTo(x, y);
				else ctx.lineTo(x, y);
			});
			ctx.stroke();

			ctx.strokeStyle = 'rgb(200,150,255)';
			ctx.lineWidth = 2;
			ctx.beginPath();
			hourlyData.forEach((d, i) => {
				const x = i * w / (hourlyData.length - 1);
				const vis = d.Visibility || 10000;
				const y = plotH - (vis / 24000) * plotH + 10;
				if (i === 0) ctx.moveTo(x, y);
				else ctx.lineTo(x, y);
			});
			ctx.stroke();
			ctx.fillStyle = '#ccc';
			ctx.font = '11px Tahoma';
			ctx.fillText('Wind Speed (Green) / Visibility (Purple)', 5, 8);
			break;
		}

		case 5: { // Advanced Rainbow Analysis
			const rainbows = predictRainbow(hourlyData, lat, lon, tzOff, true);
			const scores = {};
			rainbows.forEach(r => { scores[r.Time.getHours()] = r.Score; });

			let maxPrecip = 0.1,
				maxRad = 1;
			hourlyData.forEach(d => {
				if (d.Precipitation > maxPrecip) maxPrecip = d.Precipitation;
				if (d.DirectRadiation > maxRad) maxRad = d.DirectRadiation;
			});

			// Rainbow window heat map
			hourlyData.forEach((d, i) => {
				const s = scores[d.Time.getHours()] || 0;
				if (s > 10) {
					const x = i * w / (hourlyData.length - 1);
					const intensity = Math.floor((s / 100) * 80);
					ctx.fillStyle = `rgb(${intensity},0,${Math.floor(intensity/2)})`;
					ctx.fillRect(x - 2, 10, 4, plotH);
				}
			});

			ctx.strokeStyle = 'rgb(0,150,255)';
			ctx.lineWidth = 1.5;
			ctx.beginPath();
			hourlyData.forEach((d, i) => {
				const x = i * w / (hourlyData.length - 1);
				const y = plotH - (d.Precipitation / maxPrecip) * plotH + 10;
				if (i === 0) ctx.moveTo(x, y);
				else ctx.lineTo(x, y);
			});
			ctx.stroke();

			ctx.strokeStyle = 'rgb(255,100,100)';
			ctx.lineWidth = 1.5;
			ctx.beginPath();
			hourlyData.forEach((d, i) => {
				const x = i * w / (hourlyData.length - 1);
				const y = plotH - (d.DirectRadiation / maxRad) * plotH + 10;
				if (i === 0) ctx.moveTo(x, y);
				else ctx.lineTo(x, y);
			});
			ctx.stroke();

			ctx.strokeStyle = 'rgb(255,255,0)';
			ctx.lineWidth = 2;
			ctx.beginPath();
			hourlyData.forEach((d, i) => {
				const x = i * w / (hourlyData.length - 1);
				const s = scores[d.Time.getHours()] || 0;
				const y = plotH - (s / 100) * plotH + 10;
				if (i === 0) ctx.moveTo(x, y);
				else ctx.lineTo(x, y);
			});
			ctx.stroke();

			ctx.fillStyle = '#fff';
			ctx.font = '11px Tahoma';
			ctx.fillText('ADV: Score(Yel), RainVol(Blu), Rad(Red)', 5, 8);
			break;
		}
	}
}

function showHelpModal() {
	document.getElementById('helpModal').classList.add('active');
}

function closeHelpModal() {
	document.getElementById('helpModal').classList.remove('active');
}

// ============ EVENT HANDLERS ============
function setupEventListeners() {
	// Lat/Lon update
	document.getElementById('btnUpdateLoc').addEventListener('click', () => {
		const lat = parseFloat(document.getElementById('editLat').value);
		const lon = parseFloat(document.getElementById('editLon').value);
		if (!isNaN(lat) && !isNaN(lon)) {
			state.currentLat = lat;
			state.currentLon = lon;
			state.locationName = `${lat.toFixed(2)}, ${lon.toFixed(2)}`;
			state.countryName = '';
			document.getElementById('searchEdit').value = state.locationName;
			reverseGeocode(lat, lon).then(name => {
				if (name) {
					state.locationName = name;
					document.getElementById('searchEdit').value = name;
				}
			});
			updateData();
		}
	});

	// Search
	const searchEdit = document.getElementById('searchEdit');
	const searchDropdown = document.getElementById('searchDropdown');
	let searchTimeout;

	searchEdit.addEventListener('input', () => {
		clearTimeout(searchTimeout);
		const q = searchEdit.value.trim();
		if (q.length < 2) {
			searchDropdown.style.display = 'none';
			return;
		}
		searchTimeout = setTimeout(async () => {
			try {
				const result = await searchCity(q);
				if (result.results && result.results.length > 0) {
					searchDropdown.innerHTML = result.results.map((r, i) => `
						<div class="search-dropdown-item" data-index="${i}">
							${r.name}${r.admin1 ? ', ' + r.admin1 : ''} (${r.country})
						</div>
					`).join('');
					searchDropdown.style.display = 'block';
					searchDropdown.querySelectorAll('.search-dropdown-item').forEach(item => {
						item.addEventListener('click', () => {
							const idx = parseInt(item.dataset.index);
							const r = result.results[idx];
							state.currentLat = r.latitude;
							state.currentLon = r.longitude;
							state.locationName = r.name;
							state.countryName = r.country;
							document.getElementById('editLat').value = r.latitude.toFixed(4);
							document.getElementById('editLon').value = r.longitude.toFixed(4);
							searchEdit.value = `${r.name}, ${r.country}`;
							searchDropdown.style.display = 'none';
							updateData();
						});
					});
				} else {
					searchDropdown.style.display = 'none';
				}
			} catch (e) {
				searchDropdown.style.display = 'none';
			}
		}, 300);
	});

	document.getElementById('btnSearch').addEventListener('click', () => {
		searchEdit.dispatchEvent(new Event('input'));
	});

	// Geo location
	document.getElementById('btnGeo').addEventListener('click', async () => {
		const btn = document.getElementById('btnGeo');
		btn.disabled = true;
		btn.textContent = '⏳';
		try {
			const loc = await getUserLocation();
			state.currentLat = loc.lat;
			state.currentLon = loc.lon;
			state.locationName = loc.city;
			state.countryName = loc.country;
			document.getElementById('editLat').value = loc.lat.toFixed(4);
			document.getElementById('editLon').value = loc.lon.toFixed(4);
			searchEdit.value = `${loc.city}, ${loc.country}`;
			updateData();
		} catch (e) {
			console.error(e);
		}
		btn.disabled = false;
		btn.textContent = '📍';
	});

	// Verified checkbox
	document.getElementById('checkVerified').addEventListener('change', function() {
		if (state.selectedHourData) {
			const ts = state.selectedHourData.Time.getTime().toString();
			if (this.checked) {
				const [icon] = weatherInfo(state.selectedHourData.WeatherCode);
				state.verifications[ts] = {
					Timestamp: ts,
					Date: state.selectedHourData.Time.toLocaleString('en-GB', {
						day: '2-digit',
						month: '2-digit',
						year: 'numeric',
						hour: '2-digit',
						minute: '2-digit',
						hour12: false
					}).replace(/\//g, '-'),
					Lat: state.currentLat,
					Lon: state.currentLon,
					Location: state.locationName,
					Temp: state.selectedHourData.Temperature2m,
					Icon: icon
				};
			} else {
				delete state.verifications[ts];
			}
			saveVerifications();
			updateHourlyTable();
			updateVerifiedTable();
		}
	});

	// Date search
	document.getElementById('btnDateSearch').addEventListener('click', async () => {
		const text = document.getElementById('dateSearchEdit').value;
		const parts = text.split('-');
		if (parts.length !== 3) {
			alert('Invalid date format. Use dd-mm-yyyy');
			return;
		}
		const t = new Date(`${parts[2]}-${parts[1]}-${parts[0]}T00:00:00`);
		if (isNaN(t.getTime())) {
			alert('Invalid date');
			return;
		}

		const dateStr = t.toISOString().split('T')[0];
		const allData = getAllDailyData();
		if (allData.some(d => d.Date.toISOString().split('T')[0] === dateStr)) {
			alert('Date already exists in table');
			return;
		}

		const btn = document.getElementById('btnDateSearch');
		btn.disabled = true;
		btn.textContent = '⏳';

		try {
			const isHistorical = t < new Date(Date.now() - 80 * 24 * 3600000);
			const w = await fetchDateData(state.currentLat, state.currentLon, dateStr, isHistorical);

			if (w.daily && w.daily.time && w.daily.time.length > 0) {
				const dayDataList = extractDailyData(w);
				if (dayDataList.length > 0) {
					const dayData = dayDataList[0];
					const hourlyData = extractHourlyDataForDay(w, t);
					state.extraDailyData.push(dayData);
					state.extraHourlyData[dateStr] = hourlyData;
					updateDailyTable();
					updateVerifiedTable();
				}
			} else {
				alert('No data found for this date.');
			}
		} catch (e) {
			alert('Failed to fetch data: ' + e.message);
		}
		btn.disabled = false;
		btn.textContent = 'Add Date';
	});

	// Analysis combo
	document.getElementById('analysisCombo').addEventListener('change', () => {
		drawAnalysisCanvas();
	});

	// Theme combo
	document.getElementById('themeCombo').addEventListener('change', function() {
		// In a full implementation, this would load theme JSON files
		// For now, just use default colors
	});

	// Modal close
	document.getElementById('helpModal').addEventListener('click', function(e) {
		if (e.target === this) closeHelpModal();
	});

	// Close search dropdown on outside click
	document.addEventListener('click', function(e) {
		if (!e.target.closest('.search-group')) {
			searchDropdown.style.display = 'none';
		}
	});
}

// ============ STORAGE ============
function saveVerifications() {
	localStorage.setItem('rainbow_verifications', JSON.stringify(state.verifications));
}

function loadVerifications() {
	const saved = localStorage.getItem('rainbow_verifications');
	if (saved) {
		try {
			state.verifications = JSON.parse(saved);
		} catch (e) {
			state.verifications = {};
		}
	}
}

// ============ INITIALIZATION ============
function init() {
	loadVerifications();
	setupEventListeners();

	// Set default date search value
	const now = new Date();
	document.getElementById('dateSearchEdit').value =
		`${String(now.getDate()).padStart(2,'0')}-${String(now.getMonth()+1).padStart(2,'0')}-${now.getFullYear()}`;

	// Load initial data
	getUserLocation().then(loc => {
		state.currentLat = loc.lat;
		state.currentLon = loc.lon;
		state.locationName = loc.city;
		state.countryName = loc.country;
		document.getElementById('editLat').value = loc.lat.toFixed(4);
		document.getElementById('editLon').value = loc.lon.toFixed(4);
		document.getElementById('searchEdit').value = `${loc.city}, ${loc.country}`;
		updateData();
	}).catch(() => {
		updateData();
	});

	// Animation loop
	function animate() {
		state.animFrame++;
		// Only redraw main canvas (analysis is on-demand)
		if (state.weatherCache) {
			drawMainCanvas();
		}
		requestAnimationFrame(animate);
	}
	animate();
}

init();