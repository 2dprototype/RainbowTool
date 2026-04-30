// Global state
let currentLat = 40.71;
let currentLon = -74.00;
let locationName = "New York";
let countryName = "US";
let tempUnit = "celsius";
let weatherCache = null;
let selectedDayData = null;
let selectedHourlyData = null;
let selectedDate = "today";
let timezoneOffsetHours = -4;
let searchItems = [];

// Helper functions
function formatTemp(c) {
	if (tempUnit === "fahrenheit") {
		return Math.round(c * 9/5 + 32) + "°F";
	}
	return Math.round(c) + "°C";
}

function weatherInfo(code) {
	const weatherMap = {
		0: ["<i class='bi bi-brightness-high-fill'></i>", "Clear sky"],
		1: ["<i class='bi bi-brightness-alt-high-fill'></i>", "Mainly clear"],
		2: ["<i class='bi bi-cloud-sun-fill'></i>", "Partly cloudy"],
		3: ["<i class='bi bi-cloud-fill'></i>", "Overcast"],
		45: ["<i class='bi bi-cloud-fog-fill'></i>", "Fog"],
		48: ["<i class='bi bi-cloud-fog-fill'></i>", "Fog"],
		51: ["<i class='bi bi-cloud-drizzle-fill'></i>", "Light drizzle"],
		53: ["<i class='bi bi-cloud-drizzle-fill'></i>", "Moderate drizzle"],
		55: ["<i class='bi bi-cloud-rain-fill'></i>", "Dense drizzle"],
		61: ["<i class='bi bi-cloud-rain-fill'></i>", "Slight rain"],
		63: ["<i class='bi bi-cloud-rain-heavy-fill'></i>", "Moderate rain"],
		65: ["<i class='bi bi-cloud-rain-heavy-fill'></i>", "Heavy rain"],
		71: ["<i class='bi bi-cloud-snow-fill'></i>", "Snow"],
		73: ["<i class='bi bi-cloud-snow-fill'></i>", "Snow"],
		75: ["<i class='bi bi-cloud-snow-fill'></i>", "Snow"],
		80: ["<i class='bi bi-cloud-rain-fill'></i>", "Rain showers"],
		81: ["<i class='bi bi-cloud-rain-heavy-fill'></i>", "Moderate showers"],
		82: ["<i class='bi bi-cloud-rain-heavy-fill'></i>", "Violent showers"],
		95: ["<i class='bi bi-cloud-lightning-rain-fill'></i>", "Thunderstorm"],
		96: ["<i class='bi bi-cloud-hail-fill'></i>", "T-storm w/ hail"],
		99: ["<i class='bi bi-cloud-hail-fill'></i>", "T-storm w/ hail"]
	};
	return weatherMap[code] || ["<i class='bi bi-cloud-fill'></i>", "Unknown"];
}

function getSolarElevationUTC(lat, lon, utc) {
	const dayOfYear = utc.getDate();
	const declination = 23.44 * Math.sin((2 * Math.PI / 365) * (dayOfYear - 81));
	const declRad = declination * Math.PI / 180;
	const latRad = lat * Math.PI / 180;
	const hourUTC = utc.getHours() + utc.getMinutes() / 60;
	const solarNoonUTC = 12 - (lon / 15);
	const hourAngle = (hourUTC - solarNoonUTC) * 15 * Math.PI / 180;
	const sinAlt = Math.sin(latRad) * Math.sin(declRad) + Math.cos(latRad) * Math.cos(declRad) * Math.cos(hourAngle);
	return Math.asin(Math.max(-1, Math.min(1, sinAlt))) * 180 / Math.PI;
}

