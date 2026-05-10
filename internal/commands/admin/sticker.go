package admin

import (
	"context"

	"github.com/Turgho/YuukoWhatsapp/internal/commands"
	"github.com/Turgho/YuukoWhatsapp/internal/utils"
	"github.com/Turgho/YuukoWhatsapp/pkg/sticker"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// StickerCommand envia um sticker salvo pelo nome.
// Uso: !sticker <nome>
func StickerCommand() commands.HandlerFunc {
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		if len(args) == 0 {
			return utils.Reply(ctx, client, evt, "Hmph... qual sticker, tolo? Use: !sticker <nome>")
		}

		name := args[0]

		if err := sticker.Send(ctx, client, evt, name); err != nil {
			return utils.Reply(ctx, client, evt, "❌ Sticker não encontrado ou falha ao enviar.")
		}

		return nil
	}
}
