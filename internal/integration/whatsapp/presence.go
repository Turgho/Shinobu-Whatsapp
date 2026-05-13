package whatsapp

import (
	"context"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// withTyping mostra "a escrever..." antes da ação e repõe pausa no fim.
func withTyping(ctx context.Context, client *whatsmeow.Client, evt *events.Message, media types.ChatPresenceMedia, fn func() error) error {
	_ = client.SendChatPresence(ctx, evt.Info.Chat, types.ChatPresenceComposing, media)
	defer client.SendChatPresence(ctx, evt.Info.Chat, types.ChatPresencePaused, media)
	return fn()
}
