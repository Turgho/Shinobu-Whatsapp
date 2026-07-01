package ia

import (
	"context"
	"strings"
	"time"

	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/history"
)

// baseSystemMessages monta a lista inicial de mensagens do sistema:
// personalidade + (opcional) tratamento especial para o dono.
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

// appendPersistentAndRecent anexa ao slice de mensagens o conteúdo persistente
// do chat: fatos de memória (key-value), fatos atômicos (IA), resumo da conversa
// e transcript das últimas mensagens.
func appendPersistentAndRecent(ctx context.Context, messages []history.IAMessage, chat, sender string, store *history.Store, transcriptLimit int) []history.IAMessage {
	if store == nil || chat == "" {
		return messages
	}

	// Memória de usuário: fatos extraídos por padrões (key-value).
	if store.UserMemory != nil && sender != "" {
		if facts, err := store.UserMemory.GetFacts(ctx, chat, sender); err == nil && len(facts) > 0 {
			if formatted := history.FormatFacts(facts); formatted != "" {
				messages = append(messages, history.IAMessage{
					Role:    "system",
					Content: formatted,
				})
			}
		}
	}

	// Fatos atômicos extraídos por IA (user_facts).
	if afacts, err := store.GetFacts(ctx, chat, sender); err == nil && len(afacts) > 0 {
		if formatted := history.FormatAtomicFacts(afacts); formatted != "" {
			messages = append(messages, history.IAMessage{
				Role:    "system",
				Content: formatted,
			})
		}
	}

	if summary, err := store.GetSummary(ctx, chat); err == nil && strings.TrimSpace(summary) != "" {
		summary = truncateText(strings.TrimSpace(summary), 1200)
		messages = append(messages, history.IAMessage{
			Role:    "system",
			Content: "Resumo persistente da conversa anterior:\n" + summary,
		})
	}
	if tr, err := store.TranscriptRecent(ctx, chat, transcriptLimit, 12*time.Hour); err == nil && strings.TrimSpace(tr) != "" {
		tr = truncateText(tr, 2200)
		messages = append(messages, history.IAMessage{
			Role: "system",
			Content: "Últimas mensagens neste chat (antigas → recentes; use para continuidade, não copie de graça):\n" +
				tr,
		})
	}
	return messages
}