function predictRainbow(hourlyData, lat, lon, tzOffsetHours, includePast) {
	const results = [];
	const now = new Date();
	
	for (let i = 0; i < Math.min(hourlyData.length, 48); i++) {
		const localTime = new Date(hourlyData[i].time);
		
		if (!includePast && localTime < now) continue;
		
		const utcTime = new Date(localTime.getTime() - tzOffsetHours * 3600000);
		const sunElev = getSolarElevationUTC(lat, lon, utcTime);
		
		if (sunElev <= 0 || sunElev >= 42) continue;
		
		const precipProb = hourlyData[i].precipProb;
		const cloud = hourlyData[i].cloudCover;
		
		let score = 0;
		
		// Sun elevation score
		if (sunElev > 5 && sunElev < 35) {
			score += 30 - Math.abs(sunElev - 20) * 0.5;
		} else {
			score += 15;
		}
		
		// Precipitation score
		if (precipProb >= 30 && precipProb <= 70) {
			score += 40;
		} else if (precipProb > 70) {
			score += 30;
		} else {
			score += precipProb * 0.5;
		}
		
		// Cloud cover score
		if (cloud >= 30 && cloud <= 70) {
			score += 30;
		} else if (cloud > 70 && cloud < 90) {
			score += 15;
		} else if (cloud >= 90) {
			score += 0;
		} else {
			score += cloud * 0.5;
		}
		
		// Penalties
		if (cloud > 95) score *= 0.1;
		if (precipProb < 10) score *= 0.1;
		
		const finalScore = Math.min(98, Math.round(score));
		if (finalScore > 10) {
			results.push({
				time: localTime,
				score: finalScore,
				sunElev: Math.round(sunElev),
				precipProb: precipProb
			});
		}
	}
	
	results.sort((a, b) => b.score - a.score);
	return results;
}

async function fetchWeather(lat, lon) {
	const url = `https://api.open-meteo.com/v1/forecast?latitude=${lat}&longitude=${lon}&current=temperature_2m,relative_humidity_2m,apparent_temperature,precipitation,weather_code,cloud_cover,wind_speed_10m,wind_direction_10m,uv_index,dew_point_2m,surface_pressure&hourly=precipitation_probability,cloud_cover,temperature_2m,dew_point_2m,weather_code,relative_humidity_2m&daily=temperature_2m_max,temperature_2m_min,precipitation_probability_max,weather_code&timezone=auto&forecast_days=7&past_days=7`;
	
	const response = await fetch(url);
	const data = await response.json();
	return data;
}

async function searchCity(q) {
	const url = `https://geocoding-api.open-meteo.com/v1/search?name=${q}&count=5`;
	const response = await fetch(url);
	const data = await response.json();
	if (data.results) {
		return data.results.map(r => ({
			name: r.name,
			lat: r.latitude,
			lon: r.longitude,
			country: r.country,
			admin1: r.admin1
		}));
	}
	return [];
}

async function getUserLocation() {
	return new Promise((resolve) => {
		if ("geolocation" in navigator) {
			navigator.geolocation.getCurrentPosition(
				async (position) => {
					const lat = position.coords.latitude;
					const lon = position.coords.longitude;
					// Reverse geocoding
					try {
						const response = await fetch(`https://nominatim.openstreetmap.org/reverse?lat=${lat}&lon=${lon}&format=json`);
						const data = await response.json();
						resolve({
							lat, lon,
							city: data.address?.city || data.address?.town || data.address?.village || "Unknown",
							country: data.address?.country || ""
						});
					} catch {
						resolve({ lat, lon, city: "Unknown", country: "" });
					}
				},
				() => resolve({ lat: 40.71, lon: -74.00, city: "New York", country: "US" })
			);
		} else {
			resolve({ lat: 40.71, lon: -74.00, city: "New York", country: "US" });
		}
	});
}

function extractDailyData(w) {
	if (!w) return [];
	const dailyData = [];
	for (let i = 0; i < w.daily.time.length; i++) {
		dailyData.push({
			date: new Date(w.daily.time[i]),
			tempMax: w.daily.temperature_2m_max[i],
			tempMin: w.daily.temperature_2m_min[i],
			precipMax: w.daily.precipitation_probability_max[i],
			weatherCode: w.daily.weather_code[i]
		});
	}
	return dailyData;
}

function extractHourlyDataForDay(w, targetDate) {
	if (!w) return [];
	const hourlyData = [];
	const targetStr = targetDate.toISOString().split('T')[0];
	
	for (let i = 0; i < w.hourly.time.length; i++) {
		const hourTime = new Date(w.hourly.time[i]);
		if (hourTime.toISOString().split('T')[0] === targetStr) {
			hourlyData.push({
				time: hourTime,
				temp: w.hourly.temperature_2m[i],
				weatherCode: w.hourly.weather_code[i],
				precipProb: w.hourly.precipitation_probability[i],
				cloudCover: w.hourly.cloud_cover[i],
				humidity: w.hourly.relative_humidity_2m[i]
			});
		}
	}
	return hourlyData;
}

