package birthday

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// HandleGroup roteia os subcomandos do !aniversario no grupo.
//
// Qualquer membro:
//
//	!aniversario DD/MM       — salva o próprio aniversário
//	!aniversario remover     — remove o próprio aniversário
//	!aniversario lista       — lista aniversários do grupo
//
// Dono/admins apenas:
//
//	!aniversario salvar @pessoa DD/MM — salva aniversário de outra pessoa
//	!aniversario remover @pessoa      — remove aniversário de outra pessoa
func HandleGroup(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string, ownerNumber string, admins []string) error {
	if len(args) == 0 {
		return usage(ctx, client, evt)
	}

	switch strings.ToLower(args[0]) {
	case "lista":
		return handleList(ctx, client, evt)

	case "salvar":
		// Salvar outro → só dono/admin
		if len(args) >= 3 && strings.HasPrefix(args[1], "@") {
			if !isPrivileged(evt, ownerNumber, admins) {
				return whatsapp.SendText(ctx, client, evt,
					"❌ Apenas o dono ou admins podem salvar o aniversário de outras pessoas.", true)
			}
			return handleSaveOther(ctx, client, evt, args)
		}
		// Salvar o próprio via subcomando explícito
		if len(args) >= 2 {
			return handleSaveSelf(ctx, client, evt, args[1])
		}
		return whatsapp.SendText(ctx, client, evt,
			"❌ Use: `!aniversario salvar DD/MM` ou `!aniversario salvar @pessoa DD/MM`", true)

	case "remover":
		// Remover outro → só dono/admin
		if len(args) >= 2 && strings.HasPrefix(args[1], "@") {
			if !isPrivileged(evt, ownerNumber, admins) {
				return whatsapp.SendText(ctx, client, evt,
					"❌ Apenas o dono ou admins podem remover o aniversário de outras pessoas.", true)
			}
			return handleRemoveOther(ctx, client, evt, args)
		}
		// Remover o próprio
		return handleRemoveSelf(ctx, client, evt)

	default:
		// !aniversario DD/MM — atalho para salvar o próprio
		return handleSaveSelf(ctx, client, evt, args[0])
	}
}

// ─── Salvar ───────────────────────────────────────────────────────────────────

func handleSaveSelf(ctx context.Context, client *whatsmeow.Client, evt *events.Message, dateStr string) error {
	day, month, err := parseDate(dateStr)
	if err != nil {
		return whatsapp.SendText(ctx, client, evt,
			"❌ Data inválida. Use o formato DD/MM (ex: 25/12).", true)
	}

	userJID := evt.Info.Sender.ToNonAD().String()
	name := senderName(evt)

	if err := Set(evt.Info.Chat.String(), userJID, name, day, month); err != nil {
		return whatsapp.SendText(ctx, client, evt,
			"❌ Erro ao salvar aniversário: "+err.Error(), true)
	}

	return whatsapp.SendText(ctx, client, evt,
		fmt.Sprintf("✅ Aniversário de *%s* salvo para %02d/%02d! 🎂", name, day, month), true)
}

func handleSaveOther(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
	// args: ["salvar", "@pessoa", "DD/MM"]
	if len(args) < 3 {
		return whatsapp.SendText(ctx, client, evt,
			"❌ Use: `!aniversario salvar @pessoa DD/MM`", true)
	}

	mention := extractMention(evt)
	if mention == "" {
		return whatsapp.SendText(ctx, client, evt,
			"❌ Marque a pessoa corretamente.", true)
	}

	name := strings.TrimPrefix(args[1], "@")
	day, month, err := parseDate(args[2])
	if err != nil {
		return whatsapp.SendText(ctx, client, evt,
			"❌ Data inválida. Use o formato DD/MM (ex: 25/12).", true)
	}

	if err := Set(evt.Info.Chat.String(), mention, name, day, month); err != nil {
		return whatsapp.SendText(ctx, client, evt,
			"❌ Erro ao salvar aniversário: "+err.Error(), true)
	}

	return whatsapp.SendText(ctx, client, evt,
		fmt.Sprintf("✅ Aniversário de *%s* salvo para %02d/%02d! 🎂", name, day, month), true)
}

