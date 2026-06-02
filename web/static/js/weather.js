// weather.js - Gestion de la météo

// City coordinates from config
let cityCoordinates = {};

function loadWeather(city = 'Paris') {
    const container = document.getElementById('weather-forecast');
    if (!container) return;

    container.innerHTML = '<i class="fas fa-spinner fa-spin"></i>';

    const coords = cityCoordinates[city];
    if (!coords) {
        container.innerHTML = '<span class="weather-error">Ville non disponible</span>';
        return;
    }

    // Get 5-day forecast
    fetch(`https://api.open-meteo.com/v1/forecast?latitude=${coords.Lat}&longitude=${coords.Lon}&daily=weathercode,temperature_2m_max,temperature_2m_min&timezone=${encodeURIComponent(coords.Timezone)}&forecast_days=5`)
        .then(response => response.json())
        .then(data => {
            if (data.daily && data.daily.time && data.daily.time.length >= 5) {
                renderForecast(data, city);
            } else {
                container.innerHTML = '<span class="weather-error">Données météo non disponibles</span>';
            }
        })
        .catch(() => {
            container.innerHTML = '<span class="weather-error">Erreur de chargement</span>';
        });
}

function renderForecast(data, city) {
    const container = document.getElementById('weather-forecast');
    if (!container) return;

    const days = data.daily.time.slice(0, 5);
    const maxTemps = data.daily.temperature_2m_max.slice(0, 5);
    const minTemps = data.daily.temperature_2m_min.slice(0, 5);
    const weatherCodes = data.daily.weathercode.slice(0, 5);

    let html = '<div class="forecast-days">';

    days.forEach((day, index) => {
        const date = new Date(day);
        const dayName = date.toLocaleDateString('fr-FR', {weekday: 'short'});
        const maxTemp = Math.round(maxTemps[index]);
        const minTemp = Math.round(minTemps[index]);
        const code = weatherCodes[index];
        const icon = getWeatherIcon(code);
        const desc = getWeatherDescription(code);

        html += `
            <div class="forecast-day" title="${dayName} - ${desc}">
                <span class="forecast-dayname">${dayName}</span>
                <i class="${icon} forecast-icon"></i>
                <span class="forecast-temp">${maxTemp}°/${minTemp}°</span>
            </div>
        `;
    });

    html += '</div>';
    container.innerHTML = html;
}

function changeWeatherCity() {
    const select = document.getElementById('weather-city');
    if (select) {
        const city = select.value;
        localStorage.setItem('weatherCity', city);
        loadWeather(city);
    }
}

function getWeatherIcon(code, isDay = true) {
    const dayIcons = {
        0: 'fas fa-sun',
        1: 'fas fa-cloud-sun',
        2: 'fas fa-cloud-sun',
        3: 'fas fa-cloud',
        45: 'fas fa-smog',
        48: 'fas fa-smog',
        51: 'fas fa-cloud-rain',
        53: 'fas fa-cloud-rain',
        55: 'fas fa-cloud-rain',
        56: 'fas fa-cloud-rain',
        57: 'fas fa-cloud-rain',
        61: 'fas fa-cloud-rain',
        63: 'fas fa-cloud-showers-heavy',
        65: 'fas fa-cloud-showers-heavy',
        66: 'fas fa-snowflake',
        67: 'fas fa-snowflake',
        71: 'fas fa-snowflake',
        73: 'fas fa-snowflake',
        75: 'fas fa-snowflake',
        77: 'fas fa-snowflake',
        80: 'fas fa-cloud-showers-heavy',
        81: 'fas fa-cloud-showers-heavy',
        82: 'fas fa-cloud-showers-heavy',
        85: 'fas fa-snowflake',
        86: 'fas fa-snowflake',
        95: 'fas fa-bolt',
        96: 'fas fa-bolt',
        99: 'fas fa-bolt',
    };
    return dayIcons[code] || dayIcons[3] || 'fas fa-cloud';
}

function getWeatherDescription(code) {
    const descriptions = {
        0: 'Ciel dégagé',
        1: 'Principalement dégagé',
        2: 'Partiellement nuageux',
        3: 'Nuageux',
        45: 'Brouillard',
        48: 'Brouillard givrant',
        51: 'Bruine légère',
        53: 'Bruine modérée',
        55: 'Bruine dense',
        61: 'Pluie légère',
        63: 'Pluie modérée',
        65: 'Pluie forte',
        71: 'Neige légère',
        73: 'Neige modérée',
        75: 'Neige forte',
        80: 'Averses légères',
        81: 'Averses modérées',
        82: 'Averses violentes',
        95: 'Orage',
        96: 'Orage avec grêle légère',
        99: 'Orage avec grêle forte',
    };
    return descriptions[code] || 'Inconnu';
}

// Initialize weather with saved city on page load
function initWeather() {
    const savedCity = localStorage.getItem('weatherCity') || 'Paris';
    const citySelect = document.getElementById('weather-city');
    if (citySelect) {
        citySelect.value = savedCity;
    }
    loadWeather(savedCity);
}
