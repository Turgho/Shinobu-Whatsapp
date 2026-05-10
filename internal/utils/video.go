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

// SendVideo sends a video with an optional caption.
//
//   - gifPlayback: true = plays as a silent looping GIF
//   - seconds:     duration of the video in seconds
//   - reply:       true = quotes the original message
func SendVideo(
	ctx context.Context,
	client *whatsmeow.Client,
	evt *events.Message,
	uploaded *whatsmeow.UploadResponse,
	caption string,
	gifPlayback bool,
	seconds uint32,
	reply bool,
) error {
	return withTyping(ctx, client, evt, types.ChatPresenceMediaText, func() error {
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
				ContextInfo:   buildContext(evt, reply, nil),
			},
		}
		_, err := client.SendMessage(ctx, evt.Info.Chat, msg)
		if err != nil {
			return fmt.Errorf("utils/video: %w", err)
		}
		return nil
	})
}