// ─── Remover ──────────────────────────────────────────────────────────────────

func handleRemoveSelf(ctx context.Context, client *whatsmeow.Client, evt *events.Message) error {
	userJID := evt.Info.Sender.ToNonAD().String()
	name := senderName(evt)

	deleted, err := Remove(evt.Info.Chat.String(), userJID)
	if err != nil {
		return whatsapp.SendText(ctx, client, evt,
			"❌ Erro ao remover: "+err.Error(), true)
	}
	if !deleted {
		return whatsapp.SendText(ctx, client, evt,
			fmt.Sprintf("⚠️ Nenhum aniversário encontrado para *%s*.", name), true)
	}

	return whatsapp.SendText(ctx, client, evt,
		fmt.Sprintf("🗑️ Aniversário de *%s* removido.", name), true)
}

func handleRemoveOther(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
	// args: ["remover", "@pessoa"]
	mention := extractMention(evt)
	if mention == "" {
		return whatsapp.SendText(ctx, client, evt,
			"❌ Marque a pessoa corretamente.", true)
	}

	name := strings.TrimPrefix(args[1], "@")

	deleted, err := Remove(evt.Info.Chat.String(), mention)
	if err != nil {
		return whatsapp.SendText(ctx, client, evt,
			"❌ Erro ao remover: "+err.Error(), true)
	}
	if !deleted {
		return whatsapp.SendText(ctx, client, evt,
			fmt.Sprintf("⚠️ Aniversário de *%s* não encontrado.", name), true)
	}

	return whatsapp.SendText(ctx, client, evt,
		fmt.Sprintf("🗑️ Aniversário de *%s* removido.", name), true)
}

// ─── Lista ────────────────────────────────────────────────────────────────────

func handleList(ctx context.Context, client *whatsmeow.Client, evt *events.Message) error {
	entries := ListGroup(evt.Info.Chat.String())
	if len(entries) == 0 {
		return whatsapp.SendText(ctx, client, evt,
			"📋 Nenhum aniversário cadastrado neste grupo.", true)
	}

	now := time.Now()
	var upcoming, past []Entry
	for _, e := range entries {
		next := time.Date(now.Year(), time.Month(e.Month), e.Day, 0, 0, 0, 0, now.Location())
		if next.Before(now) {
			past = append(past, e)
		} else {
			upcoming = append(upcoming, e)
		}
	}

	var sb strings.Builder
	sb.WriteString("🎂 *Aniversários do grupo:*\n\n")
	for _, e := range append(upcoming, past...) {
		sb.WriteString(fmt.Sprintf("🎈 *%s* — %02d/%02d\n", e.Name, e.Day, e.Month))
	}

	return whatsapp.SendText(ctx, client, evt, sb.String(), true)
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func usage(ctx context.Context, client *whatsmeow.Client, evt *events.Message) error {
	return whatsapp.SendText(ctx, client, evt,
		"🎂 *Uso:*\n\n"+
			"!aniversario DD/MM — salva seu aniversário\n"+
			"!aniversario lista — lista aniversários do grupo\n"+
			"!aniversario remover — remove seu aniversário",
		true,
	)
}

// isPrivileged verifica se o remetente é o dono ou um dos admins.
func isPrivileged(evt *events.Message, ownerNumber string, admins []string) bool {
	sender := NormalizeNumber(evt.Info.Sender.ToNonAD().User)
	if sender == NormalizeNumber(ownerNumber) {
		return true
	}
	for _, admin := range admins {
		if sender == NormalizeNumber(admin) {
			return true
		}
	}
	return false
}

// extractMention pega o primeiro JID mencionado na mensagem.
func extractMention(evt *events.Message) string {
	if evt.Message == nil {
		return ""
	}
	mentions := evt.Message.GetExtendedTextMessage().GetContextInfo().GetMentionedJID()
	if len(mentions) == 0 {
		return ""
	}
	return mentions[0]
}

func senderName(evt *events.Message) string {
	if evt.Info.PushName != "" {
		return evt.Info.PushName
	}
	return evt.Info.Sender.ToNonAD().User
}