function updateUI() {
	if (!weatherCache) return;
	
	// Current weather
	if (selectedDayData && selectedDate !== "today") {
		const [icon, desc] = weatherInfo(selectedDayData.weatherCode);
		document.getElementById("weatherIcon").innerHTML = icon;
		document.getElementById("mainTemp").innerText = formatTemp(selectedDayData.tempMax);
		document.getElementById("mainFeels").innerHTML = `High / Low: ${formatTemp(selectedDayData.tempMax)} / ${formatTemp(selectedDayData.tempMin)}`;
		document.getElementById("weatherDesc").innerText = desc;
		document.getElementById("date").innerText = selectedDayData.date.toLocaleDateString('en-US', { weekday: 'short', month: 'short', day: 'numeric', year: 'numeric' });
		document.getElementById("humidity").innerText = "--";
		document.getElementById("wind").innerText = "--";
		document.getElementById("pressure").innerText = "--";
		document.getElementById("uv").innerText = "--";
	} else {
		const c = weatherCache.current;
		const [icon, desc] = weatherInfo(c.weather_code);
		document.getElementById("weatherIcon").innerHTML = icon;
		document.getElementById("mainTemp").innerText = formatTemp(c.temperature_2m);
		document.getElementById("mainFeels").innerHTML = `Feels ${formatTemp(c.apparent_temperature)}`;
		document.getElementById("weatherDesc").innerText = desc;
		document.getElementById("date").innerText = new Date().toLocaleDateString('en-US', { weekday: 'short', month: 'short', day: 'numeric', year: 'numeric' });
		document.getElementById("humidity").innerText = Math.round(c.relative_humidity_2m) + "%";
		document.getElementById("wind").innerText = Math.round(c.wind_speed_10m) + " km/h";
		document.getElementById("pressure").innerText = Math.round(c.surface_pressure) + " hPa";
		document.getElementById("uv").innerText = c.uv_index.toFixed(1);
	}
	
	const loc = locationName + (countryName ? ", " + countryName : "");
	document.getElementById("location").innerText = loc;
	
	updateFeatures();
	updateHourlyTable();
	updateDailyTable();
}

