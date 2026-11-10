//go:build integration

// Teste de integração REAL: geocoding via Nominatim + fallback Open-Meteo.
// APIs públicas gratuitas — roda apenas com: INTEG=1 go test -tags integration ./internal/domain/geocoding/
package geocoding

import (
	"context"
	"os"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestIntegrationLookupReal(t *testing.T) {
	if os.Getenv("INTEG") != "1" {
		t.Skip("só roda com INTEG=1")
	}

	g := NewGeoCoding(
		"https://nominatim.openstreetmap.org/search",
		"https://geocoding-api.open-meteo.com/v1/search",
		zap.NewNop(),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	results, err := g.Lookup(ctx, "Goiânia", 1)
	if err != nil {
		t.Fatalf("lookup real falhou: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("esperava ao menos 1 resultado para Goiânia")
	}

	r := results[0]
	if r.Latitude == 0 || r.Longitude == 0 {
		t.Fatalf("coords de Goiânia não podem ser 0: %+v", r)
	}
	// DisplayName nunca deve ser display_name cru do Nominatim (múltiplas
	// vírgulas de district/county/state): valida formato "Cidade[, Estado]"
	if r.DisplayName == "" {
		t.Fatal("DisplayName vazio")
	}
	if n := countCommas(r.DisplayName); n > 1 {
		t.Errorf("DisplayName com %d vírgulas parece display_name cru: %q", n, r.DisplayName)
	}

	t.Logf("resultado: %+v", r)
}

// countCommas conta vírgulas em s.
func countCommas(s string) int {
	n := 0
	for _, r := range s {
		if r == ',' {
			n++
		}
	}
	return n
}
