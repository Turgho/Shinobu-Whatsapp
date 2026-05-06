package sticker

import (
	"context"
	"fmt"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

// Send faz o upload do WebP para os servidores do WhatsApp e envia
// a mensagem de sticker citando a mensagem original.
func Send(ctx context.Context, client *whatsmeow.Client, evt *events.Message, webpData []byte, animated bool) error {
	uploaded, err := client.Upload(ctx, webpData, whatsmeow.MediaImage)
	if err != nil {
		return fmt.Errorf("sticker/send: falha no upload: %w", err)
	}

	msg := &waE2E.Message{
		StickerMessage: &waE2E.StickerMessage{
			URL:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			Mimetype:      proto.String("image/webp"),
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uploaded.FileLength),
			IsAnimated:    proto.Bool(animated),
			ContextInfo: &waE2E.ContextInfo{
				StanzaID:      proto.String(evt.Info.ID),
				Participant:   proto.String(evt.Info.Sender.String()),
				QuotedMessage: evt.Message,
			},
		},
	}

	if _, err = client.SendMessage(ctx, evt.Info.Chat, msg); err != nil {
		return fmt.Errorf("sticker/send: falha ao enviar: %w", err)
	}

	return nil
}
