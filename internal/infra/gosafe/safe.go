package gosafe

import (
	"sync"

	"go.uber.org/zap"
)

// Go executa fn em uma goroutine com recover.
// Panics são logados com o logger fornecido. Se nil, usa zap.NewNop().
func Go(logger *zap.Logger, fn func()) {
	if logger == nil {
		logger = zap.NewNop()
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("panic recuperado", zap.Any("panic", r), zap.Stack("stack"))
			}
		}()
		fn()
	}()
}

// Group roda tarefas em goroutines e permite esperar todas no encerramento.
// Útil no shutdown: sem ele, o processo morre com tarefas no ar (download
// do yt-dlp, upload de mídia, resumo da IA).
type Group struct {
	wg sync.WaitGroup
}

// Go executa fn em uma goroutine (com recover) e registra na Group.
func (g *Group) Go(logger *zap.Logger, fn func()) {
	if logger == nil {
		logger = zap.NewNop()
	}
	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		defer func() {
			if r := recover(); r != nil {
				logger.Error("panic recuperado", zap.Any("panic", r), zap.Stack("stack"))
			}
		}()
		fn()
	}()
}

// Wait espera todas as tarefas terminarem.
func (g *Group) Wait() {
	g.wg.Wait()
}
