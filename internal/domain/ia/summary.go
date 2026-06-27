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
	summaryLocks   sync.Mutex
	summaryRunning = make(map[string]struct{})

	factLocks   sync.Mutex
	factRunning = make(map[string]struct{})
)

// refreshChatSummary atualiza o resumo persistente da conversa de forma ASSÍNCRONA
// (goroutine separada com recover). Usa dedup por chat para evitar múltiplas
// requisições simultâneas para o mesmo chat.
// Só gera novo resumo se store.NeedsSummary indicar que passou tempo ou mensagens
// suficientes desde a última atualização.
// Não bloqueia a resposta da IA para o usuário.
func refreshChatSummary(cfg *Config, chat, lastUserPrompt, lastAssistantAnswer string, store *history.Store) {
	logger := cfg.Log
	if logger == nil {
		logger = zap.NewNop()
	}

	summaryLocks.Lock()
	if _, running := summaryRunning[chat]; running {
		summaryLocks.Unlock()
		return
	}
	summaryRunning[chat] = struct{}{}
	summaryLocks.Unlock()

	gosafe.Go(logger, func() {
		defer func() {
			summaryLocks.Lock()
			delete(summaryRunning, chat)
			summaryLocks.Unlock()
		}()
		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		defer cancel()

		need, err := store.NeedsSummary(ctx, chat, 30, 24*time.Hour)
		if err != nil || !need {
			return
		}

		currentSummary, err := store.GetSummary(ctx, chat)
		if err != nil {
			currentSummary = ""
		}

		recent, err := store.RecentMessages(ctx, chat, 20, 24*time.Hour)
		if err != nil {
			return
		}

		summary, err := generateConversationSummary(
			ctx,
			cfg,
			currentSummary,
			recent,
			lastUserPrompt,
			lastAssistantAnswer,
		)
		if err != nil {
			return
		}

		summary = strings.TrimSpace(summary)
		summary = strings.TrimPrefix(summary, "```")
		summary = strings.TrimSuffix(summary, "```")
		summary = strings.TrimSpace(summary)

		if summary == "" || strings.EqualFold(summary, "vazio") {
			return
		}

		summary = truncateText(summary, 1400)

		_ = store.SetSummary(ctx, chat, summary)
	})
}

func generateConversationSummary(
	ctx context.Context,
	cfg *Config,
	currentSummary string,
	recent []history.IAMessage,
	lastUserPrompt string,
	lastAssistantAnswer string,
) (string, error) {
	recentText := formatMessagesForSummary(recent)

	userContent := fmt.Sprintf(
		`Resumo atual da conversa (NÃO REMOVA estes itens):
%s

Mensagens recentes:
%s

Última mensagem do usuário:
%s

Última resposta da IA:
%s

Tarefa:
MANTENHA todos os bullets do resumo atual que ainda são relevantes.
ADICIONE novos bullets apenas para fatos novos e importantes que apareceram nas mensagens recentes.
NÃO remova bullets existentes a menos que estejam claramente obsoletos.

Regras:
- Mantenha apenas fatos úteis e estáveis.
- Preserve preferências, temas recorrentes, contexto em andamento, nomes, intenções e tarefas.
- Remova mensagens repetitivas, piadas soltas e detalhes inúteis.
- Escreva em português do Brasil.
- Use no máximo 12 bullets curtos.
- Não explique o que você está fazendo.
- Se não houver nada útil para guardar, responda exatamente: vazio.`,
		currentSummary,
		recentText,
		lastUserPrompt,
		lastAssistantAnswer,
	)

	msgs := []history.IAMessage{
		{
			Role: "system",
			Content: "Você transforma conversas em memória curta útil para um bot de WhatsApp. " +
				"Seja compacto, fiel e objetivo.",
		},
		{
			Role:    "user",
			Content: userContent,
		},
	}

	return callGroq(ctx, cfg.GroqURL, cfg.GroqKey, "meta-llama/llama-4-scout-17b-16e-instruct", msgs, 0, 220)
}

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

		raw, err := callGroq(ctx, cfg.GroqURL, cfg.GroqKey, modelScoutFast, msgs, 0, 200)
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

func formatMessagesForSummary(msgs []history.IAMessage) string {
	if len(msgs) == 0 {
		return "(sem mensagens recentes)"
	}

	var b strings.Builder
	for _, msg := range msgs {
		role := "Usuário"
		switch msg.Role {
		case "assistant":
			role = "Assistente"
		case "system":
			role = "Sistema"
		}
		content := strings.TrimSpace(msg.Content)
		if content == "" {
			continue
		}
		b.WriteString(role)
		b.WriteString(": ")
		b.WriteString(content)
		b.WriteString("\n")
	}

	out := strings.TrimSpace(b.String())
	if out == "" {
		return "(sem mensagens úteis)"
	}
	return out
}
