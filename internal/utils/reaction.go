package utils

import (
	"context"
	"fmt"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waCommon"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

// SendReaction reacts to the original message with an emoji.
// Pass an empty string to remove an existing reaction.
//
//	utils.SendReaction(ctx, client, evt, "👍")
//	utils.SendReaction(ctx, client, evt, "") // remove reaction
func SendReaction(
	ctx context.Context,
	client *whatsmeow.Client,
	evt *events.Message,
	emoji string,
) error {
	msg := &waE2E.Message{
		ReactionMessage: &waE2E.ReactionMessage{
			Key: &waCommon.MessageKey{
				RemoteJID:   proto.String(evt.Info.Chat.String()),
				FromMe:      proto.Bool(evt.Info.IsFromMe),
				ID:          proto.String(evt.Info.ID),
				Participant: proto.String(evt.Info.Sender.ToNonAD().String()),
			},
			Text:              proto.String(emoji),
			SenderTimestampMS: proto.Int64(time.Now().UnixMilli()),
		},
	}
	_, err := client.SendMessage(ctx, evt.Info.Chat, msg)
	if err != nil {
		return fmt.Errorf("utils/reaction: %w", err)
	}
	return nil
}
