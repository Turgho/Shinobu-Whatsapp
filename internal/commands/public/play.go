package public

import (
	"context"
	"strings"

	"github.com/Turgho/YuukoWhatsapp/internal/utils"
	"github.com/Turgho/YuukoWhatsapp/pkg/music"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// PlayCommand baixa áudio a partir do texto ou URL nos args e envia como mensagem de voz.
func PlayCommand(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
	if len(args) == 0 {
		return utils.SendText(ctx, client, evt,
			"🎵 Informe o nome ou URL da música.\nExemplo: !play imagine dragons", true)
	}

	_ = utils.SendText(ctx, client, evt, "⏳ Processando sua música...", true)

	query := strings.Join(args, " ")
	audio, ext, err := music.DownloadAudio(ctx, query)
	if err != nil {
		return utils.SendText(ctx, client, evt, "❌ Não consegui baixar essa música.", true)
	}

	uploaded, err := client.Upload(ctx, audio, whatsmeow.MediaAudio)
	if err != nil {
		return playSendAudioErr(ctx, client, evt)
	}

	if err := utils.SendAudio(ctx, client, evt, &uploaded, audioMimetype(ext), false, true); err != nil {
		return playSendAudioErr(ctx, client, evt)
	}
	return nil
}

// audioMimetype mapeia extensão retornada pelo downloader para o MIME esperado pelo WhatsApp.
func audioMimetype(ext string) string {
	if ext == "ogg" || ext == "opus" {
		return "audio/ogg; codecs=opus"
	}
	return "audio/mpeg"
}

func playSendAudioErr(ctx context.Context, client *whatsmeow.Client, evt *events.Message) error {
	return utils.SendText(ctx, client, evt, "❌ Não consegui enviar o áudio.", true)
}
