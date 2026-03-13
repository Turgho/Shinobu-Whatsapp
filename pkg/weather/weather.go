package weather

import (
	"encoding/json"
	"fmt"
	"net/http"

	"go.uber.org/zap"
)

// WeatherClient representa o client de clima
type WeatherClient struct {
	APIURL string
	Logger *zap.Logger
}

// WeatherResult guarda os dados de clima de um lugar
type WeatherResult struct {
	Temperature         float64 `json:"temperature_2m"`
	ApparentTemperature float64 `json:"apparent_temperature"`
	WeatherCode         int     `json:"weathercode"`
	Precipitation       float64 `json:"precipitation"`
	PrecipitationProb   float64 `json:"precipitation_probability"`
	RelativeHumidity    float64 `json:"relativehumidity_2m"`
	WindSpeed           float64 `json:"windspeed_10m"`
	WindDirection       float64 `json:"winddirection_10m"`
	Time                string  `json:"time"`
}

// NewWeatherClient é o construtor com logger
func NewWeatherClient(apiURL string, logger *zap.Logger) *WeatherClient {
	return &WeatherClient{
		APIURL: apiURL,
		Logger: logger,
	}
}

// GetHourlyWeather busca o clima horário de um lugar
func (w *WeatherClient) GetHourlyWeather(lat, lon float64) ([]WeatherResult, error) {
	url := fmt.Sprintf("%s?latitude=%f&longitude=%f&hourly=temperature_2m,apparent_temperature,weathercode,precipitation,precipitation_probability,relativehumidity_2m,windspeed_10m,winddirection_10m&current_weather=true&timezone=auto", w.APIURL, lat, lon)

	w.Logger.Info("Fazendo request de clima", zap.String("url", url))

	resp, err := http.Get(url)
	if err != nil {
		w.Logger.Error("Erro ao fazer request HTTP", zap.Error(err))
		return nil, err
	}
	defer resp.Body.Close()

	var data struct {
		Hourly struct {
			Time                []string  `json:"time"`
			Temperature2m       []float64 `json:"temperature_2m"`
			ApparentTemperature []float64 `json:"apparent_temperature"`
			WeatherCode         []int     `json:"weathercode"`
			Precipitation       []float64 `json:"precipitation"`
			PrecipitationProb   []float64 `json:"precipitation_probability"`
			RelativeHumidity    []float64 `json:"relativehumidity_2m"`
			WindSpeed           []float64 `json:"windspeed_10m"`
			WindDirection       []float64 `json:"winddirection_10m"`
		} `json:"hourly"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		w.Logger.Error("Erro ao decodificar JSON", zap.Error(err))
		return nil, err
	}

	results := make([]WeatherResult, len(data.Hourly.Time))
	for i := range data.Hourly.Time {
		results[i] = WeatherResult{
			Time:                data.Hourly.Time[i],
			Temperature:         data.Hourly.Temperature2m[i],
			ApparentTemperature: data.Hourly.ApparentTemperature[i],
			WeatherCode:         data.Hourly.WeatherCode[i],
			Precipitation:       data.Hourly.Precipitation[i],
			PrecipitationProb:   data.Hourly.PrecipitationProb[i],
			RelativeHumidity:    data.Hourly.RelativeHumidity[i],
			WindSpeed:           data.Hourly.WindSpeed[i],
			WindDirection:       data.Hourly.WindDirection[i],
		}
	}

	w.Logger.Info("Dados de weather obtidos", zap.Int("count", len(results)))
	return results, nil
}
