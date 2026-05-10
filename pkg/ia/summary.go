package ia

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Turgho/YuukoWhatsapp/pkg/history"
)

// refreshChatSummary atualiza a memória resumida da conversa em segundo plano.
func refreshChatSummary(chat, lastUserPrompt, lastAssistantAnswer string, store *history.Store) {
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()

	// Só resume quando a conversa já cresceu o suficiente.
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
}

// generateConversationSummary pede para a IA atualizar o resumo persistente da conversa.
func generateConversationSummary(
	ctx context.Context,
	currentSummary string,
	recent []history.IAMessage,
	lastUserPrompt string,
	lastAssistantAnswer string,
) (string, error) {
	groqURL := os.Getenv("GROQ_URL")
	groqKey := os.Getenv("GROQ_API_KEY")

	recentText := formatMessagesForSummary(recent)

	userContent := fmt.Sprintf(
		`Resumo atual da conversa:
%s

Mensagens recentes:
%s

Última mensagem do usuário:
%s

Última resposta da IA:
%s

Tarefa:
Atualize o resumo da conversa para uso futuro pelo bot.

Regras:
- Mantenha apenas fatos úteis e estáveis.
- Preserve preferências, temas recorrentes, contexto em andamento, nomes, intenções e tarefas.
- Remova mensagens repetitivas, piadas soltas e detalhes inúteis.
- Escreva em português do Brasil.
- Use no máximo 10 bullets curtos.
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

	return callGroq(ctx, groqURL, groqKey, "meta-llama/llama-4-scout-17b-16e-instruct", msgs, 0, 220)
}

// formatMessagesForSummary converte mensagens recentes em texto curto para o prompt de resumo.
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
