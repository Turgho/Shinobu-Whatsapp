package public

import (
	"context"
	"os"
	"strings"

	"github.com/Turgho/YuukoWhatsapp/internal/commands"
	"github.com/Turgho/YuukoWhatsapp/internal/utils"
	"github.com/Turgho/YuukoWhatsapp/pkg/history"
	"github.com/Turgho/YuukoWhatsapp/pkg/ia"
	"github.com/Turgho/YuukoWhatsapp/pkg/sticker"
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
			return utils.Reply(ctx, client, evt, "Hmph... fala logo, tolo.")
		}

		// Menções
		mentions := []string{}
		if ext := evt.Message.GetExtendedTextMessage(); ext != nil {
			mentions = ext.GetContextInfo().GetMentionedJID()
		}

		_ = store.Save(ctx, chat, sender, prompt)

		answer, usedSearch, err := ia.AskIA(ctx, chat, prompt, isOwner, sender, store)
		if err != nil {
			return utils.Reply(ctx, client, evt, "❌ Falha ao consultar a Shinobu.")
		}

		_ = store.Save(ctx, chat, history.AssistantSenderName, answer)

		if len(mentions) > 0 {
			if err := utils.ReplyWithMentions(ctx, client, evt, answer, mentions); err != nil {
				return err
			}

			// Envia figurinha quando usar pesquisa web
			if usedSearch {
				_ = sticker.Send(ctx, client, evt, "smart_ruby", false)
			}

			return nil
		}

		if err := utils.Reply(ctx, client, evt, answer); err != nil {
			return err
		}

		// Envia figurinha quando usar pesquisa web
		if usedSearch {
			_ = sticker.Send(ctx, client, evt, "smart_ruby", false)
		}

		return nil
	}
}
