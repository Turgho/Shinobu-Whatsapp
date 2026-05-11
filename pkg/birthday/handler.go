package birthday

import (
	"context"
	"fmt"
	"strings"

	"time"

	"github.com/Turgho/YuukoWhatsapp/internal/utils"
	"go.mau.fi/whatsmeow"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
)

// HandleDM processa comandos !aniversário enviados como mensagem direta pelo dono.
//
// Comandos:
//
//	!aniversário salvar @pessoa DD/MM — salva aniversário de outra pessoa
//	!aniversário remover @pessoa      — remove aniversário de outra pessoa
func HandleDM(ctx context.Context, client *whatsmeow.Client, evt *events.Message, ownerNumber string) {
	if evt == nil || evt.Message == nil || evt.Info.IsGroup {
		return
	}
	if NormalizeNumber(evt.Info.Sender.User) != NormalizeNumber(ownerNumber) {
		return
	}

	text := strings.TrimSpace(extractText(evt.Message))
	parts := strings.Fields(text)

	// Espera pelo menos: !aniversário <cmd>
	if len(parts) < 2 || !isAnniversaryCmd(parts[0]) {
		return
	}

	cmd := strings.ToLower(parts[1])

	switch cmd {
	case "salvar":
		handleDMSave(ctx, client, evt, parts)
	case "remover":
		handleDMRemove(ctx, client, evt, parts)
	}
}

// ─── Sub-handlers ─────────────────────────────────────────────────────────────

func handleDMSave(ctx context.Context, client *whatsmeow.Client, evt *events.Message, parts []string) {
	if len(parts) < 4 {
		reply(ctx, client, evt, "❌ Use: `!aniversário salvar @pessoa DD/MM`")
		return
	}

	mention := extractMention(evt)
	if mention == "" {
		reply(ctx, client, evt, "❌ Marque a pessoa corretamente.")
		return
	}

	name := strings.TrimPrefix(parts[2], "@")
	dateStr := parts[3]

	day, month, err := parseDate(dateStr)
	if err != nil {
		reply(ctx, client, evt, "❌ Data inválida. Use DD/MM (ex: 25/12).")
		return
	}

	// Em DM não há grupo — usa o JID do dono como chave global.
	groupKey := "dm:" + evt.Info.Sender.ToNonAD().String()

	if err := Set(groupKey, mention, name, day, month); err != nil {
		reply(ctx, client, evt, "❌ Erro ao salvar: "+err.Error())
		return
	}

	reply(ctx, client, evt, fmt.Sprintf("✅ Aniversário de *%s* salvo para %02d/%02d! 🎂", name, day, month))
}

func handleDMRemove(ctx context.Context, client *whatsmeow.Client, evt *events.Message, parts []string) {
	if len(parts) < 3 {
		reply(ctx, client, evt, "❌ Use: `!aniversário remover @pessoa`")
		return
	}

	mention := extractMention(evt)
	if mention == "" {
		reply(ctx, client, evt, "❌ Marque a pessoa corretamente.")
		return
	}

	name := strings.TrimPrefix(parts[2], "@")
	groupKey := "dm:" + evt.Info.Sender.ToNonAD().String()

	deleted, err := Remove(groupKey, mention)
	if err != nil {
		reply(ctx, client, evt, "❌ Erro ao remover: "+err.Error())
		return
	}
	if !deleted {
		reply(ctx, client, evt, fmt.Sprintf("⚠️ Aniversário de *%s* não encontrado.", name))
		return
	}

	reply(ctx, client, evt, fmt.Sprintf("🗑️ Aniversário de *%s* removido.", name))
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// reply envia uma resposta de texto citando a mensagem original.
func reply(ctx context.Context, client *whatsmeow.Client, evt *events.Message, text string) {
	_ = utils.SendText(ctx, client, evt, text, true)
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

// extractText retorna o texto puro de uma mensagem, verificando
// tanto o formato Conversation quanto o ExtendedTextMessage.
func extractText(msg *waE2E.Message) string {
	if msg == nil {
		return ""
	}
	if t := msg.GetConversation(); t != "" {
		return t
	}
	return msg.GetExtendedTextMessage().GetText()
}

// ─── Handlers de grupo ────────────────────────────────────────────────────────

// handleGroup roteia os subcomandos do grupo.
func HandleGroup(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
	switch strings.ToLower(args[0]) {
	case "lista":
		return handleList(ctx, client, evt)
	case "remover":
		return handleRemove(ctx, client, evt)
	default:
		return handleSet(ctx, client, evt, args[0])
	}
}

func handleSet(ctx context.Context, client *whatsmeow.Client, evt *events.Message, dateStr string) error {
	day, month, err := parseDate(dateStr)
	if err != nil {
		return utils.SendText(ctx, client, evt, "❌ Data inválida. Use o formato DD/MM (ex: 25/12).", true)
	}

	userJID := evt.Info.Sender.ToNonAD().String()
	name := senderName(evt)

	if err := Set(evt.Info.Chat.String(), userJID, name, day, month); err != nil {
		return utils.SendText(ctx, client, evt, "❌ Erro ao salvar aniversário: "+err.Error(), true)
	}

	return utils.SendText(ctx, client, evt,
		fmt.Sprintf("✅ Aniversário de *%s* salvo para %02d/%02d! 🎂", name, day, month), true)
}

func handleRemove(ctx context.Context, client *whatsmeow.Client, evt *events.Message) error {
	userJID := evt.Info.Sender.ToNonAD().String()
	name := senderName(evt)

	deleted, err := Remove(evt.Info.Chat.String(), userJID)
	if err != nil {
		return utils.SendText(ctx, client, evt, "❌ Erro ao remover: "+err.Error(), true)
	}
	if !deleted {
		return utils.SendText(ctx, client, evt,
			fmt.Sprintf("⚠️ Nenhum aniversário encontrado para *%s*.", name), true)
	}

	return utils.SendText(ctx, client, evt,
		fmt.Sprintf("🗑️ Aniversário de *%s* removido.", name), true)
}

func handleList(ctx context.Context, client *whatsmeow.Client, evt *events.Message) error {
	entries := ListGroup(evt.Info.Chat.String())
	if len(entries) == 0 {
		return utils.SendText(ctx, client, evt, "📋 Nenhum aniversário cadastrado neste grupo.", true)
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

	return utils.SendText(ctx, client, evt, sb.String(), true)
}

func senderName(evt *events.Message) string {
	if evt.Info.PushName != "" {
		return evt.Info.PushName
	}
	return evt.Info.Sender.ToNonAD().User
}

// isAnniversaryCmd aceita o prefixo com ou sem acento.
func isAnniversaryCmd(s string) bool {
	s = strings.ToLower(s)
	return s == "!aniversário" || s == "!aniversario"
}
