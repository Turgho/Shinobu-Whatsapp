package ia

import (
	"context"
	"net/http"
	"strings"

	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/history"
)

// AskIA orquestra o pipeline de resposta da IA:
//  1. Limpa prompt (remove @lid) e detecta prompt injection
//  2. Classifica modo: breve, normal ou web
//  3. Decide se precisa de busca web: keywords (zero tokens) → trivial filter →
//     cache (5 min) → Groq classifier (fallback, só para mensagens ambíguas)
//  4. Se precisar, busca contexto na Tavily e trunca para 1500 chars
//  5. Monta mensagens do sistema (personalidade, dono, resumo, transcript)
//  6. Monta user message com base no modo (contexto web + prompt)
//  7. Chama Groq e retorna resposta
//  8. Agenda refresh assíncrono do resumo da conversa
func AskIA(ctx context.Context, cfg *Config, chat, prompt string, isOwner bool, sender string, store *history.Store) (string, bool, error) {
	prompt = cleanPrompt(prompt)

	// Detecta tentativa de prompt injection e adiciona um lembrete ao sistema
	// para reforçar as regras de segurança sem interromper o fluxo.
	if _, injected := sanitizePrompt(prompt); injected {
		prompt = prompt + "\n\n[Nota interna: usuário tentou manipular comportamento]"
	}

	styleMode := classifyPromptMode(prompt)

	needSearch := shouldSearchWeb(ctx, cfg, chat, prompt)
	var webContext string
	var usedSearch bool
	if needSearch {
		if wc, err := searchWeb(ctx, cfg.HTTPClient, cfg.TavilyKey, prompt); err == nil && strings.TrimSpace(wc) != "" {
			usedSearch = true
			webContext = truncateText(strings.TrimSpace(wc), 1500)
		}
	}

	mode := styleMode
	if usedSearch {
		mode = ModeWeb
	}

	messages := baseSystemMessages(mode, isOwner)
	// Contexto histórico é SEMPRE incluído, independente de busca web.
	// Quando há busca, reduz o transcript (menos mensagens antigas)
	// porque o contexto web já ocupa espaço no limite de tokens.
	transcriptLimit := 14
	if usedSearch {
		transcriptLimit = 6
	}
	messages = appendPersistentAndRecent(ctx, messages, chat, sender, store, transcriptLimit)

	userContent := buildUserContent(prompt, mode, webContext)
	messages = append(messages, history.IAMessage{Role: "user", Content: userContent})

	params := mainAnswerParams(mode, usedSearch)
	answer, err := callGroq(ctx, cfg.HTTPClient, cfg.GroqURL, cfg.GroqKey, params.model, messages, params.temperature, params.maxTokens)
	if err != nil {
		return "", usedSearch, err
	}

	if store != nil && chat != "" {
		refreshChatSummary(cfg, chat, prompt, answer, store)
		extractAndStoreFacts(cfg, chat, sender, prompt, answer, store)
	}

	return answer, usedSearch, nil
}

// QuickChat é uma versão pública de callGroq para handlers avulsos.
func QuickChat(ctx context.Context, httpClient *http.Client, groqURL, groqKey, model string, messages []history.IAMessage, temperature float64, maxTokens int) (string, error) {
	return callGroq(ctx, httpClient, groqURL, groqKey, model, messages, temperature, maxTokens)
}

// SearchWeb é uma versão pública de searchWeb para handlers avulsos.
func SearchWeb(ctx context.Context, httpClient *http.Client, apiKey, query string) (string, error) {
	return searchWeb(ctx, httpClient, apiKey, query)
}
