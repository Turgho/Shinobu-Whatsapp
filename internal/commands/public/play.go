package public

import (
	"context"
	"strings"

	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/music"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// PlayCommand baixa áudio a partir do texto ou URL nos args e envia como mensagem de voz.
func PlayCommand(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
	if len(args) == 0 {
		return whatsapp.SendText(ctx, client, evt,
			"🎵 Informe o nome ou URL da música.\nExemplo: !play imagine dragons", true)
	}

	_ = whatsapp.SendText(ctx, client, evt, "⏳ Processando sua música...", true)

	query := strings.Join(args, " ")
	audio, ext, err := music.DownloadAudio(ctx, query)
	if err != nil {
		return whatsapp.SendText(ctx, client, evt, "❌ Não consegui baixar essa música.", true)
	}

	uploaded, err := client.Upload(ctx, audio, whatsmeow.MediaAudio)
	if err != nil {
		return playSendAudioErr(ctx, client, evt)
	}

	if err := whatsapp.SendAudio(ctx, client, evt, &uploaded, music.AudioMimetypeForExt(ext), false, true); err != nil {
		return playSendAudioErr(ctx, client, evt)
	}
	return nil
}

func playSendAudioErr(ctx context.Context, client *whatsmeow.Client, evt *events.Message) error {
	return whatsapp.SendText(ctx, client, evt, "❌ Não consegui enviar o áudio.", true)
}
