package public

import (
	"context"
	"os"

	"github.com/Turgho/YuukoWhatsapp/internal/utils"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

func MamboAudioCommand(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
	// Lê o arquivo .mp3
	data, err := os.ReadFile("assets/audios/mambo.ogg")
	if err != nil {
		return utils.Reply(ctx, client, evt,
			"❌ Não consegui carregar o Audio.")
	}

	// Define o mimetype de acordo com a extensão.
	mimetype := "audio/ogg; codecs=opus"

	// Upload para os servidores do WhatsApp.
	uploaded, err := client.Upload(ctx, data, whatsmeow.MediaAudio)
	if err != nil {
		return utils.Reply(ctx, client, evt,
			"❌ Não consegui enviar o Audio.")
	}

	// Envia como audio de voice note
	if err := utils.SendAudio(
		ctx,
		client,
		evt,
		&uploaded,
		mimetype,
		true,
		true,
	); err != nil {
		return utils.Reply(ctx, client, evt,
			"❌ Não consegui enviar o áudio.")
	}

	return nil
}
