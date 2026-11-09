// Testes do rate limiter por sender (janela fixa) — sem rede nem whatsmeow real.
package commands

import (
	"testing"
	"time"

	"go.uber.org/zap"
)

func newTestRouter(t *testing.T) *Router {
	t.Helper()
	return NewRouter("!", nil, zap.NewNop(), nil, nil)
}

func TestCheckRateLimitWindow(t *testing.T) {
	r := newTestRouter(t)
	r.SetRateLimit(3, time.Minute)

	// 1ª..3ª dentro da janela: permitidas
	for i := 1; i <= 3; i++ {
		if !r.checkRateLimit("user@g.us") {
			t.Fatalf("requisição %d dentro da janela deveria passar", i)
		}
	}
	// 4ª excede o máximo: bloqueada
	if r.checkRateLimit("user@g.us") {
		t.Fatal("4ª requisição deveria ser bloqueada")
	}
	// janela é por chave: outro sender tem contagem própria
	if !r.checkRateLimit("other@g.us") {
		t.Fatal("chave distinta deveria ter janela própria")
	}
}

func TestCheckRateLimitWindowReset(t *testing.T) {
	r := newTestRouter(t)
	r.SetRateLimit(1, 30*time.Millisecond)

	if !r.checkRateLimit("user@g.us") {
		t.Fatal("primeira deveria passar")
	}
	if r.checkRateLimit("user@g.us") {
		t.Fatal("segunda imediata deveria ser bloqueada (max=1)")
	}

	// janela expira → contador reseta
	time.Sleep(60 * time.Millisecond)
	if !r.checkRateLimit("user@g.us") {
		t.Fatal("depois de expirar a janela deveria passar de novo")
	}
}

func TestCleanRateLimitRemovesExpired(t *testing.T) {
	r := newTestRouter(t)

	// entrada válida (janela longa) e entrada expirada
	r.checkRateLimit("live@g.us")
	r.rateLimitMap["dead@g.us"] = &rateLimitEntry{count: 5, resetAt: time.Now().Add(-time.Hour)}

	r.cleanRateLimit()

	if _, ok := r.rateLimitMap["dead@g.us"]; ok {
		t.Fatal("entrada expirada deveria ser removida")
	}
	if _, ok := r.rateLimitMap["live@g.us"]; !ok {
		t.Fatal("entrada válida deveria permanecer")
	}
}

func TestSetRateLimit(t *testing.T) {
	r := newTestRouter(t)
	r.SetRateLimit(5, 2*time.Minute)

	if r.rateLimitMax != 5 || r.rateLimitEvery != 2*time.Minute {
		t.Fatalf("setRateLimit = (%d, %s)", r.rateLimitMax, r.rateLimitEvery)
	}
}
