package admin

import (
	"context"
	"fmt"
	"strings"

	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/sticker"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

func TestJobCommand(stickerStore *sticker.Store) func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		audioPath := "assets/audios/play_tv.ogg"
		stickerName := "play_tv"
		if len(args) > 0 {
			audioPath = args[0]
		}
		if len(args) > 1 {
			stickerName = args[1]
		}

		chat := evt.Info.Chat
		var errs []string

		if err := whatsapp.SendAudioFileToJID(ctx, client, chat, audioPath); err != nil {
			errs = append(errs, fmt.Sprintf("audio: %v", err))
		}

		if err := whatsapp.SendAllToJID(ctx, client, chat, "🎉 Teste automático sextou! @all"); err != nil {
			errs = append(errs, fmt.Sprintf("@all: %v", err))
		}

		if d, ok := stickerStore.Get(stickerName); ok {
			uploaded := &whatsmeow.UploadResponse{
				URL:           d.URL,
				DirectPath:    d.DirectPath,
				MediaKey:      d.MediaKey,
				FileEncSHA256: d.FileEncSHA256,
				FileSHA256:    d.FileSHA256,
				FileLength:    d.FileLength,
			}
			if err := whatsapp.SendStickerToJID(ctx, client, chat, uploaded, d.IsAnimated); err != nil {
				errs = append(errs, fmt.Sprintf("sticker: %v", err))
			}
		} else if stickerName != "" {
			errs = append(errs, fmt.Sprintf("sticker %q nao encontrado", stickerName))
		}

		if len(errs) > 0 {
			return whatsapp.SendText(ctx, client, evt, "❌ Erros:\n"+strings.Join(errs, "\n"), true)
		}
		return whatsapp.SendText(ctx, client, evt, "✅ Teste enviado com sucesso!", true)
	}
}
