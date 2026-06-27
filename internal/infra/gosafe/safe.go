package gosafe

import (
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
				logger.Error("panic recuperado", zap.Any("panic", r))
			}
		}()
		fn()
	}()
}
