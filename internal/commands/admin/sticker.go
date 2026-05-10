package admin

import (
	"context"
	"strings"

	"github.com/Turgho/YuukoWhatsapp/internal/commands"
	"github.com/Turgho/YuukoWhatsapp/internal/utils"
	"github.com/Turgho/YuukoWhatsapp/pkg/sticker"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// StickerCommand gerencia stickers salvos.
//
// Uso:
// !sticker salvar <nome>
// !sticker remover <nome>
// !sticker lista
// !sticker <nome>
func SaveStickerCommand(ownerNumber string) commands.HandlerFunc {
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {

		// Gerenciamento via DM
		if !evt.Info.IsGroup {
			msg := ""

			if evt.Message.GetConversation() != "" {
				msg = evt.Message.GetConversation()
			} else if evt.Message.GetExtendedTextMessage() != nil {
				msg = evt.Message.GetExtendedTextMessage().GetText()
			}

			msg = strings.TrimSpace(msg)

			if strings.HasPrefix(strings.ToLower(msg), "!sticker salvar") ||
				strings.HasPrefix(strings.ToLower(msg), "!sticker remover") ||
				strings.HasPrefix(strings.ToLower(msg), "!sticker lista") {

				sticker.HandleStickerDM(client, evt, ownerNumber)
				return nil
			}
		}

		// Enviar sticker salvo
		if len(args) == 0 {
			return utils.Reply(ctx, client, evt,
				"❌ Uso:\n"+
					"!sticker salvar <nome>\n"+
					"!sticker remover <nome>\n"+
					"!sticker lista\n"+
					"!sticker <nome>")
		}

		if err := sticker.Send(ctx, client, evt, args[0]); err != nil {
			return utils.Reply(ctx, client, evt,
				"❌ Sticker não encontrado ou falha ao enviar.")
		}

		return nil
	}
}
