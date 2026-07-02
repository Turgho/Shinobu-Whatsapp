package admin

import (
	"context"
	"fmt"
	"strings"

	"github.com/Turgho/Shinobu-Whatsapp/internal/commands"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

// truncateForWhatsApp corta texto longo e adiciona indicador de corte.
func truncateForWhatsApp(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "\n... (truncado)"
}

// protoToText formata uma mensagem protobuf no formato textproto.
// Retorna string vazia se a entrada for nil ou se a marshaling falhar.
func protoToText(m proto.Message) string {
	if m == nil {
		return ""
	}
	b, err := prototext.MarshalOptions{
		Multiline: true,
		Indent:    "  ",
	}.Marshal(m)
	if err != nil {
		return ""
	}
	return string(b)
}

// RawMsgHandler dumps o struct cru do whatsmeow de uma mensagem citada.
func RawMsgHandler(ctx context.Context, cli *whatsmeow.Client, evt *events.Message, _ []string) error {
	ext := evt.Message.GetExtendedTextMessage()
	if ext == nil || ext.GetContextInfo().GetQuotedMessage() == nil {
		return whatsapp.Reply(ctx, cli, evt, "📦 Use este comando respondendo a uma mensagem para ver o struct cru do whatsmeow.\n\nEx: responda a uma mensagem com `!rawmsg`")
	}

	qMsg := ext.GetContextInfo().GetQuotedMessage()
	dump := protoToText(qMsg)
	if dump == "" {
		dump = fmt.Sprintf("%+v", qMsg)
	}
	truncated := truncateForWhatsApp(dump, 3000)
	body := fmt.Sprintf("📦 *Mensagem crua (whatsmeow)*\n```\n%s\n```", truncated)

	return whatsapp.Reply(ctx, cli, evt, body)
}

// ChatInfoHandler mostra informações do chat atual (grupo ou DM).
func ChatInfoHandler(waClient *whatsmeow.Client) commands.HandlerFunc {
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, _ []string) error {
		jid := evt.Info.Chat
		jidStr := jid.String()

		if !evt.Info.IsGroup {
			return whatsapp.Reply(ctx, client, evt, fmt.Sprintf("💬 *Informações do chat*\nJID: `%s`\nTipo: DM", jidStr))
		}

		group, err := waClient.GetGroupInfo(ctx, jid)
		if err != nil {
			return whatsapp.Reply(ctx, client, evt, fmt.Sprintf("💬 *Informações do chat*\nJID: `%s`\nTipo: Grupo\nErro ao buscar info: %v", jidStr, err))
		}

		adminCount := 0
		for _, p := range group.Participants {
			if p.IsAdmin || p.IsSuperAdmin {
				adminCount++
			}
		}

		nome := group.GroupName.Name
		if nome == "" {
			nome = "N/A"
		}

		msg := fmt.Sprintf("💬 *Informações do chat*\nJID: `%s`\nTipo: Grupo\nNome: %s\nParticipantes: %d\nAdmins: %d",
			jidStr, nome, group.ParticipantCount, adminCount)

		return whatsapp.Reply(ctx, client, evt, msg)
	}
}

// RawEventHandler dumps o evento completo do whatsmeow (Info + Message + RawMessage).
func RawEventHandler(ctx context.Context, cli *whatsmeow.Client, evt *events.Message, _ []string) error {
	// Sempre funciona — usa o evento atual, não requer reply
	var parts []string

	parts = append(parts, "--- Info ---")
	parts = append(parts, fmt.Sprintf("%+v", evt.Info))

	if m := protoToText(evt.Message); m != "" {
		parts = append(parts, "--- Message ---")
		parts = append(parts, m)
	}
	if rm := protoToText(evt.RawMessage); rm != "" {
		parts = append(parts, "--- RawMessage ---")
		parts = append(parts, rm)
	}

	dump := strings.Join(parts, "\n")
	truncated := truncateForWhatsApp(dump, 3000)
	body := fmt.Sprintf("📦 *Evento completo (whatsmeow)*\n```\n%s\n```", truncated)

	return whatsapp.Reply(ctx, cli, evt, body)
}

// GroupJIDHandler é um atalho que retorna apenas o JID do grupo atual.
func GroupJIDHandler(ctx context.Context, cli *whatsmeow.Client, evt *events.Message, _ []string) error {
	if !evt.Info.IsGroup {
		return whatsapp.Reply(ctx, cli, evt, "📋 Este não é um grupo.")
	}
	return whatsapp.Reply(ctx, cli, evt, fmt.Sprintf("📋 `%s`", evt.Info.Chat.String()))
}
