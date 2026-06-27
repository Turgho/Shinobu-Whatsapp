package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"go.uber.org/zap"
)

type WeatherClient struct {
	APIURL     string
	Logger     *zap.Logger
	httpClient *http.Client
}

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

type hourlyResponse struct {
	Current struct {
		Time                string  `json:"time"`
		Temperature         float64 `json:"temperature_2m"`
		RelativeHumidity    float64 `json:"relative_humidity_2m"`
		ApparentTemperature float64 `json:"apparent_temperature"`
		Precipitation       float64 `json:"precipitation"`
		WeatherCode         int     `json:"weather_code"`
		WindSpeed           float64 `json:"wind_speed_10m"`
		WindDirection       float64 `json:"wind_direction_10m"`
	} `json:"current"`
	Hourly struct {
		Time              []string  `json:"time"`
		Temperature       []float64 `json:"temperature_2m"`
		RelativeHumidity  []float64 `json:"relative_humidity_2m"`
		ApparentTemp      []float64 `json:"apparent_temperature"`
		Precipitation     []float64 `json:"precipitation"`
		WeatherCode       []int     `json:"weather_code"`
		WindSpeed         []float64 `json:"wind_speed_10m"`
		WindDirection     []float64 `json:"wind_direction_10m"`
		PrecipitationProb []float64 `json:"precipitation_probability"`
	} `json:"hourly"`
}

