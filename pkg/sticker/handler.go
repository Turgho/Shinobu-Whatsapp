package sticker

import (
	"context"
	"fmt"
	"strings"

	"github.com/Turgho/YuukoWhatsapp/internal/utils"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
)

// HandleDM processa comandos !fig enviados como mensagem direta pelo dono.
//
// Comandos:
//
//	!fig salvar <nome>   — salva um sticker (mande junto ou cite um)
//	!fig remover <nome>  — remove um sticker salvo
//	!fig lista           — lista todos os stickers cadastrados
func HandleDM(ctx context.Context, client *whatsmeow.Client, evt *events.Message, ownerNumber string) {
	if evt == nil || evt.Message == nil || evt.Info.IsGroup {
		return
	}
	if NormalizeNumber(evt.Info.Sender.User) != NormalizeNumber(ownerNumber) {
		return
	}

	text := strings.TrimSpace(extractText(evt.Message))
	parts := strings.Fields(text)

	// Espera pelo menos: !fig <cmd>
	if len(parts) < 2 || strings.ToLower(parts[0]) != "!fig" {
		return
	}

	cmd := strings.ToLower(parts[1])

	switch cmd {
	case "salvar":
		handleSave(ctx, client, evt, parts)
	case "remover":
		handleDelete(ctx, client, evt, parts)
	case "lista":
		handleList(ctx, client, evt)
	}
}

// ─── Sub-handlers ─────────────────────────────────────────────────────────────

func handleSave(ctx context.Context, client *whatsmeow.Client, evt *events.Message, parts []string) {
	if len(parts) < 3 {
		reply(ctx, client, evt, "❌ Use: `!fig salvar <nome>` — mande ou cite um sticker junto.")
		return
	}

	name := parts[2]

	sticker := evt.Message.GetStickerMessage()
	if sticker == nil {
		sticker = stickerFromQuoted(evt)
	}
	if sticker == nil {
		reply(ctx, client, evt, "❌ Mande o sticker junto ou responda a uma mensagem com sticker.")
		return
	}

	if err := Save(name, sticker); err != nil {
		reply(ctx, client, evt, "❌ Erro ao salvar: "+err.Error())
		return
	}

	reply(ctx, client, evt, fmt.Sprintf("✅ Sticker *%s* salvo!", NormalizeName(name)))
}

func handleDelete(ctx context.Context, client *whatsmeow.Client, evt *events.Message, parts []string) {
	if len(parts) < 3 {
		reply(ctx, client, evt, "❌ Use: `!fig remover <nome>`")
		return
	}

	name := parts[2]

	deleted, err := Delete(name)
	if err != nil {
		reply(ctx, client, evt, "❌ Erro ao remover: "+err.Error())
		return
	}
	if !deleted {
		reply(ctx, client, evt, fmt.Sprintf("⚠️ Sticker *%s* não encontrado.", NormalizeName(name)))
		return
	}

	reply(ctx, client, evt, fmt.Sprintf("🗑️ Sticker *%s* removido.", NormalizeName(name)))
}

func handleList(ctx context.Context, client *whatsmeow.Client, evt *events.Message) {
	names := List()
	if len(names) == 0 {
		reply(ctx, client, evt, "📋 Nenhum sticker cadastrado ainda.")
		return
	}
	reply(ctx, client, evt, "🗂️ Stickers cadastrados:\n"+strings.Join(names, ", "))
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// reply envia uma resposta de texto citando a mensagem original.
func reply(ctx context.Context, client *whatsmeow.Client, evt *events.Message, text string) {
	_ = utils.SendText(ctx, client, evt, text, true)
}

// stickerFromQuoted extrai um sticker de uma mensagem citada/respondida.
func stickerFromQuoted(evt *events.Message) *waE2E.StickerMessage {
	quoted := evt.Message.GetExtendedTextMessage().GetContextInfo().GetQuotedMessage()
	if quoted == nil {
		return nil
	}
	return quoted.GetStickerMessage()
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
