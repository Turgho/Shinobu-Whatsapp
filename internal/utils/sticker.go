package utils

import (
	"context"
	"fmt"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

// SendSticker sends a static or animated WebP sticker.
//
//   - animated: true = animated sticker (.webp with animation)
//   - reply:    true = quotes the original message
func SendSticker(
	ctx context.Context,
	client *whatsmeow.Client,
	evt *events.Message,
	uploaded *whatsmeow.UploadResponse,
	animated bool,
	reply bool,
) error {
	return withTyping(ctx, client, evt, types.ChatPresenceMediaText, func() error {
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
				ContextInfo:   buildContext(evt, reply, nil),
			},
		}
		_, err := client.SendMessage(ctx, evt.Info.Chat, msg)
		if err != nil {
			return fmt.Errorf("utils/sticker: %w", err)
		}
		return nil
	})
}
