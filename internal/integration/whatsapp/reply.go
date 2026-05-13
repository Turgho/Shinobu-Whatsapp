package whatsapp

import (
	"context"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// Reply envia texto citando a mensagem que disparou o handler.
func Reply(ctx context.Context, client *whatsmeow.Client, evt *events.Message, text string) error {
	return SendText(ctx, client, evt, text, true)
}

// ReplyWithMentions envia resposta citada com menções @.
func ReplyWithMentions(ctx context.Context, client *whatsmeow.Client, evt *events.Message, text string, mentions []string) error {
	return SendTextWithMentions(ctx, client, evt, text, mentions, true)
}
