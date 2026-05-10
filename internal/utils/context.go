package utils

import (
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

// replyContext returns a ContextInfo that quotes the original message.
// Pass mentions as a list of JIDs (e.g. "5511999999999@s.whatsapp.net"),
// or nil if no mentions are needed.
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

// mentionContext returns a ContextInfo with mentions but WITHOUT quoting.
func mentionContext(mentions []string) *waE2E.ContextInfo {
	if len(mentions) == 0 {
		return nil
	}
	return &waE2E.ContextInfo{
		MentionedJID: mentions,
	}
}

// buildContext returns the right ContextInfo based on the reply flag.
// reply=true  → quotes the original message
// reply=false → plain send, no quote
func buildContext(evt *events.Message, reply bool, mentions []string) *waE2E.ContextInfo {
	if reply {
		return replyContext(evt, mentions)
	}
	return mentionContext(mentions)
}
