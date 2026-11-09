//go:build integration

// Teste de integração REAL: AwesomeAPI de cotações (gratuita, sem chave).
// Roda apenas com: INTEG=1 go test -tags integration ./internal/domain/cotacao/
package cotacao

import (
	"context"
	"os"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestIntegrationFetchReal(t *testing.T) {
	if os.Getenv("INTEG") != "1" {
		t.Skip("só roda com INTEG=1")
	}

	c := NewCotacaoClient(
		"https://economia.awesomeapi.com.br/json/last/USD-BRL,EUR-BRL",
		zap.NewNop(),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	results, err := c.Fetch(ctx)
	if err != nil {
		t.Fatalf("fetch real falhou: %v", err)
	}
	if len(results) < 2 {
		t.Fatalf("esperava USD e EUR, got %d resultados", len(results))
	}

	for _, r := range results {
		if r.Name == "" {
			t.Fatalf("Name vazio em %+v", r)
		}
		if r.Bid <= 0 || r.Bid > 10000 {
			t.Errorf("cotação suspeita: %s bid = %v", r.Name, r.Bid)
		}
		t.Logf("%s: R$ %.2f (%s)", r.Name, r.Bid, r.UpdatedAt)
	}
}
