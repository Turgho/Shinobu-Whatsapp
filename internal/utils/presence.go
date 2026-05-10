package utils

import (
	"context"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// withTyping sends the "typing..." indicator before the action runs,
// then clears it with a deferred pause. Used by all send functions.
func withTyping(ctx context.Context, client *whatsmeow.Client, evt *events.Message, media types.ChatPresenceMedia, fn func() error) error {
	_ = client.SendChatPresence(ctx, evt.Info.Chat, types.ChatPresenceComposing, media)
	defer client.SendChatPresence(ctx, evt.Info.Chat, types.ChatPresencePaused, media)
	return fn()
}
