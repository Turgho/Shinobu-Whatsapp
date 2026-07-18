package public

import (
	"context"
	"fmt"

	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/sticker"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/media"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// StickerAlbumCommand processa múltiplos itens de um album (imagens/vídeos)
// e envia uma figurinha para cada. Chamado pelo AlbumCoordinator quando o
// comando !sticker é usado em um album.
func StickerAlbumCommand(ctx context.Context, client *whatsmeow.Client, items []*events.Message, args []string) error {
	_ = whatsapp.Reply(ctx, client, items[0], fmt.Sprintf("⏳ Processando %d figurinhas do album...", len(items)))

	var ok, fail int
	for _, item := range items {
		dl, err := media.DownloadFromEvent(ctx, client, item, media.FilterVisual)
		if err != nil {
			fail++
			continue
		}

		webp, err := sticker.ConvertToWebP(ctx, dl.Data, dl.Ext, dl.Animated)
		if err != nil {
			fail++
			continue
		}

		uploaded, err := client.Upload(ctx, webp, whatsmeow.MediaImage)
		if err != nil {
			fail++
			continue
		}

		if err := whatsapp.SendSticker(ctx, client, item, &uploaded, dl.Animated, false); err != nil {
			fail++
			continue
		}
		ok++
	}

	if fail > 0 && ok > 0 {
		_ = whatsapp.Reply(ctx, client, items[0],
			fmt.Sprintf("✅ %d figurinha(s) pronta(s). ⚠️ %d falharam.", ok, fail))
	} else if fail > 0 {
		_ = whatsapp.Reply(ctx, client, items[0],
			"Não consegui processar nenhuma mídia do album. Verifique se são imagens ou vídeos.")
	}

	return nil
}
