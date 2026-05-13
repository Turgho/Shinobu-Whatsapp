package public

import (
	"context"
	"os"
	"strings"

	"github.com/Turgho/Shinobu-Whatsapp/internal/commands"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/history"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/ia"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/sticker"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

func ShinobuCommand(store *history.Store) commands.HandlerFunc {
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		ownerNumber := os.Getenv("OWNER_NUMBER")

		chat := evt.Info.Chat.String()
		sender := evt.Info.Sender.User
		isOwner := sender == ownerNumber
		prompt := strings.Join(args, " ")

		// Remove o nome do bot do prompt — evita ruído quando chamado por menção
		prompt = strings.TrimSpace(
			strings.NewReplacer("shinobu", "", "Shinobu", "").Replace(prompt),
		)

		if len(args) == 0 || prompt == "" {
			return whatsapp.Reply(ctx, client, evt, "Hmph... fala logo, tolo.")
		}

		// Menções
		mentions := []string{}
		if ext := evt.Message.GetExtendedTextMessage(); ext != nil {
			mentions = ext.GetContextInfo().GetMentionedJID()
		}

		_ = store.Save(ctx, chat, sender, prompt)

		answer, usedSearch, err := ia.AskIA(ctx, chat, prompt, isOwner, sender, store)
		if err != nil {
			return whatsapp.Reply(ctx, client, evt, "❌ Falha ao consultar a Shinobu.")
		}

		_ = store.Save(ctx, chat, history.AssistantSenderName, answer)

		if len(mentions) > 0 {
			if err := whatsapp.ReplyWithMentions(ctx, client, evt, answer, mentions); err != nil {
				return err
			}

			// Envia figurinha quando usar pesquisa web
			if usedSearch {
				_ = sticker.Send(ctx, client, evt, "smart_ruby", false)
			}

			return nil
		}

		if err := whatsapp.Reply(ctx, client, evt, answer); err != nil {
			return err
		}

		// Envia figurinha quando usar pesquisa web
		if usedSearch {
			_ = sticker.Send(ctx, client, evt, "smart_ruby", false)
		}

		return nil
	}
}
