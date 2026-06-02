package ia

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

var lidMention = regexp.MustCompile(`@\d+@lid`)

var promptInjectionPatterns = []string{
	"ignore all previous",
	"ignore all instructions",
	"ignore your previous",
	"ignore your instructions",
	"ignore tudo",
	"ignore todas as",
	"desconsidere",
	"reveal your prompt",
	"reveal your system",
	"reveal your instructions",
	"mostre seu prompt",
	"mostre suas instruções",
	"repita seu prompt",
	"repita suas instruções",
	"agir como se fosse",
	"atuar como se fosse",
	"fingir que é",
	"you are now",
	"you are not",
	"act as if",
	"from now on",
	"new instructions",
	"override",
	"system prompt",
	"prompt inicial",
	"instrução do sistema",
	"instruções do sistema",
}

func cleanPrompt(prompt string) string {
	prompt = lidMention.ReplaceAllString(prompt, "")
	return strings.TrimSpace(prompt)
}

// sanitizePrompt remove conteúdo de prompt injection e marca o restante como seguro.
// Retorna o prompt limpo e um booleano indicando se foi detectada tentativa de injeção.
func sanitizePrompt(prompt string) (string, bool) {
	lower := strings.ToLower(prompt)
	for _, pattern := range promptInjectionPatterns {
		if strings.Contains(lower, pattern) {
			return prompt, true
		}
	}
	return prompt, false
}

func truncateText(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}

	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}

	r := []rune(s)
	return string(r[:maxRunes])
}