function updateFeatures() {
	if (!weatherCache) return;
	
	let hourlyData = selectedHourlyData && selectedHourlyData.length > 0 ? selectedHourlyData : extractHourlyDataForDay(weatherCache, new Date());
	const c = weatherCache.current;
	const dailyData = extractDailyData(weatherCache);
	const lat = weatherCache.latitude;
	const lon = weatherCache.longitude;
	const tzOff = weatherCache.utc_offset_seconds / 3600;
	
	const includePast = (selectedDayData && selectedDate !== "today");
	const rainbowPreds = predictRainbow(hourlyData, lat, lon, tzOff, includePast);
	
	let bestRainbow = "None";
	let bestRainbowDetail = "No rain/sun angle";
	let bestRainbowBadge = "Unlikely";
	if (rainbowPreds.length > 0) {
		bestRainbow = rainbowPreds[0].score + "%";
		bestRainbowDetail = `Best ${rainbowPreds[0].time.toLocaleTimeString([], {hour:'2-digit', minute:'2-digit'})} · sun ${rainbowPreds[0].sunElev}°`;
		bestRainbowBadge = "View window";
	}
	
	// Moon phase
	const moonPhase = ((Date.now() - new Date(2000, 0, 6, 18, 14).getTime()) / (1000 * 3600 * 24)) / 29.53;
	const moonIllum = 0.5 * (1 - Math.cos(2 * Math.PI * moonPhase));
	
	// Aurora score
	const absLat = Math.abs(lat);
	const latScore = absLat > 55 ? 7 : (absLat > 48 ? 4 : 1);
	const cloudScore = c.cloud_cover < 40 ? 3 : 1;
	const moonScore = moonIllum > 0.6 ? 1 : 0;
	const auroraScore = Math.min(10, Math.round(latScore + cloudScore - moonScore));
	
	// Stars rating
	const starsRating = Math.min(5, Math.round(((100 - c.cloud_cover) / 20) + (c.relative_humidity_2m < 50 ? 1 : 0) + (moonIllum < 0.3 ? 1 : 0)));
	
	// Golden hour
	const goldenScore = Math.min(100, Math.round((c.cloud_cover > 10 && c.cloud_cover < 60 ? 50 : 20) + (c.relative_humidity_2m < 55 ? 25 : 15) + (c.uv_index > 0 ? 15 : 0)));
	
	// Fog probability
	const diff = c.temperature_2m - c.dew_point_2m;
	const fogProb = Math.min(98, Math.round((diff < 2 ? 45 : (diff < 4 ? 25 : 5)) + (c.relative_humidity_2m > 80 ? 35 : 15) + (c.wind_speed_10m < 8 ? 20 : 5)));
	
	// Mirage score
	let mirageScore = 10;
	if (c.temperature_2m > 30 && c.cloud_cover < 30) {
		mirageScore = Math.min(95, 60 + (c.temperature_2m - 30) * 3);
	} else if (c.temperature_2m < 5) {
		mirageScore = Math.min(50, 20 + Math.abs(c.temperature_2m) * 3);
	}
	
	// Heat index
	const heatIndexC = c.temperature_2m + 0.5555 * (6.11 * Math.exp(5417.753 * (1/273.16 - 1/(c.dew_point_2m + 273.15))) - 10);
	let comfortRating = "Comfortable";
	if (heatIndexC < 10) comfortRating = "Cool";
	else if (heatIndexC < 22) comfortRating = "Comfortable";
	else if (heatIndexC < 28) comfortRating = "Warm";
	else if (heatIndexC < 35) comfortRating = "Hot";
	else comfortRating = "Dangerous";
	
	// Lightning risk
	let lightningRisk = 0;
	if (c.weather_code >= 95 && c.cloud_cover > 60) {
		lightningRisk = Math.min(90, (c.relative_humidity_2m - 50) * 1.5 + c.wind_speed_10m * 0.5);
	}
	
	// Pollen index
	let pollenBase = 10;
	if (c.temperature_2m > 15) pollenBase += 30;
	if (c.relative_humidity_2m < 60) pollenBase += 20;
	if (c.wind_speed_10m < 15) pollenBase += 15;
	const pollenIndex = Math.round(pollenBase);
	
	let precipProb = c.precipitation;
	if (precipProb === 0 && dailyData.length > 0) precipProb = dailyData[0].precipMax;
	
	const features = [
		{ name: "Rainbow", icon: "bi bi-rainbow", value: bestRainbow, detail: bestRainbowDetail, badge: bestRainbowBadge },
		{ name: "Aurora", icon: "bi bi-stars", value: auroraScore + "/10", detail: `Lat ${Math.abs(lat).toFixed(1)}° · cloud ${Math.round(c.cloud_cover)}%`, badge: auroraScore > 6 ? "Good chance" : "Low" },
		{ name: "Stargazing", icon: "bi bi-moon-stars", value: starsRating + "/5", detail: `Cloud ${Math.round(c.cloud_cover)}% · moon ${Math.round(moonIllum*100)}%`, badge: starsRating > 3 ? "Great" : "Poor" },
		{ name: "Golden hr", icon: "bi bi-sunrise", value: goldenScore + "%", detail: "Cloud pattern optimal", badge: "Photo score" },
		{ name: "Fog risk", icon: "bi bi-cloud-fog2", value: fogProb + "%", detail: `Spread ${diff.toFixed(1)}°C`, badge: "Dew point" },
		{ name: "Mirage", icon: "bi bi-water", value: mirageScore + "%", detail: c.temperature_2m > 30 ? "Hot surface" : (c.temperature_2m < 5 ? "Cold inversion" : "Low gradient"), badge: mirageScore > 40 ? "Visible" : "" },
		{ name: "Lightning", icon: "bi bi-cloud-lightning", value: lightningRisk > 0 ? lightningRisk + "%" : "None", detail: lightningRisk > 30 ? "Seek shelter" : "Safe", badge: lightningRisk > 40 ? "Alert" : "" },
		{ name: "Heat idx", icon: "bi bi-thermometer-sun", value: Math.round(heatIndexC) + "°C", detail: `Feels ${comfortRating}`, badge: comfortRating },
		{ name: "Pollen", icon: "bi bi-flower1", value: pollenIndex + "%", detail: "Temp & wind based", badge: "Allergy" },
		{ name: "Precip", icon: "bi bi-cloud-rain", value: Math.round(precipProb) + "%", detail: "Next hour", badge: "Rain" }
	];
	
	const grid = document.getElementById("featuresGrid");
	grid.innerHTML = features.map(f => `
		<div class="feature-card">
			<div class="feature-title"><i class="${f.icon}"></i> ${f.name}</div>
			<div class="feature-value">${f.value}</div>
			<div class="feature-detail">${f.detail}</div>
			<div class="feature-badge">${f.badge}</div>
		</div>
	`).join('');
}

