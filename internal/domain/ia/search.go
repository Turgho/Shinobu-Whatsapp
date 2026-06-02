package ia

import (
	"context"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/history"
)

const maxRunesSearchClassifier = 400

// shouldSearchWeb decide em 3 estágios se o prompt precisa de busca web:
// 1. Palavras-chave explícitas (preço, notícia, clima, etc.) → sim.
// 2. Frases triviais curtas sem '?' → não precisa.
// 3. Classificador Groq (modelo rápido, temperature 0) → sim/nao.
func shouldSearchWeb(ctx context.Context, cfg *Config, prompt string) bool {
	lower := strings.ToLower(prompt)
	if hasWebSearchCue(lower) {
		return true
	}
	if trivialWithoutSearch(lower) {
		return false
	}
	return classifyNeedsWebSearch(ctx, cfg, prompt)
}

// trivialWithoutSearch retorna true para saudações, acenos, confirmações curtas
// que não fazem sentido buscar na internet. Evita chamada desnecessária à Groq.
func trivialWithoutSearch(lower string) bool {
	s := strings.TrimSpace(lower)
	if s == "" {
		return true
	}
	acks := []string{
		"oi", "olá", "ola", "eae", "eai", "hey", "hi", "hello",
		"kkk", "kkkk", "rsrs", "haha", "blz", "beleza", "vlw", "valeu",
		"obrigado", "obrigada", "obg", "brigado", "brigada",
		"sim", "não", "nao", "ok", "okay", "certo",
		"tá", "ta", "bom dia", "boa tarde", "boa noite",
	}
	if slices.Contains(acks, s) {
		return true
	}
	if !strings.Contains(s, "?") && utf8.RuneCountInString(s) <= 20 && len(strings.Fields(s)) <= 2 {
		return !hasWebSearchCue(s)
	}
	return false
}

// classifyNeedsWebSearch usa um modelo Groq rápido (llama-3.1-8b) com temperature 0
// para classificar se o prompt precisa de dados atualizados da internet.
// Trunca o prompt para 400 runes para evitar gastar tokens desnecessariamente.
func classifyNeedsWebSearch(ctx context.Context, cfg *Config, prompt string) bool {
	if cfg.GroqURL == "" || cfg.GroqKey == "" {
		return false
	}

	promptCut := truncateText(strings.TrimSpace(prompt), maxRunesSearchClassifier)

	req := IARequest{
		Model: modelScoutFast,
		Messages: []history.IAMessage{
			{
				Role: "system",
				Content: "Responda só sim ou nao. sim = precisa de dados atuais da internet (preço, notícia, clima, esporte, lançamento, fatos que mudam). " +
					"nao = conversa geral, opinião, conhecimento estável.",
			},
			{Role: "user", Content: promptCut},
		},
		Stream:      false,
		Temperature: 0,
		MaxTokens:   3,
	}

	resp, err := groqChat(ctx, cfg.GroqURL, cfg.GroqKey, req)
	if err != nil {
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(resp.Choices[0].Message.Content))
	return strings.HasPrefix(answer, "sim")
}
