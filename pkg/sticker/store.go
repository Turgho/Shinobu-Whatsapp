package sticker

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"unicode"

	"go.mau.fi/whatsmeow/proto/waE2E"
)

const stickerFile = "assets/stickers/stickers.json"

// Data guarda os campos necessários para reenviar um sticker sem novo upload.
type Data struct {
	URL           string `json:"url"`
	DirectPath    string `json:"direct_path"`
	MediaKey      []byte `json:"media_key"`
	FileEncSHA256 []byte `json:"file_enc_sha256"`
	FileSHA256    []byte `json:"file_sha256"`
	FileLength    uint64 `json:"file_length"`
	IsAnimated    bool   `json:"is_animated"`
}

// store é o mapa em memória de nome → dados do sticker.
type store map[string]Data

// NormalizeName converte o nome para minúsculo e remove espaços extras.
func NormalizeName(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// NormalizeNumber remove todos os caracteres não numéricos de um telefone.
func NormalizeNumber(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// ─── Persistência ─────────────────────────────────────────────────────────────

func load() (store, error) {
	s := make(store)

	data, err := os.ReadFile(stickerFile)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("sticker: leitura do arquivo: %w", err)
	}
	if len(data) == 0 {
		return s, nil
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("sticker: parse do json: %w", err)
	}
	return s, nil
}

func save(s store) error {
	if err := os.MkdirAll("assets/stickers", 0755); err != nil {
		return fmt.Errorf("sticker: criar pasta: %w", err)
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("sticker: serializar: %w", err)
	}
	if err := os.WriteFile(stickerFile, data, 0644); err != nil {
		return fmt.Errorf("sticker: salvar arquivo: %w", err)
	}
	return nil
}

// ─── Operações públicas do store ──────────────────────────────────────────────

// Get retorna um sticker pelo nome, ou false se não encontrado.
func Get(name string) (Data, bool) {
	s, err := load()
	if err != nil {
		return Data{}, false
	}
	d, ok := s[NormalizeName(name)]
	return d, ok
}

// List retorna todos os nomes de stickers cadastrados, em ordem alfabética.
func List() []string {
	s, err := load()
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(s))
	for name := range s {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Save salva um sticker a partir de um waE2E.StickerMessage com o nome dado.
func Save(name string, msg *waE2E.StickerMessage) error {
	s, err := load()
	if err != nil {
		return err
	}
	s[NormalizeName(name)] = Data{
		URL:           msg.GetURL(),
		DirectPath:    msg.GetDirectPath(),
		MediaKey:      msg.GetMediaKey(),
		FileEncSHA256: msg.GetFileEncSHA256(),
		FileSHA256:    msg.GetFileSHA256(),
		FileLength:    msg.GetFileLength(),
		IsAnimated:    msg.GetIsAnimated(),
	}
	return save(s)
}

// Delete remove um sticker pelo nome. Retorna false se ele não existia.
func Delete(name string) (bool, error) {
	s, err := load()
	if err != nil {
		return false, err
	}
	key := NormalizeName(name)
	if _, exists := s[key]; !exists {
		return false, nil
	}
	delete(s, key)
	return true, save(s)
}
