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

// SendAudio sends an audio file (music, voice note, etc).
//
//   - mimetype: "audio/mpeg", "audio/ogg; codecs=opus", etc
//   - ptt:      true = voice note style (bubble), false = regular audio player
//   - reply:    true = quotes the original message
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
			return fmt.Errorf("utils/audio: %w", err)
		}
		return nil
	})
}
