package sticker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"

	"go.mau.fi/whatsmeow"
)

const stickerFile = "stickers.json"

// StickerData armazena os campos necessários para reenviar um sticker sem novo upload
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

	err = json.Unmarshal(data, &store)
	return store, err
}

func saveStore(store StickerStore) error {
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(stickerFile, data, 0644)
}

// Get retorna um sticker pelo nome, ou false se não existir
func Get(name string) (StickerData, bool) {
	store, err := loadStore()
	if err != nil {
		return StickerData{}, false
	}
	s, ok := store[strings.ToLower(name)]
	return s, ok
}

// List retorna os nomes de todos os stickers cadastrados
func List() []string {
	store, err := loadStore()
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(store))
	for name := range store {
		names = append(names, name)
	}
	return names
}

// ─── Handler de DM ────────────────────────────────────────────────────────────

// HandleStickerDM processa mensagens de DM com sticker.
// Fluxo: usuário manda !sticker salvar <nome> com um sticker citado ou anexado.
// Só funciona fora de grupos (DM).
func HandleStickerDM(client *whatsmeow.Client, evt *events.Message, ownerNumber string) {
	// Só DM e só o dono
	if evt.Info.IsGroup {
		return
	}
	if evt.Info.Sender.User != ownerNumber {
		return
	}

	msg := ""
	if evt.Message.GetConversation() != "" {
		msg = evt.Message.GetConversation()
	} else if evt.Message.GetExtendedTextMessage() != nil {
		msg = evt.Message.GetExtendedTextMessage().GetText()
	}

	msg = strings.TrimSpace(msg)
	parts := strings.Fields(strings.ToLower(msg))

	// Comando: !sticker salvar <nome>
	if len(parts) >= 3 && parts[0] == "!sticker" && parts[1] == "salvar" {
		name := strings.ToLower(parts[2])

		// Tenta pegar o sticker da mensagem citada
		sticker := extractStickerFromQuoted(evt)
		if sticker == nil {
			// Tenta pegar da própria mensagem
			sticker = evt.Message.GetStickerMessage()
		}

		if sticker == nil {
			sendText(client, evt, "❌ Manda o sticker junto ou cita uma mensagem com sticker.")
			return
		}

		err := saveSticker(name, sticker)
		if err != nil {
			sendText(client, evt, "❌ Erro ao salvar: "+err.Error())
			return
		}

		sendText(client, evt, fmt.Sprintf("✅ Sticker *%s* salvo!", name))
		return
	}

	// Comando: !sticker remover <nome>
	if len(parts) >= 3 && parts[0] == "!sticker" && parts[1] == "remover" {
		name := strings.ToLower(parts[2])

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
		saveStore(store)
		sendText(client, evt, fmt.Sprintf("🗑️ Sticker *%s* removido.", name))
		return
	}

	// Comando: !sticker lista
	if len(parts) >= 2 && parts[0] == "!sticker" && parts[1] == "lista" {
		names := List()
		if len(names) == 0 {
			sendText(client, evt, "📋 Nenhum sticker cadastrado ainda.")
			return
		}
		sendText(client, evt, "🗂️ Stickers cadastrados:\n"+strings.Join(names, ", "))
		return
	}
}

// ─── Envio ────────────────────────────────────────────────────────────────────

// Send envia um sticker salvo pelo nome para um chat
func Send(ctx context.Context, client *whatsmeow.Client, evt *events.Message, name string) error {
	s, ok := Get(name)
	if !ok {
		return fmt.Errorf("sticker '%s' não encontrado", name)
	}

	_, err := client.SendMessage(ctx, evt.Info.Chat, &waE2E.Message{
		StickerMessage: &waE2E.StickerMessage{
			URL:           proto.String(s.URL),
			DirectPath:    proto.String(s.DirectPath),
			MediaKey:      s.MediaKey,
			FileEncSHA256: s.FileEncSHA256,
			FileSHA256:    s.FileSHA256,
			FileLength:    proto.Uint64(s.FileLength),
			Mimetype:      proto.String("image/webp"),
			IsAnimated:    proto.Bool(s.IsAnimated),
		},
	})
	return err
}

// ─── Helpers internos ─────────────────────────────────────────────────────────

func saveSticker(name string, s *waE2E.StickerMessage) error {
	store, err := loadStore()
	if err != nil {
		return err
	}

	store[name] = StickerData{
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

// extractStickerFromQuoted tenta extrair o sticker de uma mensagem citada
func extractStickerFromQuoted(evt *events.Message) *waE2E.StickerMessage {
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

func sendText(client *whatsmeow.Client, evt *events.Message, text string) {
	client.SendMessage(context.Background(), evt.Info.Chat, &waE2E.Message{
		Conversation: proto.String(text),
	})
}
