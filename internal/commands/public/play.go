package public

import (
	"context"
	"strings"

	"github.com/Turgho/YuukoWhatsapp/internal/utils"
	"github.com/Turgho/YuukoWhatsapp/pkg/music"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

func PlayCommand(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
	// Valida antes de iniciar processamento pesado.
	if len(args) == 0 {
		return utils.Reply(ctx, client, evt,
			"🎵 Informe o nome ou URL da música.\nExemplo: !play imagine dragons")
	}

	// Resposta imediata para o usuário.
	_ = utils.Reply(ctx, client, evt, "⏳ Processando sua música...")

	// Junta os argumentos em uma única busca.
	query := strings.Join(args, " ")

	audio, ext, err := music.DownloadAudio(ctx, query)
	if err != nil {
		return utils.Reply(ctx, client, evt,
			"❌ Não consegui baixar essa música.")
	}

	// Define o mimetype de acordo com a extensão.
	mimetype := "audio/mpeg"
	if ext == "ogg" || ext == "opus" {
		mimetype = "audio/ogg; codecs=opus"
	}

	uploaded, err := client.Upload(ctx, audio, whatsmeow.MediaAudio)
	if err != nil {
		return utils.Reply(ctx, client, evt,
			"❌ Não consegui enviar o áudio.")
	}

	if err := utils.SendAudio(
		ctx,
		client,
		evt,
		&uploaded,
		mimetype,
		false,
		false,
	); err != nil {
		return utils.Reply(ctx, client, evt,
			"❌ Não consegui enviar o áudio.")
	}

	return nil
}
