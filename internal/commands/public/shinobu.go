package public

import (
	"context"
	"os"
	"strings"

	"github.com/Turgho/Shinobu-Whatsapp/internal/commands"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/history"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/ia"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/sticker"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
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

		// Remove o nome do bot do prompt — evita ruído quando chamado por menção.
		prompt = strings.TrimSpace(
			strings.NewReplacer("shinobu", "", "Shinobu", "").Replace(prompt),
		)

		if len(args) == 0 || prompt == "" {
			return whatsapp.Reply(ctx, client, evt, "Hmph... fala logo, tolo.")
		}

		// Menções na mensagem original.
		var mentions []string
		if ext := evt.Message.GetExtendedTextMessage(); ext != nil {
			mentions = ext.GetContextInfo().GetMentionedJID()
		}

		_ = store.Save(ctx, chat, sender, prompt)

		// Usa AskAgent se o registry estiver configurado (tool calling ativo),
		// caso contrário cai no AskIA original.
		var answer string
		var usedSearch bool
		var err error

		if reg := ia.GetRegistry(); reg != nil {
			answer, err = ia.AskAgent(ctx, client, evt, reg, chat, prompt, isOwner, store)
		} else {
			answer, usedSearch, err = ia.AskIA(ctx, chat, prompt, isOwner, sender, store)
		}

		if err != nil {
			return whatsapp.Reply(ctx, client, evt, "❌ Falha ao consultar a Shinobu.")
		}

		_ = store.Save(ctx, chat, history.AssistantSenderName, answer)

		// Envia a resposta — com ou sem menções.
		if len(mentions) > 0 {
			if err := whatsapp.ReplyWithMentions(ctx, client, evt, answer, mentions); err != nil {
				return err
			}
		} else {
			if err := whatsapp.Reply(ctx, client, evt, answer); err != nil {
				return err
			}
		}

		// Figurinha de busca web — só no fluxo sem agent (AskIA).
		if usedSearch {
			_ = sticker.Send(ctx, client, evt, "smart_ruby", false)
		}

		return nil
	}
}
