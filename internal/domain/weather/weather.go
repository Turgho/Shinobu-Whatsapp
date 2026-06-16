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

// WeatherClient é o client da API Open-Meteo.
type WeatherClient struct {
	APIURL     string
	Logger     *zap.Logger
	httpClient *http.Client
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

// NewWeatherClient cria um novo client de clima com timeout.
func NewWeatherClient(apiURL string, logger *zap.Logger) *WeatherClient {
	return &WeatherClient{
		APIURL: apiURL,
		Logger: logger,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetCurrentWeather busca o clima atual para as coordenadas fornecidas.
// Usa o campo current_weather da API Open-Meteo, complementado com dados horários.
// Se a Open-Meteo falhar, tenta fallback para wttr.in.
func (w *WeatherClient) GetCurrentWeather(ctx context.Context, lat, lon float64) (*WeatherResult, error) {
	// Primeira tentativa: Open-Meteo
	if result, err := w.fetchOpenMeteo(ctx, lat, lon); err == nil {
		return result, nil
	}
	w.Logger.Warn("Open-Meteo falhou, tentando fallback wttr.in", zap.Float64("lat", lat), zap.Float64("lon", lon))
	// Segunda tentativa: wttr.in
	if result, err := w.fetchWttr(ctx, lat, lon); err == nil {
		return result, nil
	}
	return nil, fmt.Errorf("weather: todas as fontes falharam")
}

// fetchOpenMeteo realiza a requisição à API Open-Meteo com parâmetros otimizados para precisão.
func (w *WeatherClient) fetchOpenMeteo(ctx context.Context, lat, lon float64) (*WeatherResult, error) {
	apiURL := fmt.Sprintf(
		"%s?latitude=%f&longitude=%f"+
			"&current=temperature_2m,relative_humidity_2m,apparent_temperature"+
			",precipitation,weather_code,wind_speed_10m,wind_direction_10m"+
			"&hourly=precipitation_probability"+ // único campo não disponível em current
			"&timezone=auto&forecast_days=1",
		w.APIURL, lat, lon,
	)

	w.Logger.Info("Buscando clima atual (Open-Meteo)", zap.Float64("lat", lat), zap.Float64("lon", lon))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("weather: request: %w", err)
	}

	resp, err := w.httpClient.Do(req) // ver nota sobre httpClient abaixo
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

	var data struct {
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
			PrecipitationProb []float64 `json:"precipitation_probability"`
		} `json:"hourly"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		w.Logger.Error("Erro ao decodificar resposta (Open-Meteo)", zap.Error(err))
		return nil, fmt.Errorf("weather: erro ao decodificar resposta: %w", err)
	}

	result := &WeatherResult{
		Time:                data.Current.Time,
		Temperature:         data.Current.Temperature,
		ApparentTemperature: data.Current.ApparentTemperature,
		WeatherCode:         data.Current.WeatherCode,
		Precipitation:       data.Current.Precipitation,
		RelativeHumidity:    data.Current.RelativeHumidity,
		WindSpeed:           data.Current.WindSpeed,
		WindDirection:       data.Current.WindDirection,
	}

	// precipitation_probability só existe em hourly — alinha ao horário atual
	if idx := findHourIndex(data.Hourly.Time, data.Current.Time); idx >= 0 &&
		idx < len(data.Hourly.PrecipitationProb) {
		result.PrecipitationProb = data.Hourly.PrecipitationProb[idx]
	}

	return result, nil
}

// fetchWttr tenta obter o clima do wttr.in como fallback.
func (w *WeatherClient) fetchWttr(ctx context.Context, lat, lon float64) (*WeatherResult, error) {
	// wttr.in aceita latitude,longitude e retorna JSON com formato j1
	apiURL := fmt.Sprintf("https://wttr.in/%f,%f?format=j1", lat, lon)

	w.Logger.Info("Buscando clima atual (wttr.in fallback)", zap.Float64("lat", lat), zap.Float64("lon", lon))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("weather: request wttr.in: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
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

	// Converter strings para float64
	temp := parseFloat(cond.TempC)
	feelsLike := parseFloat(cond.FeelsLikeC)
	windspeedKmph := parseFloat(cond.WindspeedKmph)
	winddirDegree := parseFloat(cond.WinddirDegree)
	humidity := parseFloat(cond.Humidity)

	// Descrição do weather não utilizada diretamente; código -1 ausente
	weatherCode := -1

	// Timestamp ISO usando a hora local (wttr.in não fornece timestamp)
	// Usado a hora atual UTC como aproximação do timestamp.
	currentTime := time.Now().UTC().Format("2006-01-02T15:00")

	return &WeatherResult{
		Temperature:         temp,
		ApparentTemperature: feelsLike,
		WindSpeed:           windspeedKmph,
		WindDirection:       winddirDegree,
		WeatherCode:         weatherCode,
		Precipitation:       0, // wttr.in não fornece precipitação imediata neste endpoint
		PrecipitationProb:   0,
		RelativeHumidity:    humidity,
		Time:                currentTime,
	}, nil
}

// parseFloat converte string para float64, retornando 0 em caso de erro.
func parseFloat(s string) float64 {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return f
}

// findHourIndex permanece igual (cópia da função original para uso interno)
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
