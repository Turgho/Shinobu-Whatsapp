package ia

import (
	"context"
	"os"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/Turgho/YuukoWhatsapp/pkg/history"
)

// maxRunesSearchClassifier limita o texto enviado ao classificador (só precisa do tema da pergunta).
const maxRunesSearchClassifier = 400

// shouldSearchWeb decide se vale tentar Tavily antes da resposta principal.
// Fluxo: palavras-chave → heurística “trivial” (pula Groq) → classificador mínimo no Scout.
func shouldSearchWeb(ctx context.Context, prompt string) bool {
	lower := strings.ToLower(prompt)
	if hasWebSearchCue(lower) {
		return true
	}
	if trivialWithoutSearch(lower) {
		return false
	}
	return classifyNeedsWebSearch(ctx, prompt)
}

// trivialWithoutSearch evita gastar tokens em cumprimentos e réplicas curtas sem interrogação.
// Não bloqueia quando já existe sinal forte de web (hasWebSearchCue), tratado antes.
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
	// Frase mínima sem interrogação: só trata como trivial com até 2 tokens (evita bloquear perguntas curtas factuais).
	if !strings.Contains(s, "?") && utf8.RuneCountInString(s) <= 20 && len(strings.Fields(s)) <= 2 {
		return !hasWebSearchCue(s)
	}
	return false
}

// classifyNeedsWebSearch é uma chamada barata ao Scout: system curto, poucos tokens de saída.
func classifyNeedsWebSearch(ctx context.Context, prompt string) bool {
	groqURL := strings.TrimSpace(os.Getenv("GROQ_URL"))
	groqKey := strings.TrimSpace(os.Getenv("GROQ_API_KEY"))
	if groqURL == "" || groqKey == "" {
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

	resp, err := groqChat(ctx, groqURL, groqKey, req)
	if err != nil {
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(resp.Choices[0].Message.Content))
	return strings.HasPrefix(answer, "sim")
}
