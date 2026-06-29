package commands

import (
	"context"
	"time"

	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/gosafe"
)

type rateLimitEntry struct {
	count   int
	resetAt time.Time
}

// StartRateLimitCleanup inicia goroutine que remove entradas expiradas do
// rateLimitMap a cada 10 minutos, evitando memory leak em chats ativos.
func (r *Router) StartRateLimitCleanup(ctx context.Context) {
	gosafe.Go(r.log, func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				r.cleanRateLimit()
			}
		}
	})
}

func (r *Router) cleanRateLimit() {
	r.rateLimitMu.Lock()
	defer r.rateLimitMu.Unlock()

	now := time.Now()
	for key, entry := range r.rateLimitMap {
		if now.After(entry.resetAt) {
			delete(r.rateLimitMap, key)
		}
	}
}

// checkRateLimit implementa rate limiting por chave (sender JID).
// Usa janela fixa: cada chave tem um contador e resetAt.
// Quando resetAt passa, o contador é zerado.
// Se o contador excede o máximo, a requisição é bloqueada.
func (r *Router) checkRateLimit(key string) bool {
	r.rateLimitMu.Lock()
	defer r.rateLimitMu.Unlock()

	now := time.Now()
	entry, ok := r.rateLimitMap[key]
	if !ok || now.After(entry.resetAt) {
		r.rateLimitMap[key] = &rateLimitEntry{
			count:   1,
			resetAt: now.Add(r.rateLimitEvery),
		}
		return true
	}

	entry.count++
	if entry.count > r.rateLimitMax {
		return false
	}
	return true
}

func (r *Router) SetRateLimit(max int, every time.Duration) {
	r.rateLimitMax = max
	r.rateLimitEvery = every
}
