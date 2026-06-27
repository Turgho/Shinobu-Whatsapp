package admin

import (
	"context"
	"fmt"
	"strings"

	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/history"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
	"go.uber.org/zap"
)

func MemoriaCommand(
	ctx context.Context,
	client *whatsmeow.Client,
	evt *events.Message,
	args []string,
	store *history.Store,
	logger *zap.Logger,
) error {
	chat := evt.Info.Chat.String()

	sub := ""
	if len(args) > 0 {
		sub = strings.ToLower(strings.TrimSpace(args[0]))
	}

	switch sub {
	case "limpar":
		if len(args) > 1 {
			mentioned := extractMentionedJID(evt)
			if mentioned == "" {
				return whatsapp.Reply(ctx, client, evt, "❌ Mencione o usuário ou use `!memoria limpar` sem menção para apagar tudo.")
			}
			if err := store.ClearFacts(ctx, chat, mentioned); err != nil {
				logger.Error("Erro ao limpar fatos do usuário", zap.Error(err))
				return whatsapp.Reply(ctx, client, evt, "❌ Erro ao limpar fatos.")
			}
			return whatsapp.Reply(ctx, client, evt,
				fmt.Sprintf("🗑️ Fatos de @%s apagados.", mentioned))
		}

		if err := store.SetSummary(ctx, chat, ""); err != nil {
			logger.Error("Erro ao limpar resumo", zap.Error(err))
		}
		allFacts, err := store.GetAllFacts(ctx, chat)
		if err == nil {
			seen := make(map[string]struct{})
			for _, f := range allFacts {
				if _, ok := seen[f.User]; !ok {
					seen[f.User] = struct{}{}
					_ = store.ClearFacts(ctx, chat, f.User)
				}
			}
		}
		return whatsapp.Reply(ctx, client, evt, "🗑️ Memória do chat apagada.")

	case "resumo":
		summary, err := store.GetSummary(ctx, chat)
		if err != nil || strings.TrimSpace(summary) == "" {
			return whatsapp.Reply(ctx, client, evt, "📋 Nenhum resumo ainda.")
		}
		return whatsapp.Reply(ctx, client, evt,
			fmt.Sprintf("📋 *Resumo do chat:*\n%s", summary))

	default:
		summary, _ := store.GetSummary(ctx, chat)
		allFacts, _ := store.GetAllFacts(ctx, chat)

		var b strings.Builder
		b.WriteString("🧠 *Memória do chat*\n\n")

		b.WriteString("📋 *Resumo:*\n")
		if strings.TrimSpace(summary) == "" {
			b.WriteString("nenhum resumo ainda\n")
		} else {
			b.WriteString(summary)
			b.WriteString("\n")
		}

		if len(allFacts) > 0 {
			b.WriteString("\n👤 *Fatos por usuário:*\n")
			byUser := make(map[string][]string)
			for _, f := range allFacts {
				byUser[f.User] = append(byUser[f.User], f.Fact)
			}
			for user, facts := range byUser {
				b.WriteString(fmt.Sprintf("@%s: %s\n", user, strings.Join(facts, ", ")))
			}
		}

		return whatsapp.Reply(ctx, client, evt, b.String())
	}
}

func MemoriaHandler(store *history.Store, logger *zap.Logger) func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
	l := logger.Named("MEMORIA")
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		return MemoriaCommand(ctx, client, evt, args, store, l)
	}
}

func extractMentionedJID(evt *events.Message) string {
	ext := evt.Message.GetExtendedTextMessage()
	if ext == nil {
		return ""
	}
	mentions := ext.GetContextInfo().GetMentionedJID()
	if len(mentions) == 0 {
		return ""
	}
	jid := mentions[0]
	// strip @s.whatsapp.net suffix, keep just the number part
	if idx := strings.Index(jid, "@"); idx > 0 {
		jid = jid[:idx]
	}
	return jid
}
