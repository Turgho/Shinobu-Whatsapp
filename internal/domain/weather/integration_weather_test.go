//go:build integration

// Teste de integração REAL: Open-Meteo (current + daily forecast).
// API pública gratuita — roda apenas com: INTEG=1 go test -tags integration ./internal/domain/weather/
package weather

import (
	"context"
	"os"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestIntegrationOpenMeteoReal(t *testing.T) {
	if os.Getenv("INTEG") != "1" {
		t.Skip("só roda com INTEG=1")
	}

	// coords de Goiânia (-16.68, -49.25)
	w := NewWeatherClient("https://api.open-meteo.com/v1/forecast", zap.NewNop())

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	current, daily, err := w.GetCurrentAndDailyForecast(ctx, -16.68689, -49.26479, 5)
	if err != nil {
		t.Fatalf("GetCurrentAndDailyForecast real falhou: %v", err)
	}

	if current == nil || current.Temperature == 0 && current.WeatherCode == 0 {
		t.Fatalf("current suspeito: %+v", current)
	}
	if len(daily) != 5 {
		t.Fatalf("daily: %d forecasts, want 5", len(daily))
	}
	for i, d := range daily {
		if d.Date == "" {
			t.Fatalf("daily[%d].Date vazio", i)
		}
	}
	t.Logf("current: %+v | daily[0]: %+v", current, daily[0])
}

func TestIntegrationDailyForecastReal(t *testing.T) {
	if os.Getenv("INTEG") != "1" {
		t.Skip("só roda com INTEG=1")
	}

	w := NewWeatherClient("https://api.open-meteo.com/v1/forecast", zap.NewNop())

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	daily, err := w.GetDailyForecast(ctx, -23.5505, -46.6333, 5)
	if err != nil {
		t.Fatalf("GetDailyForecast real (São Paulo) falhou: %v", err)
	}
	if len(daily) != 5 {
		t.Fatalf("daily: %d forecasts, want 5", len(daily))
	}
}