function updateHourlyTable() {
	if (!weatherCache) return;
	
	let hourlyData = selectedHourlyData && selectedHourlyData.length > 0 ? selectedHourlyData : extractHourlyDataForDay(weatherCache, new Date());
	const tzOff = weatherCache.utc_offset_seconds / 3600;
	const lat = weatherCache.latitude;
	const lon = weatherCache.longitude;
	
	const rainbowPreds = predictRainbow(hourlyData, lat, lon, tzOff, true);
	const rainbowScores = new Map();
	for (const rp of rainbowPreds) {
		rainbowScores.set(rp.time.getTime(), rp.score);
	}
	
	const sunAngles = new Map();
	for (const hour of hourlyData) {
		const utcTime = new Date(hour.time.getTime() - tzOff * 3600000);
		const sunElev = getSolarElevationUTC(lat, lon, utcTime);
		sunAngles.set(hour.time.getTime(), Math.round(sunElev * 10) / 10);
	}
	
	const tbody = document.getElementById("hourlyBody");
	tbody.innerHTML = "";
	
	for (const hour of hourlyData) {
		const [icon] = weatherInfo(hour.weatherCode);
		const temp = formatTemp(hour.temp);
		let sunAngle = sunAngles.get(hour.time.getTime()) || 0;
		let sunAngleStr = sunAngle.toFixed(1) + "°";
		if (sunAngle <= 0) sunAngleStr = `<i class="bi bi-moon"></i> ${sunAngle.toFixed(1)}°`;
		else if (sunAngle < 8) sunAngleStr = `<i class="bi bi-cloud-sun"></i> ${sunAngle.toFixed(1)}°`;
		else sunAngleStr = `<i class="bi bi-sun"></i> ${sunAngle.toFixed(1)}°`;
		
		let rainStr = "-";
		if (rainbowScores.has(hour.time.getTime())) {
			rainStr = rainbowScores.get(hour.time.getTime()) + "% <i class='bi bi-rainbow'></i>";
		}
		
		const row = tbody.insertRow();
		row.insertCell(0).innerText = hour.time.toLocaleTimeString([], {hour:'2-digit', minute:'2-digit'});
		row.insertCell(1).innerHTML = icon;
		row.insertCell(2).innerText = temp;
		row.insertCell(3).innerHTML = rainStr;
		row.insertCell(4).innerHTML = sunAngleStr;
	}
}

function updateDailyTable() {
	if (!weatherCache) return;
	
	const dailyData = extractDailyData(weatherCache);
	const tzOff = weatherCache.utc_offset_seconds / 3600;
	const lat = weatherCache.latitude;
	const lon = weatherCache.longitude;
	const now = new Date();
	
	const tbody = document.getElementById("dailyBody");
	tbody.innerHTML = "";
	
	for (const day of dailyData) {
		const dayHourly = extractHourlyDataForDay(weatherCache, day.date);
		const rainbows = predictRainbow(dayHourly, lat, lon, tzOff, true);
		
		let rainbowStr = "-";
		if (rainbows.length > 0) {
			rainbowStr = rainbows[0].score + "% <i class='bi bi-rainbow'></i>";
		}
		
		const isToday = day.date.toISOString().split('T')[0] === now.toISOString().split('T')[0];
		const dayName = isToday ? "Today" : day.date.toLocaleDateString('en-US', { weekday: 'short' });
		const dateStr = day.date.toLocaleDateString('en-US', { day:'2-digit', month:'2-digit', year:'2-digit' });
		const [icon] = weatherInfo(day.weatherCode);
		const high = formatTemp(day.tempMax);
		const low = formatTemp(day.tempMin);
		
		const row = tbody.insertRow();
		row.setAttribute("data-date", day.date.toISOString());
		row.onclick = () => selectDailyRow(row, day);
		if (selectedDate === day.date.toISOString().split('T')[0]) {
			row.classList.add("selected");
		}
		
		row.insertCell(0).innerText = dateStr;
		row.insertCell(1).innerText = dayName;
		row.insertCell(2).innerHTML = icon;
		row.insertCell(3).innerText = `${high} / ${low}`;
		row.insertCell(4).innerHTML = rainbowStr;
	}
}

