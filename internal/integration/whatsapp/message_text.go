package whatsapp

import (
	"strings"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
)

// PlainTextFromProto retorna Conversation ou ExtendedText (sem captions de mídia).
func PlainTextFromProto(msg *waE2E.Message) string {
	if msg == nil {
		return ""
	}
	if t := msg.GetConversation(); t != "" {
		return t
	}
	if ext := msg.GetExtendedTextMessage(); ext != nil {
		return ext.GetText()
	}
	return ""
}

// VisibleTextFromEvent extrai o texto que pode disparar comandos: corpo, extended ou caption.
func VisibleTextFromEvent(evt *events.Message) string {
	if evt == nil || evt.Message == nil {
		return ""
	}
	m := evt.Message

	msg := m.GetConversation()
	if msg == "" && m.GetExtendedTextMessage() != nil {
		msg = m.GetExtendedTextMessage().GetText()
	}
	if msg == "" && m.GetImageMessage() != nil {
		msg = m.GetImageMessage().GetCaption()
	}
	if msg == "" && m.GetVideoMessage() != nil {
		msg = m.GetVideoMessage().GetCaption()
	}
	if msg == "" && m.GetDocumentMessage() != nil {
		msg = m.GetDocumentMessage().GetCaption()
	}

	return strings.TrimSpace(msg)
}
