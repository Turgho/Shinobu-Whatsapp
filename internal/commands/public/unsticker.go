package public

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Turgho/Shinobu-Whatsapp/internal/commands"
	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/ffmpeg"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/media"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

func UnstickerCommand(
	ctx context.Context,
	client *whatsmeow.Client,
	evt *events.Message,
	_ []string,
) error {
	dl, err := media.DownloadFromEvent(ctx, client, evt, media.FilterVisual)
	if err != nil {
		return whatsapp.Reply(ctx, client, evt, msgUnstickerNoMedia)
	}

	// Salva o WebP original em temp.
	inPath := filepath.Join(os.TempDir(), fmt.Sprintf("unsticker_input_%d.webp", evt.Info.Timestamp.UnixNano()))
	outPath := filepath.Join(os.TempDir(), fmt.Sprintf("unsticker_output_%d", evt.Info.Timestamp.UnixNano()))

	if err := os.WriteFile(inPath, dl.Data, 0644); err != nil {
		return whatsapp.Reply(ctx, client, evt, msgUnstickerFail)
	}
	defer os.Remove(inPath)

	isAnimated := dl.Animated

	if isAnimated {
		outPath += ".gif"
		cmd := ffmpeg.FfmpegCmd(ctx,
			"-i", inPath,
			"-vf", "fps=15,scale=512:-1",
			outPath,
		)
		if output, err := cmd.CombinedOutput(); err != nil {
			return whatsapp.Reply(ctx, client, evt, msgUnstickerFail)
		} else if len(output) > 0 {
			// Apenas loga em debug se FFmpeg emitiu warnings
		}
		defer os.Remove(outPath)

		data, err := os.ReadFile(outPath)
		if err != nil {
			return whatsapp.Reply(ctx, client, evt, msgUnstickerFail)
		}

		uploaded, err := client.Upload(ctx, data, whatsmeow.MediaVideo)
		if err != nil {
			return whatsapp.Reply(ctx, client, evt, msgUnstickerFail)
		}

		return whatsapp.SendVideo(ctx, client, evt, &uploaded, msgUnstickerCaption, true, 5, true)
	}

	outPath += ".png"
	cmd := ffmpeg.FfmpegCmd(ctx,
		"-i", inPath,
		outPath,
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		return whatsapp.Reply(ctx, client, evt, msgUnstickerFail)
	} else if len(output) > 0 {
	}
	defer os.Remove(outPath)

	data, err := os.ReadFile(outPath)
	if err != nil {
		return whatsapp.Reply(ctx, client, evt, msgUnstickerFail)
	}

	uploaded, err := client.Upload(ctx, data, whatsmeow.MediaImage)
	if err != nil {
		return whatsapp.Reply(ctx, client, evt, msgUnstickerFail)
	}

	return whatsapp.SendImage(ctx, client, evt, &uploaded, data, msgUnstickerCaption, true)
}

func UnstickerHandler() commands.HandlerFunc {
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		return UnstickerCommand(ctx, client, evt, args)
	}
}
