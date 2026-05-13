package public

import (
	"context"

	"github.com/Turgho/YuukoWhatsapp/internal/integration/whatsapp"
	"github.com/Turgho/YuukoWhatsapp/internal/domain/sticker"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// FixedBundledAudioCommand cria um handler que envia um clip OGG (PTT + reply) e, se stickerName não for vazio, envia a figurinha a seguir.
// audioPath é relativo ao diretório de trabalho do processo (ex.: assets/audios/mambo.ogg).
func FixedBundledAudioCommand(audioPath, stickerName string) func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		if err := whatsapp.SendBundledOggVoice(ctx, client, evt, audioPath); err != nil {
			return err
		}
		if stickerName != "" {
			_ = sticker.Send(ctx, client, evt, stickerName, false)
		}
		return nil
	}
}
