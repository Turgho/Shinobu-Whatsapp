package ia

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/history"
	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/gosafe"
)

// refreshChatSummary atualiza o resumo persistente da conversa de forma ASSÍNCRONA
// (goroutine separada com recover). Só gera novo resumo se store.NeedsSummary
// indicar que passou tempo ou mensagens suficientes desde a última atualização.
// Não bloqueia a resposta da IA para o usuário.
func refreshChatSummary(cfg *Config, chat, lastUserPrompt, lastAssistantAnswer string, store *history.Store) {
	gosafe.Go(func() {
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
