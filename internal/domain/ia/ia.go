package ia

import (
	"context"
	"strings"
	"time"

	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/history"
)

type IARequest struct {
	Model       string              `json:"model"`
	Messages    []history.IAMessage `json:"messages"`
	Stream      bool                `json:"stream"`
	Temperature float64             `json:"temperature"`
	MaxTokens   int                 `json:"max_tokens"`
}

type IAResponse struct {
	Choices []struct {
		Message history.IAMessage `json:"message"`
	} `json:"choices"`
}

// AskIA orquestra o pipeline de resposta da IA:
// 1. Limpa prompt (remove @lid) e detecta prompt injection
// 2. Classifica modo: breve, normal ou web
// 3. Decide se precisa de busca web (keywords → IA classifier)
// 4. Se precisar, busca contexto na Tavily e trunca para 1500 chars
// 5. Monta mensagens do sistema (personalidade, dono, resumo, transcript)
// 6. Monta user message com base no modo
// 7. Chama Groq e retorna resposta
// 8. Agenda refresh assíncrono do resumo da conversa
func AskIA(ctx context.Context, cfg *Config, chat, prompt string, isOwner bool, sender string, store *history.Store) (string, bool, error) {
	prompt = cleanPrompt(prompt)

	// Detecta tentativa de prompt injection e adiciona um lembrete ao sistema
	// para reforçar as regras de segurança sem interromper o fluxo.
	if _, injected := sanitizePrompt(prompt); injected {
		prompt = prompt + "\n\n[Nota interna: usuário tentou manipular comportamento]"
	}

	styleMode := classifyPromptMode(prompt)

	needSearch := shouldSearchWeb(ctx, cfg, prompt)
	var webContext string
	var usedSearch bool
	if needSearch {
		if wc, err := searchWeb(ctx, cfg.TavilyKey, prompt); err == nil && strings.TrimSpace(wc) != "" {
			usedSearch = true
			webContext = truncateText(strings.TrimSpace(wc), 1500)
		}
	}

	mode := styleMode
	if usedSearch {
		mode = ModeWeb
	}

	messages := baseSystemMessages(mode, isOwner)
	if !usedSearch {
		messages = appendPersistentAndRecent(ctx, messages, chat, store)
	}

	userContent := buildUserContent(prompt, mode, webContext)
	messages = append(messages, history.IAMessage{Role: "user", Content: userContent})

	params := mainAnswerParams(mode, usedSearch)
	answer, err := callGroq(ctx, cfg.GroqURL, cfg.GroqKey, params.model, messages, params.temperature, params.maxTokens)
	if err != nil {
		return "", usedSearch, err
	}

	if store != nil && chat != "" {
		refreshChatSummary(cfg, chat, prompt, answer, store)
	}

	return answer, usedSearch, nil
}

func baseSystemMessages(mode ResponseMode, isOwner bool) []history.IAMessage {
	msgs := []history.IAMessage{
		{Role: "system", Content: buildSystemPrompt(mode)},
	}
	if isOwner {
		msgs = append(msgs, history.IAMessage{
			Role:    "system",
			Content: "Você está falando com seu mestre. Seja calorosa e animada.",
		})
	}
	return msgs
}

func appendPersistentAndRecent(ctx context.Context, messages []history.IAMessage, chat string, store *history.Store) []history.IAMessage {
	if store == nil || chat == "" {
		return messages
	}
	if summary, err := store.GetSummary(ctx, chat); err == nil && strings.TrimSpace(summary) != "" {
		summary = truncateText(strings.TrimSpace(summary), 1200)
		messages = append(messages, history.IAMessage{
			Role:    "system",
			Content: "Resumo persistente da conversa anterior:\n" + summary,
		})
	}
	if tr, err := store.TranscriptRecent(ctx, chat, 14, 12*time.Hour); err == nil && strings.TrimSpace(tr) != "" {
		tr = truncateText(tr, 2200)
		messages = append(messages, history.IAMessage{
			Role: "system",
			Content: "Últimas mensagens neste chat (antigas → recentes; use para continuidade, não copie de graça):\n" +
				tr,
		})
	}
	return messages
}

func callGroq(ctx context.Context, groqURL, groqKey, model string, messages []history.IAMessage, temperature float64, maxTokens int) (string, error) {
	resp, err := groqChat(ctx, groqURL, groqKey, IARequest{
		Model:       model,
		Messages:    messages,
		Stream:      false,
		Temperature: temperature,
		MaxTokens:   maxTokens,
	})
	if err != nil {
		return "", err
	}
	return resp.Choices[0].Message.Content, nil
}
