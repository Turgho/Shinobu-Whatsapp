package utils

import (
	"context"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"

	"google.golang.org/protobuf/proto"
)

// Reply envia uma resposta de texto citando a mensagem original.
// Recebe ctx para respeitar cancelamento e timeouts do chamador.
func Reply(ctx context.Context, client *whatsmeow.Client, evt *events.Message, text string) error {
	// Mostra "Digitando..." enquanto processa
	if err := client.SendChatPresence(ctx, evt.Info.Chat, types.ChatPresenceComposing, types.ChatPresenceMediaText); err != nil {
		return err
	}
	defer client.SendChatPresence(ctx, evt.Info.Chat, types.ChatPresencePaused, types.ChatPresenceMediaText)

	msg := &waE2E.Message{
		ExtendedTextMessage: &waE2E.ExtendedTextMessage{
			Text: proto.String(text),
			ContextInfo: &waE2E.ContextInfo{
				StanzaID:      &evt.Info.ID,
				Participant:   proto.String(evt.Info.Sender.String()),
				QuotedMessage: evt.Message,
			},
		},
	}

	_, err := client.SendMessage(ctx, evt.Info.Chat, msg)
	return err
}