func NewWeatherClient(apiURL string, logger *zap.Logger) *WeatherClient {
	return &WeatherClient{
		APIURL: apiURL,
		Logger: logger,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (w *WeatherClient) GetCurrentWeather(ctx context.Context, lat, lon float64) (*WeatherResult, error) {
	if resp, err := w.fetchHourly(ctx, lat, lon, 1); err == nil {
		result := &WeatherResult{
			Time:                resp.Current.Time,
			Temperature:         resp.Current.Temperature,
			ApparentTemperature: resp.Current.ApparentTemperature,
			WeatherCode:         resp.Current.WeatherCode,
			Precipitation:       resp.Current.Precipitation,
			RelativeHumidity:    resp.Current.RelativeHumidity,
			WindSpeed:           resp.Current.WindSpeed,
			WindDirection:       resp.Current.WindDirection,
		}
		if idx := findHourIndex(resp.Hourly.Time, resp.Current.Time); idx >= 0 &&
			idx < len(resp.Hourly.PrecipitationProb) {
			result.PrecipitationProb = resp.Hourly.PrecipitationProb[idx]
		}
		return result, nil
	}

	w.Logger.Warn("Open-Meteo falhou, tentando fallback wttr.in", zap.Float64("lat", lat), zap.Float64("lon", lon))
	if result, err := w.fetchWttr(ctx, lat, lon); err == nil {
		return result, nil
	}
	return nil, fmt.Errorf("weather: todas as fontes falharam")
}

func (w *WeatherClient) GetForecastForDate(ctx context.Context, lat, lon float64, target time.Time) (*WeatherResult, error) {
	days := max(int(time.Until(target).Hours()/24)+1, 1)
	if days > 16 {
		return nil, fmt.Errorf("weather: previsão disponível apenas para até 16 dias")
	}

	resp, err := w.fetchHourly(ctx, lat, lon, days)
	if err != nil {
		return nil, err
	}

	targetStr := target.Format("2006-01-02") + "T12:00"
	idx := findHourIndex(resp.Hourly.Time, targetStr)
	if idx < 0 || idx >= len(resp.Hourly.Temperature) {
		return nil, fmt.Errorf("weather: data sem dados horários disponíveis")
	}

	result := &WeatherResult{
		Time:                resp.Hourly.Time[idx],
		Temperature:         resp.Hourly.Temperature[idx],
		ApparentTemperature: resp.Hourly.ApparentTemp[idx],
		WeatherCode:         resp.Hourly.WeatherCode[idx],
		Precipitation:       resp.Hourly.Precipitation[idx],
		RelativeHumidity:    resp.Hourly.RelativeHumidity[idx],
		WindSpeed:           resp.Hourly.WindSpeed[idx],
		WindDirection:       resp.Hourly.WindDirection[idx],
	}
	if idx < len(resp.Hourly.PrecipitationProb) {
		result.PrecipitationProb = resp.Hourly.PrecipitationProb[idx]
	}
	return result, nil
}

func (w *WeatherClient) fetchHourly(ctx context.Context, lat, lon float64, forecastDays int) (*hourlyResponse, error) {
	apiURL := fmt.Sprintf(
		"%s?latitude=%f&longitude=%f"+
			"&current=temperature_2m,relative_humidity_2m,apparent_temperature"+
			",precipitation,weather_code,wind_speed_10m,wind_direction_10m"+
			"&hourly=temperature_2m,relative_humidity_2m,apparent_temperature"+
			",precipitation,weather_code,wind_speed_10m,wind_direction_10m,precipitation_probability"+
			"&timezone=auto&forecast_days=%d",
		w.APIURL, lat, lon, forecastDays,
	)

	w.Logger.Info("Buscando dados Open-Meteo",
		zap.Float64("lat", lat),
		zap.Float64("lon", lon),
		zap.Int("forecast_days", forecastDays),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("weather: request: %w", err)
	}

	resp, err := w.httpClient.Do(req)
	if err != nil {
		w.Logger.Error("Erro ao fazer request HTTP (Open-Meteo)", zap.Error(err))
		return nil, fmt.Errorf("weather: erro na requisição: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		w.Logger.Error("Open-Meteo retornou status inesperado",
			zap.Int("status", resp.StatusCode), zap.String("body", string(body)))
		return nil, fmt.Errorf("weather: status %d", resp.StatusCode)
	}

	var data hourlyResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		w.Logger.Error("Erro ao decodificar resposta (Open-Meteo)", zap.Error(err))
		return nil, fmt.Errorf("weather: erro ao decodificar resposta: %w", err)
	}

	return &data, nil
}

func (w *WeatherClient) fetchWttr(ctx context.Context, lat, lon float64) (*WeatherResult, error) {
	apiURL := fmt.Sprintf("https://wttr.in/%f,%f?format=j1", lat, lon)

	w.Logger.Info("Buscando clima atual (wttr.in fallback)", zap.Float64("lat", lat), zap.Float64("lon", lon))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("weather: request wttr.in: %w", err)
	}
	resp, err := w.httpClient.Do(req)
	if err != nil {
		w.Logger.Error("Erro ao fazer request HTTP (wttr.in)", zap.Error(err))
		return nil, fmt.Errorf("weather: erro na requisição wttr.in: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		w.Logger.Error("wttr.in retornou status inesperado",
			zap.Int("status", resp.StatusCode), zap.String("body", string(body)))
		return nil, fmt.Errorf("weather: wttr.in status %d", resp.StatusCode)
	}

	var data struct {
		CurrentCondition []struct {
			TempC         string `json:"temp_C"`
			FeelsLikeC    string `json:"FeelsLikeC"`
			WindspeedKmph string `json:"windspeedKmph"`
			WinddirDegree string `json:"winddirDegree"`
			WeatherDesc   []struct {
				Value string `json:"value"`
			} `json:"weatherDesc"`
			Humidity string `json:"humidity"`
		} `json:"current_condition"`
		NeedsUpdate int `json:"needs_update"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		w.Logger.Error("Erro ao decodificar resposta (wttr.in)", zap.Error(err))
		return nil, fmt.Errorf("weather: erro ao decodificar resposta wttr.in: %w", err)
	}

	if len(data.CurrentCondition) == 0 {
		return nil, fmt.Errorf("weather: wttr.in sem dados de condição atual")
	}
	cond := &data.CurrentCondition[0]

	temp := parseFloat(cond.TempC)
	feelsLike := parseFloat(cond.FeelsLikeC)
	windspeedKmph := parseFloat(cond.WindspeedKmph)
	winddirDegree := parseFloat(cond.WinddirDegree)
	humidity := parseFloat(cond.Humidity)

	weatherCode := -1
	currentTime := time.Now().UTC().Format("2006-01-02T15:00")

	return &WeatherResult{
		Temperature:         temp,
		ApparentTemperature: feelsLike,
		WindSpeed:           windspeedKmph,
		WindDirection:       winddirDegree,
		WeatherCode:         weatherCode,
		Precipitation:       0,
		PrecipitationProb:   0,
		RelativeHumidity:    humidity,
		Time:                currentTime,
	}, nil
}

func parseFloat(s string) float64 {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return f
}

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
