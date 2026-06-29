package public

import (
	"context"

	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/sticker"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/media"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// StickerCommand converte uma imagem ou vídeo em figurinha.
//   - Envie uma imagem ou vídeo com a legenda "!sticker"
//   - Ou responda a uma mensagem de mídia com "!sticker"
func StickerCommand(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
	_ = whatsapp.Reply(ctx, client, evt, "⏳ Processando sua figurinha...")

	dl, err := media.DownloadFromEvent(ctx, client, evt, media.FilterVisual)
	if err != nil {
		return whatsapp.Reply(ctx, client, evt, msgNoMediaForSticker)
	}

	webp, err := sticker.ConvertToWebP(ctx, dl.Data, dl.Ext, dl.Animated)
	if err != nil {
		return whatsapp.Reply(ctx, client, evt, msgConvertStickerFail)
	}

	uploaded, err := client.Upload(ctx, webp, whatsmeow.MediaImage)
	if err != nil {
		return whatsapp.Reply(ctx, client, evt, msgSendStickerFail)
	}

	if err := whatsapp.SendSticker(ctx, client, evt, &uploaded, dl.Animated, true); err != nil {
		return whatsapp.Reply(ctx, client, evt, msgSendStickerFail)
	}

	return nil
}
