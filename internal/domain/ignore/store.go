package ignore

import (
	"sync"

	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/store"
)

const storeFile = "assets/info/ignored.json"

// Store gerencia a lista de JIDs ignorados, com persistência em JSON.
type Store struct {
	js      *store.JSONStore[[]string]
	mu      sync.RWMutex
	ignored map[string]bool
	once    sync.Once
}

func NewStore() *Store {
	return &Store{
		js: store.NewJSONStore[[]string](storeFile),
	}
}

func (s *Store) load() {
	list, err := s.js.Read()
	if err != nil || list == nil {
		s.ignored = make(map[string]bool)
		return
	}
	s.ignored = make(map[string]bool, len(list))
	for _, jid := range list {
		s.ignored[jid] = true
	}
}

func (s *Store) persist() {
	list := make([]string, 0, len(s.ignored))
	for jid := range s.ignored {
		list = append(list, jid)
	}
	_ = s.js.Write(list)
}

func (s *Store) IsIgnored(jid string) bool {
	s.once.Do(s.load)

	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ignored[jid]
}

func (s *Store) Add(jid string) error {
	s.once.Do(s.load)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.ignored[jid] = true
	s.persist()
	return nil
}

func (s *Store) Remove(jid string) (bool, error) {
	s.once.Do(s.load)

	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.ignored[jid] {
		return false, nil
	}
	delete(s.ignored, jid)
	s.persist()
	return true, nil
}

func (s *Store) List() []string {
	s.once.Do(s.load)

	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]string, 0, len(s.ignored))
	for jid := range s.ignored {
		list = append(list, jid)
	}
	return list
}
