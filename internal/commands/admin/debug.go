package admin

import (
	"context"
	"fmt"
	"strings"

	"github.com/Turgho/Shinobu-Whatsapp/internal/commands"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// resolveTargetJID extrai o JID alvo de uma mensagem: prioriza reply,
// depois menção, depois o remetente da própria mensagem.
func resolveTargetJID(evt *events.Message) types.JID {
	if ext := evt.Message.GetExtendedTextMessage(); ext != nil {
		ctxInfo := ext.GetContextInfo()
		if ctxInfo != nil {
			// Reply: quoted message → participant JID
			if ctxInfo.GetQuotedMessage() != nil {
				if p := ctxInfo.GetParticipant(); p != "" {
					if jid, err := types.ParseJID(p); err == nil {
						return jid.ToNonAD()
					}
				}
			}
			// Menção: primeiro JID mencionado
			if mentions := ctxInfo.GetMentionedJID(); len(mentions) > 0 {
				if jid, err := types.ParseJID(mentions[0]); err == nil {
					return jid.ToNonAD()
				}
			}
		}
	}
	return evt.Info.Sender.ToNonAD()
}

// truncateForWhatsApp corta texto longo e adiciona indicador de corte.
func truncateForWhatsApp(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "\n... (truncado)"
}

// pushNameFromEvent tenta extrair o push name do alvo a partir do evento.
// Só funciona quando o alvo é o remetente da mensagem atual.
func pushNameFromEvent(evt *events.Message, target types.JID) string {
	if evt.Info.Sender.ToNonAD().String() == target.String() {
		return evt.Info.PushName
	}
	return ""
}

// WhoisHandler retorna informações de JID/LID de um contato.
func WhoisHandler(waClient *whatsmeow.Client) commands.HandlerFunc {
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, _ []string) error {
		target := resolveTargetJID(evt)

		name := pushNameFromEvent(evt, target)
		if name == "" {
			name = whatsapp.ResolveContactName(waClient, target)
		}
		if name == "" {
			userInfo, err := waClient.GetUserInfo(ctx, []types.JID{target})
			if err == nil {
				if info, ok := userInfo[target]; ok {
					if info.VerifiedName != nil && info.VerifiedName.Details != nil {
						if vn := info.VerifiedName.Details.GetVerifiedName(); vn != "" {
							name = vn
						}
					}
				}
			}
		}
		if name == "" {
			name = "desconhecido"
		}

		isLID := target.Server == types.HiddenUserServer
		var lidBase string
		var pnResolved string
		if isLID {
			baseJID := whatsapp.StripLIDDeviceSuffix(target)
			if baseJID != target {
				lidBase = baseJID.String()
			} else {
				lidBase = target.String()
			}
			if pnJID, ok := whatsapp.ResolvePNFromLID(waClient, target); ok {
				pnResolved = pnJID.String()
			}
		} else {
			userInfo, err := waClient.GetUserInfo(ctx, []types.JID{target})
			if err == nil {
				if info, ok := userInfo[target]; ok && !info.LID.IsEmpty() {
					lidBase = info.LID.String()
				}
			}
		}

		tipo := "DM"
		if evt.Info.IsGroup {
			tipo = "Grupo"
		}

		var b strings.Builder
		b.WriteString("🆔 *Informações do contato*\n")
		b.WriteString(fmt.Sprintf("Nome: %s\n", name))
		b.WriteString(fmt.Sprintf("JID: `%s`\n", target.String()))
		if isLID {
			b.WriteString(fmt.Sprintf("LID base: `%s`\n", lidBase))
		} else if lidBase != "" {
			b.WriteString(fmt.Sprintf("LID: `%s`\n", lidBase))
		}
		if pnResolved != "" {
			b.WriteString(fmt.Sprintf("PN resolvido: `%s`\n", pnResolved))
		}
		b.WriteString(fmt.Sprintf("Tipo: %s\n", tipo))
		if evt.Info.IsGroup {
			b.WriteString(fmt.Sprintf("Grupo JID: `%s`", evt.Info.Chat.String()))
		}

		return whatsapp.Reply(ctx, client, evt, b.String())
	}
}

// RawMsgHandler dumps o struct cru do whatsmeow de uma mensagem citada.
func RawMsgHandler(ctx context.Context, cli *whatsmeow.Client, evt *events.Message, _ []string) error {
	ext := evt.Message.GetExtendedTextMessage()
	if ext == nil || ext.GetContextInfo().GetQuotedMessage() == nil {
		return whatsapp.Reply(ctx, cli, evt, "📦 Use este comando respondendo a uma mensagem para ver o struct cru do whatsmeow.\n\nEx: responda a uma mensagem com `!rawmsg`")
	}

	qMsg := ext.GetContextInfo().GetQuotedMessage()
	dump := fmt.Sprintf("%+v", qMsg)
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

		var b strings.Builder
		b.WriteString("💬 *Informações do chat*\n")
		b.WriteString(fmt.Sprintf("JID: `%s`\n", jidStr))
		b.WriteString(fmt.Sprintf("Tipo: Grupo\n"))
		b.WriteString(fmt.Sprintf("Nome: %s\n", nome))
		b.WriteString(fmt.Sprintf("Participantes: %d\n", group.ParticipantCount))
		b.WriteString(fmt.Sprintf("Admins: %d", adminCount))

		return whatsapp.Reply(ctx, client, evt, b.String())
	}
}

// RawEventHandler dumps o evento completo do whatsmeow (Info + Message + RawMessage).
func RawEventHandler(ctx context.Context, cli *whatsmeow.Client, evt *events.Message, _ []string) error {
	// Sempre funciona — usa o evento atual, não requer reply
	dump := fmt.Sprintf("%+v", evt)
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
