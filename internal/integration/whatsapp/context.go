package whatsapp

import (
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

func replyContext(evt *events.Message, mentions []string) *waE2E.ContextInfo {
	ctx := &waE2E.ContextInfo{
		StanzaID:      proto.String(evt.Info.ID),
		Participant:   proto.String(evt.Info.Sender.ToNonAD().String()),
		QuotedMessage: evt.Message,
	}
	if len(mentions) > 0 {
		ctx.MentionedJID = mentions
	}
	return ctx
}

func mentionContext(mentions []string) *waE2E.ContextInfo {
	if len(mentions) == 0 {
		return nil
	}
	return &waE2E.ContextInfo{
		MentionedJID: mentions,
	}
}

func buildContext(evt *events.Message, reply bool, mentions []string) *waE2E.ContextInfo {
	if reply {
		return replyContext(evt, mentions)
	}
	return mentionContext(mentions)
}
