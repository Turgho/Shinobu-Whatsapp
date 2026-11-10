package weather

import "testing"

// dailyForecastAt não pode estourar índice quando a API devolve arrays
// de tamanhos diferentes (ou vazios). Antes isso era um panic em runtime.
func TestDailyForecastAtArraysDesiguais(t *testing.T) {
	daily := dailyData{
		Time:                   []string{"2026-11-10", "2026-11-11", "2026-11-12"},
		WeatherCode:            []int{3, 61},
		Temperature2MMax:       []float64{28.5},
		Temperature2MMin:       []float64{17.2, 16.9, 18.0},
		ApparentTemperatureMax: []float64{30.1, 29.4},
		PrecipitationProbMax:   []float64{10, 80, 0},
	}

	fc := dailyForecastAt(0, "2026-11-10", daily)
	if fc.TempMax != 28.5 || fc.TempMin != 17.2 || fc.WeatherCode != 3 {
		t.Fatalf("índice 0 lido errado: %+v", fc)
	}

	// Índice 1: Temperature2MMax não existe → 0, resto lido.
	fc = dailyForecastAt(1, "2026-11-11", daily)
	if fc.TempMax != 0 {
		t.Errorf("TempMax ausente deveria ser 0, veio %v", fc.TempMax)
	}
	if fc.TempMin != 16.9 || fc.WeatherCode != 61 || fc.PrecipitationProb != 80 {
		t.Errorf("índice 1 lido errado: %+v", fc)
	}

	// Índice 2: só min e precip existem.
	fc = dailyForecastAt(2, "2026-11-12", daily)
	if fc.TempMax != 0 || fc.WeatherCode != 0 || fc.ApparentTempMax != 0 {
		t.Errorf("índice 2 deveria ter 0 nos campos ausentes: %+v", fc)
	}
	if fc.TempMin != 18.0 || fc.PrecipitationProb != 0 {
		t.Errorf("índice 2 lido errado: %+v", fc)
	}
}

func TestDailyForecastAtDailyVazio(t *testing.T) {
	fc := dailyForecastAt(0, "2026-11-10", dailyData{})
	if fc.TempMax != 0 || fc.WeatherCode != 0 || fc.PrecipitationProb != 0 {
		t.Fatalf("daily vazio deveria virar zeros: %+v", fc)
	}
}
