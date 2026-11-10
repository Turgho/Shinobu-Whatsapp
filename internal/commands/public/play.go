package public

import (
	"context"
	"strings"

	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/music"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
	"go.uber.org/zap"
)

// PlayCommand baixa a música via music.Service e envia como áudio.
func PlayCommand(svc *music.Service, logger *zap.Logger) func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
	l := logger.Named("PLAY")
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		if len(args) == 0 {
			return whatsapp.SendText(ctx, client, evt, msgNoQuery, true)
		}

		_ = whatsapp.SendText(ctx, client, evt, "⏳ Processando sua música...", true)

		query := strings.Join(args, " ")
		audio, ext, err := svc.DownloadAudio(ctx, query)
		if err != nil {
			l.Warn("play: download falhou",
				zap.String("query", query), zap.Error(err))
			// Erros classificados recebem mensagem específica em PT.
			if ed, ok := music.AsErrDownload(err); ok {
				return whatsapp.SendText(ctx, client, evt, "❌ "+ed.Category.CategoryText(), true)
			}
			return whatsapp.SendText(ctx, client, evt, msgDownloadFail, true)
		}

		audio, err = music.EmbedMetadata(audio, ext, query)
		if err != nil {
			// metadata é opcional; continua com o áudio original
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
}

func playSendAudioErr(ctx context.Context, client *whatsmeow.Client, evt *events.Message) error {
	return whatsapp.SendText(ctx, client, evt, msgSendAudioFail, true)
}
