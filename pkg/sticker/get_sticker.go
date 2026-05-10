package sticker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"unicode"

	"github.com/Turgho/YuukoWhatsapp/internal/utils"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

const stickerFile = "assets/stickers/stickers.json"

// StickerData guarda os dados necessários para reenviar um sticker sem novo upload.
type StickerData struct {
	URL           string `json:"url"`
	DirectPath    string `json:"direct_path"`
	MediaKey      []byte `json:"media_key"`
	FileEncSHA256 []byte `json:"file_enc_sha256"`
	FileSHA256    []byte `json:"file_sha256"`
	FileLength    uint64 `json:"file_length"`
	IsAnimated    bool   `json:"is_animated"`
}

type StickerStore map[string]StickerData

func normalizeName(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func normalizeNumber(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// ─── Persistência ─────────────────────────────────────────────────────────────

func loadStore() (StickerStore, error) {
	store := make(StickerStore)

	data, err := os.ReadFile(stickerFile)
	if os.IsNotExist(err) {
		return store, nil
	}
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return store, nil
	}

	if err := json.Unmarshal(data, &store); err != nil {
		return nil, err
	}

	return store, nil
}

func saveStore(store StickerStore) error {
	if err := os.MkdirAll("assets/stickers", 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(stickerFile, data, 0644)
}

// Get retorna um sticker pelo nome.
func Get(name string) (StickerData, bool) {
	store, err := loadStore()
	if err != nil {
		return StickerData{}, false
	}
	s, ok := store[normalizeName(name)]
	return s, ok
}

// List retorna os nomes de todos os stickers cadastrados.
func List() []string {
	store, err := loadStore()
	if err != nil {
		return nil
	}

	names := make([]string, 0, len(store))
	for name := range store {
		names = append(names, name)
	}

	sort.Strings(names)
	return names
}

// ─── Handler de DM ────────────────────────────────────────────────────────────

// HandleStickerDM processa comandos de sticker somente em DM.
// Uso:
//
//	!sticker salvar <nome>   -> com sticker citado/reply
//	!sticker remover <nome>
//	!sticker lista
func HandleStickerDM(client *whatsmeow.Client, evt *events.Message, ownerNumber string) {
	if evt == nil || evt.Message == nil {
		return
	}

	// Só DM
	if evt.Info.IsGroup {
		return
	}

	// Só o dono
	if normalizeNumber(evt.Info.Sender.User) != normalizeNumber(ownerNumber) {
		return
	}

	msg := getText(evt.Message)
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return
	}

	parts := strings.Fields(msg)
	if len(parts) < 2 {
		return
	}

	if strings.ToLower(parts[0]) != "!sticker" {
		return
	}

	cmd := strings.ToLower(parts[1])

	switch cmd {
	case "salvar":
		if len(parts) < 3 {
			sendText(client, evt, "❌ Use: `!sticker salvar <nome>` respondendo/citando um sticker.")
			return
		}

		name := normalizeName(parts[2])

		// Pega primeiro o sticker da própria mensagem (se existir),
		// depois tenta pegar o sticker citado/reply.
		sticker := evt.Message.GetStickerMessage()
		if sticker == nil {
			sticker = extractStickerFromQuoted(evt)
		}

		if sticker == nil {
			sendText(client, evt, "❌ Manda o sticker junto ou responde/ cita uma mensagem com sticker.")
			return
		}

		if err := saveSticker(name, sticker); err != nil {
			sendText(client, evt, "❌ Erro ao salvar: "+err.Error())
			return
		}

		sendText(client, evt, fmt.Sprintf("✅ Sticker *%s* salvo!", name))

	case "remover":
		if len(parts) < 3 {
			sendText(client, evt, "❌ Use: `!sticker remover <nome>`")
			return
		}

		name := normalizeName(parts[2])

		store, err := loadStore()
		if err != nil {
			sendText(client, evt, "⚠️ Erro ao carregar stickers.")
			return
		}

		if _, exists := store[name]; !exists {
			sendText(client, evt, "⚠️ Sticker não encontrado.")
			return
		}

		delete(store, name)

		if err := saveStore(store); err != nil {
			sendText(client, evt, "❌ Erro ao remover sticker: "+err.Error())
			return
		}

		sendText(client, evt, fmt.Sprintf("🗑️ Sticker *%s* removido.", name))

	case "lista":
		names := List()
		if len(names) == 0 {
			sendText(client, evt, "📋 Nenhum sticker cadastrado ainda.")
			return
		}

		sendText(client, evt, "🗂️ Stickers cadastrados:\n"+strings.Join(names, ", "))
	}
}

// ─── Envio ────────────────────────────────────────────────────────────────────

// Send envia um sticker salvo pelo nome para um chat.
func Send(
	ctx context.Context,
	client *whatsmeow.Client,
	evt *events.Message,
	name string,
) error {
	s, ok := Get(name)
	if !ok {
		return fmt.Errorf("sticker '%s' não encontrado", name)
	}

	uploaded := &whatsmeow.UploadResponse{
		URL:           s.URL,
		DirectPath:    s.DirectPath,
		MediaKey:      s.MediaKey,
		FileEncSHA256: s.FileEncSHA256,
		FileSHA256:    s.FileSHA256,
		FileLength:    s.FileLength,
	}

	return utils.SendSticker(ctx, client, evt, uploaded, s.IsAnimated)
}

// ─── Helpers internos ─────────────────────────────────────────────────────────

func saveSticker(name string, s *waE2E.StickerMessage) error {
	store, err := loadStore()
	if err != nil {
		return err
	}

	store[normalizeName(name)] = StickerData{
		URL:           s.GetURL(),
		DirectPath:    s.GetDirectPath(),
		MediaKey:      s.GetMediaKey(),
		FileEncSHA256: s.GetFileEncSHA256(),
		FileSHA256:    s.GetFileSHA256(),
		FileLength:    s.GetFileLength(),
		IsAnimated:    s.GetIsAnimated(),
	}

	return saveStore(store)
}

func extractStickerFromQuoted(evt *events.Message) *waE2E.StickerMessage {
	if evt == nil || evt.Message == nil {
		return nil
	}

	ext := evt.Message.GetExtendedTextMessage()
	if ext == nil {
		return nil
	}

	ctx := ext.GetContextInfo()
	if ctx == nil {
		return nil
	}

	quoted := ctx.GetQuotedMessage()
	if quoted == nil {
		return nil
	}

	return quoted.GetStickerMessage()
}

func getText(msg *waE2E.Message) string {
	if msg == nil {
		return ""
	}

	if msg.GetConversation() != "" {
		return msg.GetConversation()
	}

	if ext := msg.GetExtendedTextMessage(); ext != nil {
		return ext.GetText()
	}

	return ""
}

func sendText(client *whatsmeow.Client, evt *events.Message, text string) {
	_, _ = client.SendMessage(context.Background(), evt.Info.Chat, &waE2E.Message{
		Conversation: proto.String(text),
	})
}
