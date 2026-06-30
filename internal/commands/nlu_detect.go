package commands

import (
	"strings"

	"go.mau.fi/whatsmeow/types/events"
	"go.uber.org/zap"
)

// groupNLTriggers são verbos de intenção que disparam NLU em grupos
// quando nlpGroupTrigger está ativo. Case-insensitive, checados via
// HasPrefix após lower + trim.
var groupNLTriggers = []string{
	"coloca", "toca", "baixa", "manda", "me manda",
	"qual", "quero", "preciso", "como", "quando", "onde",
	"tem ", "vai ter", "vai chover", "lembra", "agenda",
	"me lembra", "pesquisa", "busca", "procura",
}

// groupNLDirectAddress são padrões de endereçamento direto que disparam
// NLU em grupos quando nlpGroupTrigger está ativo. Checados via Contains.
var groupNLDirectAddress = []string{
	"me fala", "me diz", "me manda", "você sabe", "sabe me dizer",
}

// dispatchableCommands são os comandos públicos que a NLU pode despachar.
// Comandos de admin/owner não entram aqui — exigem verificação de dono.
var dispatchableCommands = map[string]bool{
	"clima":       true,
	"play":        true,
	"sticker":     true,
	"efeito":      true,
	"aniversário": true,
	"agenda":      true,
	"cotacao":     true,
	"feriado":     true,
	"noticia":     true,
	"receita":     true,
	"piada":       true,
	"fato":        true,
	"filme":       true,
	"contagem":    true,
	"unsticker":   true,
	"traduz":      true,
}

func dispatchableCommand(name string) bool {
	return dispatchableCommands[name]
}

// isNLApplicable decide se uma mensagem merece processamento NLU.
// DMs sempre disparam NLU. Em grupos, o comportamento depende de nlpGroupTrigger:
//   - false (default): só dispara se Shinobu foi mencionada explicitamente
//   - true: ativa heurísticas adicionais (pergunta, verbos de intenção, endereçamento direto)
func (r *Router) isNLApplicable(evt *events.Message, msg string) bool {
	chat := evt.Info.Chat.String()
	isGroup := strings.HasSuffix(chat, "@g.us")
	if !isGroup {
		return true
	}

	// Heurística 1: menção explícita ao nome ou @jid — sempre válida
	if isMentioned(msg, r.botJID) {
		r.log.Debug("NLU triggered", zap.String("reason", "mention"), zap.String("chat", chat))
		return true
	}

	// Sem nlpGroupTrigger: em grupo, só responde se mencionada
	if !r.nlpGroupTrigger {
		r.log.Debug("NLU ignorado em grupo: Shinobu não mencionada",
			zap.String("chat", chat),
			zap.String("msg", msg[:min(40, len(msg))]),
		)
		return false
	}

	// --- Heurísticas abaixo só são aplicáveis com nlpGroupTrigger=true ---

	lower := strings.ToLower(msg)
	words := strings.Fields(lower)
	wordCount := len(words)

	// Mensagens muito curtas (<3 palavras) sem "?" são ignoradas
	// para evitar responder "kkk", "boa!", "tá bom" etc.
	hasQuestion := strings.HasSuffix(strings.TrimSpace(msg), "?")
	if wordCount < 3 && !hasQuestion {
		return false
	}

	// Heurística 2: pergunta direta terminando em "?" com conteúdo
	if hasQuestion && wordCount > 4 {
		r.log.Debug("NLU triggered", zap.String("reason", "question"), zap.String("chat", chat))
		return true
	}

	// Heurística 3: começa com verbo de intenção
	trimmed := strings.TrimSpace(lower)
	for _, verb := range groupNLTriggers {
		if strings.HasPrefix(trimmed, verb) {
			r.log.Debug("NLU triggered", zap.String("reason", "intent_verb"), zap.String("verb", verb), zap.String("chat", chat))
			return true
		}
	}

	// Heurística 4: contém padrão de endereçamento direto
	for _, pattern := range groupNLDirectAddress {
		if strings.Contains(lower, pattern) {
			r.log.Debug("NLU triggered", zap.String("reason", "direct_address"), zap.String("pattern", pattern), zap.String("chat", chat))
			return true
		}
	}

	return false
}

// isMentioned verifica se a mensagem menciona a Shinobu como palavra isolada
// (não como substring) ou via menção explícita @jid.
func isMentioned(msg, botJID string) bool {
	lower := strings.ToLower(msg)
	if wordMatch(lower, "shinobu") {
		return true
	}
	if botJID != "" && strings.Contains(msg, botJID) {
		return true
	}
	return false
}

// wordMatch verifica se word aparece como palavra isolada em s.
// Palavra isolada: precedida e seguida por espaço, início ou fim da string,
// ou por caractere não-alfabético (ex: pontuação).
func wordMatch(s, word string) bool {
	idx := strings.Index(s, word)
	if idx == -1 {
		return false
	}
	before := idx == 0 || !isLetter(rune(s[idx-1]))
	after := idx+len(word) >= len(s) || !isLetter(rune(s[idx+len(word)]))
	return before && after
}

func isLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}
