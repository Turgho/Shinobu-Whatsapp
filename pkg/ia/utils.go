package ia

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// cleanPrompt remove menções do WhatsApp (@número@lid) e espaços extras do prompt
func cleanPrompt(prompt string) string {
	re := regexp.MustCompile(`@\d+@lid`)
	return strings.TrimSpace(re.ReplaceAllString(prompt, ""))
}

// truncateText limita uma string pelo número de runes, sem quebrar UTF-8 no meio.
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
