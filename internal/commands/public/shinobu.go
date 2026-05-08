package public

import (
	"context"
	"os"
	"strings"

	"github.com/Turgho/YuukoWhatsapp/internal/commands"
	"github.com/Turgho/YuukoWhatsapp/internal/utils"
	"github.com/Turgho/YuukoWhatsapp/pkg/history"
	"github.com/Turgho/YuukoWhatsapp/pkg/ia"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

func ShinobuCommand(store *history.Store) commands.HandlerFunc {
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		ownerNumber := os.Getenv("OWNER_NUMBER")

		if len(args) == 0 {
			return utils.Reply(ctx, client, evt, "Hmph... fala logo, tolo.")
		}

		sender := evt.Info.Sender.User
		isOwner := sender == ownerNumber
		prompt := strings.Join(args, " ")

		// Menções
		mentions := []string{}
		if ext := evt.Message.GetExtendedTextMessage(); ext != nil {
			mentions = ext.GetContextInfo().GetMentionedJID()
		}

		store.Save(ctx, evt.Info.Chat.String(), sender, prompt)

		answer, err := ia.AskIA(ctx, prompt, isOwner, sender, store)
		if err != nil {
			return utils.Reply(ctx, client, evt, "❌ Falha ao consultar a Shinobu.")
		}

		store.Save(ctx, evt.Info.Chat.String(), "Shinobu", answer)

		if len(mentions) > 0 {
			return utils.ReplyWithMentions(ctx, client, evt, answer, mentions)
		}

		return utils.Reply(ctx, client, evt, answer)
	}
}
