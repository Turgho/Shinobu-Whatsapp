package weather

// WeatherInfo guarda descrição e emoji
type WeatherCodeInfo struct {
	Description string
	Emoji       string
}

// WeatherCodeMap associa cada weather code da Open-Meteo a descrição + emoji
var WeatherCodeMap = map[int]WeatherCodeInfo{
	0:  {"Céu limpo", "☀️"},
	1:  {"Principalmente limpo, parcialmente nublado e encoberto", "🌤️"},
	2:  {"Principalmente limpo, parcialmente nublado e encoberto", "🌤️"},
	3:  {"Principalmente limpo, parcialmente nublado e encoberto", "☁️"},
	45: {"Névoa e neblina com gelo depositado", "🌫️"},
	48: {"Névoa e neblina com gelo depositado", "🌫️"},
	51: {"Garoa: leve, moderada e intensa", "🌦️"},
	53: {"Garoa: leve, moderada e intensa", "🌧️"},
	55: {"Garoa: leve, moderada e intensa", "🌧️"},
	56: {"Garoa congelante: leve e intensa", "🌨️"},
	57: {"Garoa congelante: leve e intensa", "❄️"},
	61: {"Chuva: leve, moderada e forte", "🌧️"},
	63: {"Chuva: leve, moderada e forte", "🌧️"},
	65: {"Chuva: leve, moderada e forte", "🌧️"},
	66: {"Chuva congelante: leve e intensa", "🌨️"},
	67: {"Chuva congelante: leve e intensa", "❄️"},
	71: {"Neve: leve, moderada e intensa", "❄️"},
	73: {"Neve: leve, moderada e intensa", "❄️"},
	75: {"Neve: leve, moderada e intensa", "❄️"},
	77: {"Grãos de neve", "🌨️"},
	80: {"Chuva de pancadas: leve, moderada e forte", "🌦️"},
	81: {"Chuva de pancadas: leve, moderada e forte", "🌧️"},
	82: {"Chuva de pancadas: leve, moderada e forte", "⛈️"},
	85: {"Neve de pancadas: leve e intensa", "🌨️"},
	86: {"Neve de pancadas: leve e intensa", "❄️"},
	95: {"Trovoada: leve ou moderada", "⛈️"},
	96: {"Trovoada com granizo: leve e intensa", "⛈️⚡"},
	99: {"Trovoada com granizo: leve e intensa", "⛈️❄️"},
}
