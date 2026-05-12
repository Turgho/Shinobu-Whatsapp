package public

import (
	"context"

	"github.com/Turgho/YuukoWhatsapp/internal/utils"
	"github.com/Turgho/YuukoWhatsapp/pkg/sticker"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// StickerCommand converte uma imagem ou vídeo em figurinha.
//   - Envie uma imagem ou vídeo com a legenda "!sticker"
//   - Ou responda a uma mensagem de mídia com "!sticker"
func StickerCommand(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
	_ = utils.Reply(ctx, client, evt, "⏳ Processando sua figurinha...")

	media, err := utils.DownloadFromEvent(ctx, client, evt, utils.FilterVisual)
	if err != nil {
		return utils.Reply(ctx, client, evt,
			"📎 Envie uma imagem ou vídeo com a legenda `!sticker`, ou responda a uma mídia com `!sticker`.")
	}

	webp, err := sticker.ConvertToWebP(ctx, media.Data, media.Ext, media.Animated)
	if err != nil {
		return utils.Reply(ctx, client, evt,
			"❌ Não consegui converter a mídia. Certifique-se de que é uma imagem ou vídeo válido.")
	}

	uploaded, err := client.Upload(ctx, webp, whatsmeow.MediaImage)
	if err != nil {
		return utils.Reply(ctx, client, evt,
			"❌ Falha ao enviar a figurinha.")
	}

	if err := utils.SendSticker(ctx, client, evt, &uploaded, media.Animated, true); err != nil {
		return utils.Reply(ctx, client, evt,
			"❌ Falha ao enviar a figurinha. Tente novamente.")
	}

	return nil
}
