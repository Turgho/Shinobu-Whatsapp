package admin

import (
	"context"

	"github.com/Turgho/Shinobu-Whatsapp/internal/commands"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/sticker"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// StickerCommand gerencia stickers salvos.
//
// Em DM (apenas o dono):
//
//	!fig salvar <nome>  — salva o sticker enviado ou citado
//	!fig remover <nome> — remove um sticker salvo
//	!fig lista          — lista todos os stickers
//
// Em grupos ou DM:
//
//	!fig <nome>         — envia o sticker salvo
func SaveStickerCommand(ownerNumber string) commands.HandlerFunc {
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		// HandleDM já filtra por DM, dono e prefixo !fig salvar/remover/lista.
		sticker.HandleDM(ctx, client, evt, ownerNumber)

		// Enviar sticker salvo pelo nome.
		if len(args) == 0 {
			return whatsapp.SendText(ctx, client, evt,
				"❌ Uso:\n"+
					"!fig salvar <nome>\n"+
					"!fig remover <nome>\n"+
					"!fig lista\n"+
					"!fig <nome>",
				true,
			)
		}

		if err := sticker.Send(ctx, client, evt, args[0], true); err != nil {
			return whatsapp.SendText(ctx, client, evt, "❌ Sticker não encontrado ou falha ao enviar.", true)
		}

		return nil
	}
}
