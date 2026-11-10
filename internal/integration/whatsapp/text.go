package whatsapp

import (
	"context"
	"fmt"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

// maxTextLength é o limite seguro de texto do WhatsApp. Acima de ~4096
// caracteres o envio é recusado pelo servidor — e o bot ficaria sem
// resposta nenhuma, sem erro visível.
const maxTextLength = 3500

// truncateText corta o texto no limite, preservando runes (nada de cortar
// um caractere UTF-8 no meio) e avisando que houve corte.
func truncateText(text string) string {
	runes := []rune(text)
	if len(runes) <= maxTextLength {
		return text
	}
	const sufixo = "… (continua)"
	return string(runes[:maxTextLength-len([]rune(sufixo))]) + sufixo
}

// SendText envia texto; reply=true cita a mensagem que disparou o comando.
func SendText(ctx context.Context, client *whatsmeow.Client, evt *events.Message, text string, reply bool) error {
	return withTyping(ctx, client, evt, types.ChatPresenceMediaText, func() error {
		msg := &waE2E.Message{
			ExtendedTextMessage: &waE2E.ExtendedTextMessage{
				Text:        proto.String(truncateText(text)),
				ContextInfo: buildContext(evt, reply, nil),
			},
		}
		_, err := client.SendMessage(ctx, evt.Info.Chat, msg)
		if err != nil {
			return fmt.Errorf("whatsapp: send text: %w", err)
		}
		return nil
	})
}

// SendTextWithMentions envia texto com @; reply=true cita a mensagem original.
func SendTextWithMentions(ctx context.Context, client *whatsmeow.Client, evt *events.Message, text string, mentions []string, reply bool) error {
	return withTyping(ctx, client, evt, types.ChatPresenceMediaText, func() error {
		msg := &waE2E.Message{
			ExtendedTextMessage: &waE2E.ExtendedTextMessage{
				Text:        proto.String(truncateText(text)),
				ContextInfo: buildContext(evt, reply, mentions),
			},
		}
		_, err := client.SendMessage(ctx, evt.Info.Chat, msg)
		if err != nil {
			return fmt.Errorf("whatsapp: send text with mentions: %w", err)
		}
		return nil
	})
}

// SendTextToJID envia texto para um JID sem evento (ex.: scheduler de aniversários).
func SendTextToJID(ctx context.Context, client *whatsmeow.Client, chat types.JID, text string, mentions []string) error {
	_ = client.SendChatPresence(ctx, chat, types.ChatPresenceComposing, types.ChatPresenceMediaText)
	defer client.SendChatPresence(ctx, chat, types.ChatPresencePaused, types.ChatPresenceMediaText)

	msg := &waE2E.Message{
		ExtendedTextMessage: &waE2E.ExtendedTextMessage{
			Text:        proto.String(truncateText(text)),
			ContextInfo: mentionContext(mentions),
		},
	}
	_, err := client.SendMessage(ctx, chat, msg)
	if err != nil {
		return fmt.Errorf("whatsapp: send text to jid: %w", err)
	}
	return nil
}

// SendAllToJID envia texto com @all nativo para um JID.
func SendAllToJID(ctx context.Context, client *whatsmeow.Client, chat types.JID, text string) error {
	nonJID := uint32(1)
	_ = client.SendChatPresence(ctx, chat, types.ChatPresenceComposing, types.ChatPresenceMediaText)
	defer client.SendChatPresence(ctx, chat, types.ChatPresencePaused, types.ChatPresenceMediaText)

	msg := &waE2E.Message{
		ExtendedTextMessage: &waE2E.ExtendedTextMessage{
			Text: proto.String(truncateText(text)),
			ContextInfo: &waE2E.ContextInfo{
				NonJIDMentions: &nonJID,
			},
		},
	}
	_, err := client.SendMessage(ctx, chat, msg)
	if err != nil {
		return fmt.Errorf("whatsapp: send all to jid: %w", err)
	}
	return nil
}
