package public

import (
	"context"

	"github.com/Turgho/YuukoWhatsapp/internal/utils"
	"github.com/Turgho/YuukoWhatsapp/pkg/sticker"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// FixedBundledAudioCommand cria um handler que envia um clip OGG (PTT + reply) e, se stickerName não for vazio, envia a figurinha a seguir.
// audioPath é relativo ao diretório de trabalho do processo (ex.: assets/audios/mambo.ogg).
func FixedBundledAudioCommand(audioPath, stickerName string) func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		if err := utils.SendBundledOggVoice(ctx, client, evt, audioPath); err != nil {
			return err
		}
		if stickerName != "" {
			_ = sticker.Send(ctx, client, evt, stickerName, false)
		}
		return nil
	}
}
