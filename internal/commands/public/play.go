package public

import (
	"context"
	"strings"

	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/music"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

func PlayCommand(musicCfg *music.Config) func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		if len(args) == 0 {
			return whatsapp.SendText(ctx, client, evt,
				"🎵 Informe o nome ou URL da música.\nExemplo: !play imagine dragons", true)
		}

		_ = whatsapp.SendText(ctx, client, evt, "⏳ Processando sua música...", true)

		query := strings.Join(args, " ")
		audio, ext, err := music.DownloadAudio(ctx, musicCfg, query)
		if err != nil {
			return whatsapp.SendText(ctx, client, evt, "❌ Não consegui baixar essa música.", true)
		}

		fileName := query + "." + ext
		mime := music.AudioMimetypeForExt(ext)
		coverArt := music.ExtractCoverArtFromBytes(audio, ext)

		if err := whatsapp.SendAudioAsDocument(ctx, client, evt, audio, mime, fileName, coverArt); err != nil {
			return playSendAudioErr(ctx, client, evt)
		}
		return nil
	}
}

func playSendAudioErr(ctx context.Context, client *whatsmeow.Client, evt *events.Message) error {
	return whatsapp.SendText(ctx, client, evt, "❌ Não consegui enviar o áudio.", true)
}