function selectDailyRow(row, day) {
	document.querySelectorAll("#dailyBody tr").forEach(r => r.classList.remove("selected"));
	row.classList.add("selected");
	
	const todayStr = new Date().toISOString().split('T')[0];
	const selectedDateStr = day.date.toISOString().split('T')[0];
	
	if (selectedDateStr === todayStr) {
		selectedDayData = null;
		selectedHourlyData = null;
		selectedDate = "today";
	} else {
		selectedDayData = day;
		selectedHourlyData = extractHourlyDataForDay(weatherCache, day.date);
		selectedDate = selectedDateStr;
	}
	
	updateUI();
}

async function updateData() {
	document.getElementById("location").innerText = "Loading weather data...";
	selectedDayData = null;
	selectedHourlyData = null;
	selectedDate = "today";
	
	try {
		weatherCache = await fetchWeather(currentLat, currentLon);
		timezoneOffsetHours = weatherCache.utc_offset_seconds / 3600;
		updateUI();
	} catch (err) {
		document.getElementById("location").innerText = "Error loading weather data";
		console.error(err);
	}
}

function showRainbowFormula() {
	const text = `A rainbow requires three conditions:
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
- <10%    : Not shown`;
	
	document.getElementById("modalBody").innerHTML = text.replace(/\n/g, '<br>');
	document.getElementById("helpModal").style.display = "flex";
}

// Event handlers
async function onSearch() {
	const q = document.getElementById("searchInput").value.trim();
	if (q.length < 2) return;
	
	const results = await searchCity(q);
	const resultsDiv = document.getElementById("searchResults");
	if (results.length > 0) {
		resultsDiv.innerHTML = results.map(r => 
			`<div class="search-result-item" data-lat="${r.lat}" data-lon="${r.lon}" data-name="${r.name}" data-country="${r.country}">
				${r.name}${r.admin1 ? `, ${r.admin1}` : ''} (${r.country})
			</div>`
		).join('');
		resultsDiv.style.display = "block";
		
		document.querySelectorAll(".search-result-item").forEach(el => {
			el.onclick = () => {
				currentLat = parseFloat(el.dataset.lat);
				currentLon = parseFloat(el.dataset.lon);
				locationName = el.dataset.name;
				countryName = el.dataset.country;
				document.getElementById("searchInput").value = locationName + ", " + countryName;
				resultsDiv.style.display = "none";
				updateData();
			};
		});
	} else {
		resultsDiv.style.display = "none";
	}
}

async function onGeoClick() {
	const btn = document.getElementById("geoBtn");
	btn.disabled = true;
	btn.innerHTML = '<span class="spinner-border spinner-border-sm"></span>';
	
	const loc = await getUserLocation();
	currentLat = loc.lat;
	currentLon = loc.lon;
	locationName = loc.city;
	countryName = loc.country;
	
	await updateData();
	
	btn.disabled = false;
	btn.innerHTML = '<i class="bi bi-geo-alt-fill"></i> Location';
}

// Initialize
async function init() {
	document.getElementById("searchBtn").onclick = onSearch;
	document.getElementById("geoBtn").onclick = onGeoClick;
	document.getElementById("helpBtn").onclick = showRainbowFormula;
	document.getElementById("closeModal").onclick = () => document.getElementById("helpModal").style.display = "none";
	document.getElementById("helpModal").onclick = (e) => { if (e.target === document.getElementById("helpModal")) document.getElementById("helpModal").style.display = "none"; };
	
	document.getElementById("searchInput").onkeyup = (e) => { if (e.key === "Enter") onSearch(); };
	document.addEventListener("click", (e) => {
		if (!document.getElementById("searchResults").contains(e.target) && e.target !== document.getElementById("searchInput")) {
			document.getElementById("searchResults").style.display = "none";
		}
	});
	
	const loc = await getUserLocation();
	currentLat = loc.lat;
	currentLon = loc.lon;
	locationName = loc.city;
	countryName = loc.country;
	document.getElementById("searchInput").value = locationName + ", " + countryName;
	await updateData();
}

init();