package public

import (
	"context"
	"strings"

	"github.com/Turgho/Shinobu-Whatsapp/internal/commands"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/history"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/ia"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/sticker"
	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/configs"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

func ShinobuCommand(store *history.Store, cfg *configs.Config) commands.HandlerFunc {
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		chat := evt.Info.Chat.String()
		sender := evt.Info.Sender.User
		isOwner := sender == cfg.Owner.Number
		prompt := strings.Join(args, " ")

		prompt = strings.TrimSpace(
			strings.NewReplacer("shinobu", "", "Shinobu", "").Replace(prompt),
		)

		if len(args) == 0 || prompt == "" {
			return whatsapp.Reply(ctx, client, evt, "Hmph... fala logo, tolo.")
		}

		mentions := []string{}
		if ext := evt.Message.GetExtendedTextMessage(); ext != nil {
			mentions = ext.GetContextInfo().GetMentionedJID()
		}

		_ = store.Save(ctx, chat, sender, prompt)

		iaCfg := &ia.Config{
			GroqURL:   cfg.Groq.URL,
			GroqKey:   cfg.Groq.APIKey,
			TavilyKey: cfg.Tavily.APIKey,
		}

		answer, usedSearch, err := ia.AskIA(ctx, iaCfg, chat, prompt, isOwner, sender, store)
		if err != nil {
			return whatsapp.Reply(ctx, client, evt, "❌ Falha ao consultar a Shinobu.")
		}

		_ = store.Save(ctx, chat, history.AssistantSenderName, answer)

		if len(mentions) > 0 {
			if err := whatsapp.ReplyWithMentions(ctx, client, evt, answer, mentions); err != nil {
				return err
			}
			if usedSearch {
				_ = sticker.Send(ctx, client, evt, "smart_ruby", false)
			}
			return nil
		}

		if err := whatsapp.Reply(ctx, client, evt, answer); err != nil {
			return err
		}

		if usedSearch {
			_ = sticker.Send(ctx, client, evt, "smart_ruby", false)
		}

		return nil
	}
}
