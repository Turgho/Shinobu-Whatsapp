package sticker

import (
	"errors"
	"sort"
	"strings"

	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/store"
	"go.mau.fi/whatsmeow/proto/waE2E"
)

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

type Store struct {
	js *store.JSONStore[stickerStore]
}

func NewStore() *Store {
	return &Store{
		js: store.NewJSONStore[stickerStore](stickerFile),
	}
}

func NormalizeName(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func (s *Store) Get(name string) (Data, bool) {
	store, err := s.js.Read()
	if err != nil {
		return Data{}, false
	}
	d, ok := store[NormalizeName(name)]
	return d, ok
}

func (s *Store) List() []string {
	store, err := s.js.Read()
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

func (s *Store) Save(name string, msg *waE2E.StickerMessage) error {
	return s.js.Update(func(store stickerStore) (stickerStore, error) {
		if store == nil {
			store = make(stickerStore)
		}
		store[NormalizeName(name)] = Data{
			URL:           msg.GetURL(),
			DirectPath:    msg.GetDirectPath(),
			MediaKey:      msg.GetMediaKey(),
			FileEncSHA256: msg.GetFileEncSHA256(),
			FileSHA256:    msg.GetFileSHA256(),
			FileLength:    msg.GetFileLength(),
			IsAnimated:    msg.GetIsAnimated(),
		}
		return store, nil
	})
}

func (s *Store) Delete(name string) (bool, error) {
	var deleted bool
	err := s.js.Update(func(store stickerStore) (stickerStore, error) {
		if store == nil {
			return store, nil
		}
		key := NormalizeName(name)
		if _, exists := store[key]; !exists {
			return store, errNotFound
		}
		delete(store, key)
		deleted = true
		return store, nil
	})
	if errors.Is(err, errNotFound) {
		return false, nil
	}
	return deleted, err
}
