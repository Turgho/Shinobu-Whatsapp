package ia

import (
	"strings"

	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/history"
)

// formatMessagesForSummary converte um slice de IAMessage em texto legível
// com rótulos "Usuário:", "Assistente:", "Sistema:".
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
