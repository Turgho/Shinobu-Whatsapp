package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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

// hourlyPayload espelha o bloco hourly da Open-Meteo para decode e pós-processamento.
type hourlyPayload struct {
	Time                []string  `json:"time"`
	ApparentTemperature []float64 `json:"apparent_temperature"`
	Precipitation       []float64 `json:"precipitation"`
	PrecipitationProb   []float64 `json:"precipitation_probability"`
	RelativeHumidity    []float64 `json:"relativehumidity_2m"`
}

// GetCurrentWeather busca o clima atual para as coordenadas fornecidas.
// Usa o campo current_weather da API para o dado mais preciso do momento,
// complementado com dados horários para apparent_temperature e precipitação.
func (w *WeatherClient) GetCurrentWeather(ctx context.Context, lat, lon float64) (*WeatherResult, error) {
	apiURL := fmt.Sprintf(
		"%s?latitude=%f&longitude=%f"+
			"&hourly=apparent_temperature,precipitation,precipitation_probability,relativehumidity_2m"+
			"&current_weather=true&timezone=auto&forecast_days=1",
		w.APIURL, lat, lon,
	)

	w.Logger.Info("Buscando clima atual", zap.Float64("lat", lat), zap.Float64("lon", lon))

	// Faz a requisição HTTP para a API Open-Meteo
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("weather: request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		w.Logger.Error("Erro ao fazer request HTTP", zap.Error(err))
		return nil, fmt.Errorf("weather: erro na requisição: %w", err)
	}
	defer resp.Body.Close()

	// Verifica se a resposta da API é OK
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		w.Logger.Error("Open-Meteo retornou status inesperado",
			zap.Int("status", resp.StatusCode), zap.String("body", string(body)))
		return nil, fmt.Errorf("weather: status %d", resp.StatusCode)
	}

	var data struct {
		CurrentWeather struct {
			Temperature   float64 `json:"temperature"`
			WindSpeed     float64 `json:"windspeed"`
			WindDirection float64 `json:"winddirection"`
			WeatherCode   int     `json:"weathercode"`
			Time          string  `json:"time"` // formato: "2006-01-02T15:00"
		} `json:"current_weather"`
		Hourly hourlyPayload `json:"hourly"`
	}

	// Decodifica a resposta da API
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		w.Logger.Error("Erro ao decodificar resposta", zap.Error(err))
		return nil, fmt.Errorf("weather: erro ao decodificar resposta: %w", err)
	}

	result := &WeatherResult{
		Temperature:   data.CurrentWeather.Temperature,
		WindSpeed:     data.CurrentWeather.WindSpeed,
		WindDirection: data.CurrentWeather.WindDirection,
		WeatherCode:   data.CurrentWeather.WeatherCode,
		Time:          data.CurrentWeather.Time,
	}

	// Preenche os dados horários extras
	fillHourlyExtras(result, &data.Hourly, data.CurrentWeather.Time)

	w.Logger.Info("Clima obtido", zap.String("time", result.Time))
	return result, nil
}

// fillHourlyExtras alinha série horária ao instante de current_weather (mesmo formato ISO).
func fillHourlyExtras(dst *WeatherResult, hourly *hourlyPayload, currentTime string) {
	idx := findHourIndex(hourly.Time, currentTime)
	if idx < 0 {
		return
	}
	if idx < len(hourly.ApparentTemperature) {
		dst.ApparentTemperature = hourly.ApparentTemperature[idx]
	}
	if idx < len(hourly.Precipitation) {
		dst.Precipitation = hourly.Precipitation[idx]
	}
	if idx < len(hourly.PrecipitationProb) {
		dst.PrecipitationProb = hourly.PrecipitationProb[idx]
	}
	if idx < len(hourly.RelativeHumidity) {
		dst.RelativeHumidity = hourly.RelativeHumidity[idx]
	}
}

// findHourIndex encontra a posição do horário atual dentro da lista horária da API.
// Os timestamps têm o formato "2006-01-02T15:00" — mesmo formato em current_weather.time.
func findHourIndex(times []string, currentTime string) int {
	if len(currentTime) < 13 {
		return -1
	}
	currentHour := currentTime[:13]

	for i, t := range times {
		if len(t) >= 13 && t[:13] == currentHour {
			return i
		}
	}

	for i := len(times) - 1; i >= 0; i-- {
		if times[i] < currentTime {
			return i
		}
	}

	return -1
}
