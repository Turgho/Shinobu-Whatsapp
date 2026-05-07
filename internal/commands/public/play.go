package public

import (
	"context"
	"strings"
	"time"

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

	// Processa em background para não estourar o timeout do handler.
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()

		// Junta os argumentos em uma única busca.
		query := strings.Join(args, " ")

		audio, ext, err := music.DownloadAudio(bgCtx, query)
		if err != nil {
			_ = utils.Reply(bgCtx, client, evt,
				"❌ Não consegui baixar essa música.")
			return
		}

		// Define o mimetype de acordo com a extensão.
		mimetype := "audio/mpeg"
		if ext == "ogg" || ext == "opus" {
			mimetype = "audio/ogg; codecs=opus"
		}

		if err := music.SendAudio(bgCtx, client, evt, audio, mimetype, false); err != nil {
			_ = utils.Reply(bgCtx, client, evt,
				"❌ Não consegui enviar o áudio.")
			return
		}
	}()

	return nil
}
