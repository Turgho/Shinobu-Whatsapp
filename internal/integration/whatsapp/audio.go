package whatsapp

import (
	"context"
	"fmt"
	"os"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

const mimetypeOggOpus = "audio/ogg; codecs=opus"

// SendBundledOggVoice lê um OGG Opus local, faz upload e envia como nota de voz (PTT) citando a mensagem.
func SendBundledOggVoice(ctx context.Context, client *whatsmeow.Client, evt *events.Message, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return Reply(ctx, client, evt, "❌ Não consegui carregar o áudio.")
	}
	uploaded, err := client.Upload(ctx, data, whatsmeow.MediaAudio)
	if err != nil {
		return Reply(ctx, client, evt, "❌ Não consegui preparar o áudio.")
	}
	if err := SendAudio(ctx, client, evt, &uploaded, mimetypeOggOpus, true, true); err != nil {
		return Reply(ctx, client, evt, "❌ Não consegui enviar o áudio.")
	}
	return nil
}

// SendAudio envia áudio já uploaded; ptt=true estilo nota de voz.
func SendAudio(
	ctx context.Context,
	client *whatsmeow.Client,
	evt *events.Message,
	uploaded *whatsmeow.UploadResponse,
	mimetype string,
	ptt bool,
	reply bool,
) error {
	return withTyping(ctx, client, evt, types.ChatPresenceMediaAudio, func() error {
		msg := &waE2E.Message{
			AudioMessage: &waE2E.AudioMessage{
				URL:           proto.String(uploaded.URL),
				DirectPath:    proto.String(uploaded.DirectPath),
				MediaKey:      uploaded.MediaKey,
				Mimetype:      proto.String(mimetype),
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    proto.Uint64(uploaded.FileLength),
				PTT:           proto.Bool(ptt),
				ContextInfo:   buildContext(evt, reply, nil),
			},
		}
		_, err := client.SendMessage(ctx, evt.Info.Chat, msg)
		if err != nil {
			return fmt.Errorf("whatsapp: send audio: %w", err)
		}
		return nil
	})
}
