package ia

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/history"
	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/gosafe"
	"go.uber.org/zap"
)

var (
	factLocks   sync.Mutex
	factRunning = make(map[string]struct{})
)

type extractedFact struct {
	Fact       string `json:"fact"`
	Confidence string `json:"confidence"`
}

// extractAndStoreFacts extrai fatos atômicos sobre o usuário da última interação
// e persiste via store.UpsertFact. Executa em goroutine separada (não bloqueia
// a resposta da IA). Usa dedup por chat+user para evitar extrações simultâneas.
func extractAndStoreFacts(cfg *Config, chat, senderJID, lastUserPrompt, lastAssistantAnswer string, store *history.Store) {
	logger := cfg.Log
	if logger == nil {
		logger = zap.NewNop()
	}

	dedupKey := chat + "\x00" + senderJID

	factLocks.Lock()
	if _, running := factRunning[dedupKey]; running {
		factLocks.Unlock()
		return
	}
	factRunning[dedupKey] = struct{}{}
	factLocks.Unlock()

	gosafe.Go(logger, func() {
		defer func() {
			factLocks.Lock()
			delete(factRunning, dedupKey)
			factLocks.Unlock()
		}()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		userContent := fmt.Sprintf(`Mensagem do usuário (JID: %s):

%s
Resposta da IA:

%s
Extraia fatos discretos e estáveis sobre o usuário desta conversa.

Retorne APENAS um array JSON. Se não houver fatos úteis, retorne [].
Formato:
[{"fact": "prefere músicas de rock", "confidence": "high"}, {"fact": "mora em São Paulo", "confidence": "medium"}]

Regras:
- Apenas fatos sobre o usuário, não sobre a conversa em si
- Fatos estáveis (preferências, localização, nome, hábitos)
- Ignorar perguntas casuais sem contexto pessoal
- Máximo 3 fatos por extração
- confidence: "high" se explícito, "medium" se inferido, "low" se incerto`,
			senderJID, lastUserPrompt, lastAssistantAnswer,
		)

		msgs := []history.IAMessage{
			{Role: "system", Content: "Você extrai fatos sobre usuários de conversas."},
			{Role: "user", Content: userContent},
		}

		raw, err := callGroq(ctx, cfg.HTTPClient, cfg.GroqURL, cfg.GroqKey, modelFastClass, msgs, 0, 200)
		if err != nil {
			logger.Debug("extractAndStoreFacts: groq error", zap.Error(err))
			return
		}

		raw = strings.TrimSpace(raw)
		raw = strings.TrimPrefix(raw, "```json")
		raw = strings.TrimPrefix(raw, "```")
		raw = strings.TrimSuffix(raw, "```")
		raw = strings.TrimSpace(raw)

		if raw == "" || raw == "[]" {
			return
		}

		var facts []extractedFact
		if err := json.Unmarshal([]byte(raw), &facts); err != nil {
			logger.Debug("extractAndStoreFacts: json parse error", zap.String("raw", raw), zap.Error(err))
			return
		}

		for _, f := range facts {
			conf := f.Confidence
			if conf == "" {
				conf = "medium"
			}
			if err := store.UpsertFact(ctx, chat, senderJID, f.Fact, conf); err != nil {
				logger.Debug("extractAndStoreFacts: upsert error", zap.Error(err))
				continue
			}
			logger.Info("Fato extraído",
				zap.String("chat", chat),
				zap.String("user", senderJID),
				zap.String("fact", f.Fact),
				zap.String("confidence", conf),
			)
		}
	})
}
