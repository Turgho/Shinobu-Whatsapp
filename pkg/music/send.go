package music

import (
	"context"
	"fmt"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

// SendAudio faz upload do áudio e envia como mensagem de áudio no WhatsApp.
// - data: bytes do áudio (mp3, opus, etc)
// - mimetype: ex "audio/mpeg"
// - ptt: true = áudio tipo "voz" (bolinha), false = áudio normal
func SendAudio(
	ctx context.Context,
	client *whatsmeow.Client,
	evt *events.Message,
	data []byte,
	mimetype string,
	ptt bool,
) error {

	// upload para servidores do WhatsApp
	uploaded, err := client.Upload(ctx, data, whatsmeow.MediaAudio)
	if err != nil {
		return fmt.Errorf("music/send: falha no upload: %w", err)
	}

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
			ContextInfo: &waE2E.ContextInfo{
				StanzaID:      proto.String(evt.Info.ID),
				Participant:   proto.String(evt.Info.Sender.String()),
				QuotedMessage: evt.Message,
			},
		},
	}

	if _, err = client.SendMessage(ctx, evt.Info.Chat, msg); err != nil {
		return fmt.Errorf("music/send: falha ao enviar: %w", err)
	}

	return nil
}
