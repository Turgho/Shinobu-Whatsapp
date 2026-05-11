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

// SendText sends a plain text message.
// reply=true  → quotes the original message
// reply=false → sends standalone in the chat
func SendText(ctx context.Context, client *whatsmeow.Client, evt *events.Message, text string, reply bool) error {
	return withTyping(ctx, client, evt, types.ChatPresenceMediaText, func() error {
		msg := &waE2E.Message{
			ExtendedTextMessage: &waE2E.ExtendedTextMessage{
				Text:        proto.String(text),
				ContextInfo: buildContext(evt, reply, nil),
			},
		}
		_, err := client.SendMessage(ctx, evt.Info.Chat, msg)
		if err != nil {
			return fmt.Errorf("utils/text: %w", err)
		}
		return nil
	})
}

// SendTextWithMentions sends a text message tagging one or more users.
// reply=true  → quotes the original message
// reply=false → sends standalone in the chat
func SendTextWithMentions(ctx context.Context, client *whatsmeow.Client, evt *events.Message, text string, mentions []string, reply bool) error {
	return withTyping(ctx, client, evt, types.ChatPresenceMediaText, func() error {
		msg := &waE2E.Message{
			ExtendedTextMessage: &waE2E.ExtendedTextMessage{
				Text:        proto.String(text),
				ContextInfo: buildContext(evt, reply, mentions),
			},
		}
		_, err := client.SendMessage(ctx, evt.Info.Chat, msg)
		if err != nil {
			return fmt.Errorf("utils/text: %w", err)
		}
		return nil
	})
}

// SendTextToChat envia uma mensagem de texto diretamente para um JID,
// sem precisar de um evt — usado pelo scheduler de aniversários.
func SendTextToChat(ctx context.Context, client *whatsmeow.Client, chat types.JID, text string, mentions []string) error {
	_ = client.SendChatPresence(ctx, chat, types.ChatPresenceComposing, types.ChatPresenceMediaText)
	defer client.SendChatPresence(ctx, chat, types.ChatPresencePaused, types.ChatPresenceMediaText)

	msg := &waE2E.Message{
		ExtendedTextMessage: &waE2E.ExtendedTextMessage{
			Text:        proto.String(text),
			ContextInfo: mentionContext(mentions),
		},
	}
	_, err := client.SendMessage(ctx, chat, msg)
	if err != nil {
		return fmt.Errorf("utils/text: %w", err)
	}
	return nil
}

// SendTextToJID envia uma mensagem de texto diretamente para um JID,
// sem precisar de um evt — usado pelo scheduler de aniversários.
func SendTextToJID(ctx context.Context, client *whatsmeow.Client, chat types.JID, text string, mentions []string) error {
	_ = client.SendChatPresence(ctx, chat, types.ChatPresenceComposing, types.ChatPresenceMediaText)
	defer client.SendChatPresence(ctx, chat, types.ChatPresencePaused, types.ChatPresenceMediaText)

	msg := &waE2E.Message{
		ExtendedTextMessage: &waE2E.ExtendedTextMessage{
			Text:        proto.String(text),
			ContextInfo: mentionContext(mentions),
		},
	}
	_, err := client.SendMessage(ctx, chat, msg)
	if err != nil {
		return fmt.Errorf("utils/text: %w", err)
	}
	return nil
}
