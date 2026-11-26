package public

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/Turgho/Shinobu-Whatsapp/internal/commands"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/sticker"
	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/ffmpeg"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/media"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// AudioStickerCommand cria uma figurinha com som a partir de um vídeo.
//   - Envie um vídeo (ou GIF) com a legenda "!ss"
//   - Ou responda a um vídeo com "!ss"
//
// O ffmpeg separa a trilha de áudio (OGG Opus, cortada em 7s — limite do
// WhatsApp), a figurinha é gerada normalmente e as duas mídias vão juntas
// como audio sticker. Arquivo sem som cai para figurinha comum, avisando.
func AudioStickerCommand(
	ctx context.Context,
	client *whatsmeow.Client,
	evt *events.Message,
	_ []string,
) error {
	_ = whatsapp.Reply(ctx, client, evt, msgAudioStickerWorking)

	dl, err := media.DownloadFromEvent(ctx, client, evt, media.Filter{Image: true, Video: true})
	if err != nil {
		return whatsapp.Reply(ctx, client, evt, msgAudioStickerNoMedia)
	}

	audioPath, seconds, audioErr := extractStickerAudio(ctx, dl)

	webp, err := sticker.ConvertToWebP(ctx, dl.Data, dl.Ext, dl.Animated)
	if err != nil {
		if audioErr == nil {
			os.Remove(audioPath)
		}
		return whatsapp.Reply(ctx, client, evt, msgConvertStickerFail)
	}

	if audioErr != nil {
		return fallbackToPlainSticker(ctx, client, evt, webp, dl.Animated, audioErr)
	}
	defer os.Remove(audioPath)

	imageUp, err := client.Upload(ctx, webp, whatsmeow.MediaImage)
	if err != nil {
		return whatsapp.Reply(ctx, client, evt, msgSendStickerFail)
	}

	soundData, err := os.ReadFile(audioPath)
	if err != nil {
		return whatsapp.Reply(ctx, client, evt, msgAudioStickerAudioFail)
	}

	soundUp, err := client.Upload(ctx, soundData, whatsmeow.MediaAudio)
	if err != nil {
		return whatsapp.Reply(ctx, client, evt, msgSendStickerFail)
	}

	if err := whatsapp.SendAudioSticker(ctx, client, evt, &imageUp, &soundUp, dl.Animated, uint32(seconds)); err != nil {
		return whatsapp.Reply(ctx, client, evt, msgSendStickerFail)
	}
	return nil
}

// extractStickerAudio grava a mídia baixada num temporário e extrai a trilha
// de áudio para OGG Opus. Devolve o caminho do arquivo e a duração em
// segundos; ErrNoAudioTrack quando a entrada não tem som.
func extractStickerAudio(ctx context.Context, dl *media.Media) (string, int, error) {
	tmp, err := os.CreateTemp("", "soundsticker-src-*"+dl.Ext)
	if err != nil {
		return "", 0, fmt.Errorf("sticker_audio: temporário: %w", err)
	}
	srcPath := tmp.Name()

	if _, err := tmp.Write(dl.Data); err != nil {
		tmp.Close()
		os.Remove(srcPath)
		return "", 0, fmt.Errorf("sticker_audio: gravar entrada: %w", err)
	}
	tmp.Close()

	// O ffmpeg lê e já escreve a saída em outro arquivo; o fonte não precisa
	// sobreviver.
	defer os.Remove(srcPath)

	return ffmpeg.ExtractOpus(ctx, srcPath)
}

// fallbackToPlainSticker envia a figurinha sem som quando o arquivo não tem
// trilha de áudio (ou a extração falhou) — não é erro, só um aviso.
func fallbackToPlainSticker(
	ctx context.Context,
	client *whatsmeow.Client,
	evt *events.Message,
	webp []byte,
	animated bool,
	audioErr error,
) error {
	uploaded, err := client.Upload(ctx, webp, whatsmeow.MediaImage)
	if err != nil {
		return whatsapp.Reply(ctx, client, evt, msgSendStickerFail)
	}
	if err := whatsapp.SendSticker(ctx, client, evt, &uploaded, animated, true); err != nil {
		return whatsapp.Reply(ctx, client, evt, msgSendStickerFail)
	}

	if errors.Is(audioErr, ffmpeg.ErrNoAudioTrack) {
		return whatsapp.Reply(ctx, client, evt, msgAudioStickerNoAudio)
	}
	return whatsapp.Reply(ctx, client, evt, msgAudioStickerAudioFail)
}

func AudioStickerHandler() commands.HandlerFunc {
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		return AudioStickerCommand(ctx, client, evt, args)
	}
}
