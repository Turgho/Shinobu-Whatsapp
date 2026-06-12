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
	// Parâmetros escolhidos para aumentar precisão e cobertura de dados
	apiURL := fmt.Sprintf(
		"%s?latitude=%f&longitude=%f"+
			"&hourly=temperature_2m,relativehumidity_2m,apparent_temperature,precipitation,precipitation_probability,weathercode,windspeed_10m,winddirection_10m,cloudcover"+
			"&current_weather=true&timezone=auto&forecast_days=1",
		w.APIURL, lat, lon,
	)

	w.Logger.Info("Buscando clima atual (Open-Meteo)", zap.Float64("lat", lat), zap.Float64("lon", lon))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("weather: request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
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
		CurrentWeather struct {
			Temperature   float64 `json:"temperature"`
			Windspeed     float64 `json:"windspeed"`
			WindDirection float64 `json:"winddirection"`
			WeatherCode   int     `json:"weathercode"`
			Time          string  `json:"time"` // formato: "2006-01-02T15:00"
		} `json:"current_weather"`
		Hourly struct {
			Time                []string  `json:"time"`
			Temperature2m       []float64 `json:"temperature_2m"`
			RelativeHumidity2m  []float64 `json:"relativehumidity_2m"`
			ApparentTemperature []float64 `json:"apparent_temperature"`
			Precipitation       []float64 `json:"precipitation"`
			PrecipitationProb   []float64 `json:"precipitation_probability"`
			WeatherCode         []int     `json:"weathercode"`
			Windspeed10m        []float64 `json:"windspeed_10m"`
			WindDirection10m    []float64 `json:"winddirection_10m"`
			Cloudcover          []float64 `json:"cloudcover"`
		} `json:"hourly"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		w.Logger.Error("Erro ao decodificar resposta (Open-Meteo)", zap.Error(err))
		return nil, fmt.Errorf("weather: erro ao decodificar resposta: %w", err)
	}

	result := &WeatherResult{
		Temperature:   data.CurrentWeather.Temperature,
		WindSpeed:     data.CurrentWeather.Windspeed,
		WindDirection: data.CurrentWeather.WindDirection,
		WeatherCode:   data.CurrentWeather.WeatherCode,
		Time:          data.CurrentWeather.Time,
	}

	// Preenche os dados horários extras alinhados ao horário atual
	idx := findHourIndex(data.Hourly.Time, data.CurrentWeather.Time)
	if idx >= 0 {
		if idx < len(data.Hourly.Temperature2m) {
			// Temperatura já vem de current_weather; mantemos esse valor.
		}
		if idx < len(data.Hourly.RelativeHumidity2m) {
			result.RelativeHumidity = data.Hourly.RelativeHumidity2m[idx]
		}
		if idx < len(data.Hourly.ApparentTemperature) {
			result.ApparentTemperature = data.Hourly.ApparentTemperature[idx]
		}
		if idx < len(data.Hourly.Precipitation) {
			result.Precipitation = data.Hourly.Precipitation[idx]
		}
		if idx < len(data.Hourly.PrecipitationProb) {
			result.PrecipitationProb = data.Hourly.PrecipitationProb[idx]
		}
		// Velocidade e direção do vento já vem de current_weather (10 m).
		if idx < len(data.Hourly.Windspeed10m) {
			result.WindSpeed = data.Hourly.Windspeed10m[idx]
		}
		if idx < len(data.Hourly.WindDirection10m) {
			result.WindDirection = data.Hourly.WindDirection10m[idx]
		}
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

	// Descrição do weather não utilizada diretamente; código 0 (céu limpo) como fallback.
	weatherCode := 0

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
