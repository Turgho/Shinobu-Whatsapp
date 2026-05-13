package weather

// WeatherCodeInfo guarda a descrição e o emoji de um código climático.
type WeatherCodeInfo struct {
	Description string
	Emoji       string
}

// WeatherCodeMap mapeia os códigos da Open-Meteo para descrições em português.
// Referência: https://open-meteo.com/en/docs#weathervariables
var WeatherCodeMap = map[int]WeatherCodeInfo{
	0:  {"Céu limpo", "☀️"},
	1:  {"Principalmente limpo", "🌤️"},
	2:  {"Parcialmente nublado", "⛅"},
	3:  {"Encoberto", "☁️"},
	45: {"Névoa", "🌫️"},
	48: {"Névoa com gelo depositado", "🌫️"},
	51: {"Garoa leve", "🌦️"},
	53: {"Garoa moderada", "🌧️"},
	55: {"Garoa intensa", "🌧️"},
	56: {"Garoa congelante leve", "🌨️"},
	57: {"Garoa congelante intensa", "❄️"},
	61: {"Chuva leve", "🌧️"},
	63: {"Chuva moderada", "🌧️"},
	65: {"Chuva forte", "🌧️"},
	66: {"Chuva congelante leve", "🌨️"},
	67: {"Chuva congelante intensa", "❄️"},
	71: {"Neve leve", "❄️"},
	73: {"Neve moderada", "❄️"},
	75: {"Neve intensa", "❄️"},
	77: {"Grãos de neve", "🌨️"},
	80: {"Chuva de pancadas leve", "🌦️"},
	81: {"Chuva de pancadas moderada", "🌧️"},
	82: {"Chuva de pancadas forte", "⛈️"},
	85: {"Neve de pancadas leve", "🌨️"},
	86: {"Neve de pancadas intensa", "❄️"},
	95: {"Trovoada", "⛈️"},
	96: {"Trovoada com granizo leve", "⛈️⚡"},
	99: {"Trovoada com granizo intenso", "⛈️❄️"},
}

// Lookup retorna as informações de um código climático.
// Se o código não for encontrado, retorna um fallback genérico.
func Lookup(code int) WeatherCodeInfo {
	if info, ok := WeatherCodeMap[code]; ok {
		return info
	}
	return WeatherCodeInfo{"Condição desconhecida", "🌡️"}
}
