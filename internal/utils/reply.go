package utils

import (
	"context"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// Reply sends a text message that quotes the triggering message.
func Reply(ctx context.Context, client *whatsmeow.Client, evt *events.Message, text string) error {
	return SendText(ctx, client, evt, text, true)
}

// ReplyWithMentions sends a quoted reply that also @-mentions the given JIDs.
func ReplyWithMentions(ctx context.Context, client *whatsmeow.Client, evt *events.Message, text string, mentions []string) error {
	return SendTextWithMentions(ctx, client, evt, text, mentions, true)
}
