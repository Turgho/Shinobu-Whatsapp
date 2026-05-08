package utils

import (
	"context"
	"fmt"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

// SendImage envia uma imagem já processada/enviada ao WhatsApp.
func SendImage(
	ctx context.Context,
	client *whatsmeow.Client,
	evt *events.Message,
	uploaded *whatsmeow.UploadResponse,
	caption string,
) error {
	msg := &waE2E.Message{
		ImageMessage: &waE2E.ImageMessage{
			URL:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			Mimetype:      proto.String("image/jpeg"),
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uploaded.FileLength),
			Caption:       proto.String(caption),
			ContextInfo:   quotedContext(evt, nil),
		},
	}

	if _, err := client.SendMessage(ctx, evt.Info.Chat, msg); err != nil {
		return fmt.Errorf("utils/send image: falha ao enviar: %w", err)
	}

	return nil
}

// SendVideo envia um vídeo normal ou como GIF, dependendo de gifPlayback.
func SendVideo(
	ctx context.Context,
	client *whatsmeow.Client,
	evt *events.Message,
	uploaded *whatsmeow.UploadResponse,
	caption string,
	gifPlayback bool,
	seconds uint32,
) error {
	msg := &waE2E.Message{
		VideoMessage: &waE2E.VideoMessage{
			URL:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			Mimetype:      proto.String("video/mp4"),
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uploaded.FileLength),
			GifPlayback:   proto.Bool(gifPlayback),
			Seconds:       proto.Uint32(seconds),
			Caption:       proto.String(caption),
			ContextInfo:   quotedContext(evt, nil),
		},
	}

	if _, err := client.SendMessage(ctx, evt.Info.Chat, msg); err != nil {
		return fmt.Errorf("utils/send video: falha ao enviar: %w", err)
	}

	return nil
}

// SendAudio faz upload do áudio e envia como mensagem de áudio no WhatsApp.
// - data: bytes do áudio (mp3, opus, etc)
// - mimetype: ex "audio/mpeg"
// - ptt: true = áudio tipo "voz" (bolinha), false = áudio normal
func SendAudio(
	ctx context.Context,
	client *whatsmeow.Client,
	evt *events.Message,
	uploaded *whatsmeow.UploadResponse,
	mimetype string,
	ptt bool,
) error {
	msg := &waE2E.Message{
		AudioMessage: &waE2E.AudioMessage{
			URL:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			Mimetype:      proto.String(mimetype),
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uploaded.FileLength),
			PTT:           proto.Bool(ptt), // true = estilo voice note
			ContextInfo:   quotedContext(evt, nil),
		},
	}

	if _, err := client.SendMessage(ctx, evt.Info.Chat, msg); err != nil {
		return fmt.Errorf("utils/send audio: falha ao enviar: %w", err)
	}

	return nil
}

// Send faz o upload do WebP para os servidores do WhatsApp e envia
// a mensagem de sticker citando a mensagem original.
func SendSticker(
	ctx context.Context,
	client *whatsmeow.Client,
	evt *events.Message,
	uploaded *whatsmeow.UploadResponse,
	animated bool,
) error {
	msg := &waE2E.Message{
		StickerMessage: &waE2E.StickerMessage{
			URL:                proto.String(uploaded.URL),
			DirectPath:         proto.String(uploaded.DirectPath),
			MediaKey:           uploaded.MediaKey,
			Mimetype:           proto.String("image/webp"),
			FileEncSHA256:      uploaded.FileEncSHA256,
			FileSHA256:         uploaded.FileSHA256,
			FileLength:         proto.Uint64(uploaded.FileLength),
			IsAnimated:         proto.Bool(animated),
			AccessibilityLabel: proto.String("Shinobu"),
			ContextInfo:        quotedContext(evt, nil),
		},
	}

	if _, err := client.SendMessage(ctx, evt.Info.Chat, msg); err != nil {
		return fmt.Errorf("utils/send sticker: falha ao enviar: %w", err)
	}
	return nil
}
