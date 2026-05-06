package weather

import (
	"encoding/json"
	"fmt"
	"net/http"

	"go.uber.org/zap"
)

// WeatherClient é o client da API Open-Meteo.
type WeatherClient struct {
	APIURL string
	Logger *zap.Logger
}

// WeatherResult contém os dados climáticos do momento atual.
type WeatherResult struct {
	Temperature         float64
	ApparentTemperature float64
	WeatherCode         int
	Precipitation       float64
	PrecipitationProb   float64
	RelativeHumidity    float64
	WindSpeed           float64
	WindDirection       float64
	Time                string
}

// NewWeatherClient cria um novo client de clima.
func NewWeatherClient(apiURL string, logger *zap.Logger) *WeatherClient {
	return &WeatherClient{APIURL: apiURL, Logger: logger}
}

// GetCurrentWeather busca o clima atual para as coordenadas fornecidas.
// Usa o campo current_weather da API para o dado mais preciso do momento,
// complementado com dados horários para apparent_temperature e precipitação.
func (w *WeatherClient) GetCurrentWeather(lat, lon float64) (*WeatherResult, error) {
	apiURL := fmt.Sprintf(
		"%s?latitude=%f&longitude=%f"+
			"&hourly=apparent_temperature,precipitation,precipitation_probability,relativehumidity_2m"+
			"&current_weather=true&timezone=auto&forecast_days=1",
		w.APIURL, lat, lon,
	)

	w.Logger.Info("Buscando clima atual", zap.Float64("lat", lat), zap.Float64("lon", lon))

	// Requisição da Api
	resp, err := http.Get(apiURL)
	if err != nil {
		w.Logger.Error("Erro ao fazer request HTTP", zap.Error(err))
		return nil, fmt.Errorf("weather: erro na requisição: %w", err)
	}
	defer resp.Body.Close()

	// Estrutura esperada da requisição
	var data struct {
		CurrentWeather struct {
			Temperature   float64 `json:"temperature"`
			WindSpeed     float64 `json:"windspeed"`
			WindDirection float64 `json:"winddirection"`
			WeatherCode   int     `json:"weathercode"`
			Time          string  `json:"time"` // formato: "2006-01-02T15:00"
		} `json:"current_weather"`
		Hourly struct {
			Time                []string  `json:"time"`
			ApparentTemperature []float64 `json:"apparent_temperature"`
			Precipitation       []float64 `json:"precipitation"`
			PrecipitationProb   []float64 `json:"precipitation_probability"`
			RelativeHumidity    []float64 `json:"relativehumidity_2m"`
		} `json:"hourly"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		w.Logger.Error("Erro ao decodificar resposta", zap.Error(err))
		return nil, fmt.Errorf("weather: erro ao decodificar resposta: %w", err)
	}

	// Resultado dos dados para envio
	result := &WeatherResult{
		Temperature:   data.CurrentWeather.Temperature,
		WindSpeed:     data.CurrentWeather.WindSpeed,
		WindDirection: data.CurrentWeather.WindDirection,
		WeatherCode:   data.CurrentWeather.WeatherCode,
		Time:          data.CurrentWeather.Time,
	}

	// Encontra o índice da hora atual nos dados horários para preencher
	// apparent_temperature, precipitation e relativehumidity
	if idx := findHourIndex(data.Hourly.Time, data.CurrentWeather.Time); idx >= 0 {
		if idx < len(data.Hourly.ApparentTemperature) {
			result.ApparentTemperature = data.Hourly.ApparentTemperature[idx]
		}

		if idx < len(data.Hourly.Precipitation) {
			result.Precipitation = data.Hourly.Precipitation[idx]
		}
		if idx < len(data.Hourly.PrecipitationProb) {
			result.PrecipitationProb = data.Hourly.PrecipitationProb[idx]
		}
		if idx < len(data.Hourly.RelativeHumidity) {
			result.RelativeHumidity = data.Hourly.RelativeHumidity[idx]
		}
	}

	w.Logger.Info("Clima obtido", zap.String("time", result.Time))
	return result, nil
}

// findHourIndex encontra a posição do horário atual dentro da lista horária da API.
// Os timestamps têm o formato "2006-01-02T15:00" — mesmo formato em current_weather.time.
func findHourIndex(times []string, currentTime string) int {
	currentHour := currentTime[:13] // pega apenas a hora

	for i, t := range times {
		if len(t) >= 13 && t[:13] == currentHour {
			return i
		}
	}

	// fallback: pega o mais próximo anterior
	for i := len(times) - 1; i >= 0; i-- {
		if times[i] < currentTime {
			return i
		}
	}

	return -1
}
