package ia

import (
	"context"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/history"
	"go.uber.org/zap"
)

const maxRunesSearchClassifier = 400

// searchCache evita re-classificar prompts que claramente continuam
// o tópico de uma busca recém-feita no mesmo chat.
// Ex: usuário pergunta "qual a cotação do dólar?" e 20s depois "e do euro?".
// A segunda pergunta compartilha o contexto de "cotação" e não precisa
// de uma nova chamada ao classificador Groq.
var searchCache struct {
	mu sync.Mutex
	m  map[string]searchCacheEntry
}

type searchCacheEntry struct {
	lastResult bool
	updatedAt  time.Time
}

const searchCacheTTL = 5 * time.Minute

// shouldSearchWeb decide em 3 estágios se o prompt precisa de busca web:
// 1. Palavras-chave explícitas (preço, notícia, clima, etc.) → sim.
// 2. Frases triviais curtas sem '?' → não precisa.
// 3. Cache por chat: se já classificamos este chat nos últimos N min, reusa.
// 4. Classificador Groq (modelo rápido, temperature 0) → sim/nao.
func shouldSearchWeb(ctx context.Context, cfg *Config, chat, prompt string) bool {
	lower := strings.ToLower(prompt)
	if hasWebSearchCue(lower) {
		return true
	}
	if trivialWithoutSearch(lower) {
		return false
	}

	// Cache: se já classificamos este chat recentemente, reusa.
	if result, ok := getCachedSearch(chat); ok {
		return result
	}

	result := classifyNeedsWebSearch(ctx, cfg, prompt)
	setCachedSearch(chat, result)
	return result
}

// getCachedSearch retorna o resultado em cache se ainda for válido.
func getCachedSearch(chat string) (bool, bool) {
	searchCache.mu.Lock()
	defer searchCache.mu.Unlock()

	e, ok := searchCache.m[chat]
	if !ok || time.Since(e.updatedAt) > searchCacheTTL {
		return false, false
	}
	return e.lastResult, true
}

func setCachedSearch(chat string, result bool) {
	searchCache.mu.Lock()
	defer searchCache.mu.Unlock()

	if searchCache.m == nil {
		searchCache.m = make(map[string]searchCacheEntry)
	}
	searchCache.m[chat] = searchCacheEntry{
		lastResult: result,
		updatedAt:  time.Now(),
	}
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

// classifyNeedsWebSearch usa um modelo Groq rápido com temperature 0
// para classificar se o prompt precisa de dados atualizados da internet.
// Trunca o prompt para 400 runes para evitar gastar tokens desnecessariamente.
// Só é chamada quando hasWebSearchCue retornou false e a mensagem não é trivial.
func classifyNeedsWebSearch(ctx context.Context, cfg *Config, prompt string) bool {
	if cfg.GroqURL == "" || cfg.GroqKey == "" {
		return false
	}

	promptCut := truncateText(strings.TrimSpace(prompt), maxRunesSearchClassifier)

	req := IARequest{
		Model: modelFastClass,
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
		MaxTokens:   10,
	}

	resp, err := groqChat(ctx, cfg.HTTPClient, cfg.GroqURL, cfg.GroqKey, req)
	if err != nil {
		if cfg.Log != nil {
			cfg.Log.Warn("Classificador de busca web falhou, assumindo false", zap.Error(err))
		}
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(resp.Choices[0].Message.Content))
	return strings.HasPrefix(answer, "sim")
}
