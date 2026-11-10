package weather

import (
	"strings"
	"testing"
)

// GenerateForecastCard com lista vazia precisa devolver erro — antes
// forecasts[0] causava panic e o comando !clima morria sem resposta.
func TestGenerateForecastCardSemPrevisao(t *testing.T) {
	_, err := GenerateForecastCard(nil, nil, "São Paulo", "Brasil")
	if err == nil {
		t.Fatal("esperava erro com lista de previsões vazia")
	}
	if !strings.Contains(err.Error(), "nenhuma previsão") {
		t.Errorf("mensagem de erro pouco clara: %v", err)
	}
}

// Com um único dia, o card ainda deve ser gerado (hero + 1 linha).
func TestGenerateForecastCardUmDia(t *testing.T) {
	forecasts := []DailyForecast{{
		Date:        "2026-11-10",
		WeatherCode: 3,
		TempMax:     28,
		TempMin:     17,
	}}
	png, err := GenerateForecastCard(forecasts, nil, "São Paulo", "Brasil")
	if err != nil {
		t.Fatalf("esperava card gerado: %v", err)
	}
	if len(png) == 0 {
		t.Fatal("card vazio")
	}
}
