// Package ia integra Groq (persona Shinobu), decisão de busca web e Tavily.
package ia

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/Turgho/YuukoWhatsapp/internal/domain/history"
)

// IARequest é o corpo JSON enviado à API compatível com OpenAI (Groq).
type IARequest struct {
	Model       string              `json:"model"`
	Messages    []history.IAMessage `json:"messages"`
	Stream      bool                `json:"stream"`
	Temperature float64             `json:"temperature"`
	MaxTokens   int                 `json:"max_tokens"`
}

// IAResponse é o formato mínimo de resposta usado após Decode.
type IAResponse struct {
	Choices []struct {
		Message history.IAMessage `json:"message"`
	} `json:"choices"`
}

// AskIA monta contexto (personalidade, histórico opcional, Tavily opcional) e chama o Groq.
//
// Ordem de decisão (economia de tokens e I/O):
//  1. Palavras-chave fortes → tenta Tavily direto, sem classificador Groq.
//  2. Mensagens triviais (cumprimento curto) → sem busca nem classificador.
//  3. Caso contrário → classificador leve no Scout (prompt truncado, system mínimo).
//  4. Histórico do SQLite só entra se não houver contexto web (evita ler dados que seriam descartados).
//
// chat = chave da conversa (JID do privado ou do grupo).
func AskIA(ctx context.Context, chat, prompt string, isOwner bool, sender string, store *history.Store) (string, bool, error) {
	groqURL := os.Getenv("GROQ_URL")
	groqKey := os.Getenv("GROQ_API_KEY")

	prompt = cleanPrompt(prompt)
	styleMode := classifyPromptMode(prompt)

	// 1–3: decisão de busca antes de tocar no histórico.
	needSearch := shouldSearchWeb(ctx, prompt)
	var webContext string
	var usedSearch bool
	if needSearch {
		if wc, err := searchWeb(ctx, prompt); err == nil && strings.TrimSpace(wc) != "" {
			usedSearch = true
			webContext = truncateText(strings.TrimSpace(wc), 1500)
		}
	}

	// Monta o modo de resposta
	mode := styleMode
	if usedSearch {
		mode = ModeWeb
	}

	// Monta o sistema base de mensagens
	messages := baseSystemMessages(mode, isOwner)
	if !usedSearch {
		messages = appendPersistentAndRecent(ctx, messages, chat, store)
	}

	// Monta o conteúdo do usuário
	userContent := buildUserContent(prompt, mode, webContext)
	messages = append(messages, history.IAMessage{Role: "user", Content: userContent})

	// Chama o Groq
	params := mainAnswerParams(mode, usedSearch)
	answer, err := callGroq(ctx, groqURL, groqKey, params.model, messages, params.temperature, params.maxTokens)
	if err != nil {
		return "", usedSearch, err
	}

	// Atualiza o resumo da conversa
	if store != nil && chat != "" {
		go refreshChatSummary(chat, prompt, answer, store)
	}

	return answer, usedSearch, nil
}

// baseSystemMessages monta system prompt da Shinobu e, se for o dono, instrução extra de tom.
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

// appendPersistentAndRecent acrescenta resumo salvo e últimas mensagens ao slice (fluxo sem web).
func appendPersistentAndRecent(ctx context.Context, messages []history.IAMessage, chat string, store *history.Store) []history.IAMessage {
	if store == nil || chat == "" {
		return messages
	}
	// Acrescenta o resumo persistente da conversa
	if summary, err := store.GetSummary(ctx, chat); err == nil && strings.TrimSpace(summary) != "" {
		summary = truncateText(strings.TrimSpace(summary), 1200)
		messages = append(messages, history.IAMessage{
			Role:    "system",
			Content: "Resumo persistente da conversa anterior:\n" + summary,
		})
	}
	// Transcrição compacta (1× system) em vez de várias mensagens user/assistant: menos overhead
	// de roles e marcas temporais ajudam o modelo a situar o que é recente.
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

// callGroq envia mensagens ao modelo indicado e devolve o texto da primeira resposta.
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
