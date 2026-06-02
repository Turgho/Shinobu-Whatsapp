package sticker

import (
	"errors"
	"sort"
	"strings"

	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/store"
	"go.mau.fi/whatsmeow/proto/waE2E"
)

// errNotFound é um sentinel error usado internamente pelo Update callback
// para distinguir "sticker não existe" de erro de I/O.
// O caller (Delete) usa errors.Is para decidir se retorna false ou propaga o erro.
var errNotFound = errors.New("sticker não encontrado")

const stickerFile = "assets/stickers/stickers.json"

type Data struct {
	URL           string `json:"url"`
	DirectPath    string `json:"direct_path"`
	MediaKey      []byte `json:"media_key"`
	FileEncSHA256 []byte `json:"file_enc_sha256"`
	FileSHA256    []byte `json:"file_sha256"`
	FileLength    uint64 `json:"file_length"`
	IsAnimated    bool   `json:"is_animated"`
}

type stickerStore map[string]Data

var js = store.NewJSONStore[stickerStore](stickerFile)

func NormalizeName(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func Get(name string) (Data, bool) {
	s, err := js.Read()
	if err != nil {
		return Data{}, false
	}
	d, ok := s[NormalizeName(name)]
	return d, ok
}

func List() []string {
	s, err := js.Read()
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

func Save(name string, msg *waE2E.StickerMessage) error {
	return js.Update(func(s stickerStore) (stickerStore, error) {
		if s == nil {
			s = make(stickerStore)
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
		return s, nil
	})
}

func Delete(name string) (bool, error) {
	var deleted bool
	err := js.Update(func(s stickerStore) (stickerStore, error) {
		if s == nil {
			return s, nil
		}
		key := NormalizeName(name)
		if _, exists := s[key]; !exists {
			return s, errNotFound
		}
		delete(s, key)
		deleted = true
		return s, nil
	})
	if errors.Is(err, errNotFound) {
		return false, nil
	}
	return deleted, err
}
