package admin

import (
	"context"
	"fmt"

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
			name = whatsapp.ResolveContactName(ctx, waClient, target)
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
			if pnJID, ok := whatsapp.ResolvePNFromLID(ctx, waClient, target); ok {
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

		msg := fmt.Sprintf("🆔 *Informações do contato*\nNome: %s\nJID: `%s`\n", name, target.String())
		if isLID {
			msg += fmt.Sprintf("LID base: `%s`\n", lidBase)
		} else if lidBase != "" {
			msg += fmt.Sprintf("LID: `%s`\n", lidBase)
		}
		if pnResolved != "" {
			msg += fmt.Sprintf("PN resolvido: `%s`\n", pnResolved)
		}
		msg += fmt.Sprintf("Tipo: %s\n", tipo)
		if evt.Info.IsGroup {
			msg += fmt.Sprintf("Grupo JID: `%s`", evt.Info.Chat.String())
		}

		return whatsapp.Reply(ctx, client, evt, msg)
	}
}
