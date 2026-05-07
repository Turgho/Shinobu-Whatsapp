package admin

import (
	"context"
	"os"

	"github.com/Turgho/YuukoWhatsapp/internal/utils"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

func ShinobuCommand(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
	// Lê o arquivo mp4 convertido.
	data, err := os.ReadFile("assets/videos/shinobu_media.mp4")
	if err != nil {
		return utils.Reply(ctx, client, evt,
			"❌ Não consegui carregar o GIF.")
	}

	// Upload para os servidores do WhatsApp.
	uploaded, err := client.Upload(ctx, data, whatsmeow.MediaVideo)
	if err != nil {
		return utils.Reply(ctx, client, evt,
			"❌ Não consegui enviar o GIF.")
	}

	// Envia como vídeo com playback de GIF.
	if err := utils.SendVideo(ctx, client, evt, &uploaded, "🦇 Shinobu.", true, 6); err != nil {
		return utils.Reply(ctx, client, evt,
			"❌ Falha ao enviar mensagem.")
	}

	return nil
}
