package ignore

import (
	"strings"
	"sync"

	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/phone"
	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/store"
)

const storeFile = "assets/info/ignored.json"

// Store gerencia a lista de contatos ignorados, com persistência em JSON.
//
// A chave interna é o **número normalizado** (só dígitos): assim o mesmo
// contato bloqueia tanto pelo número quanto pelo JID — antes, o !ignorar
// gravava o número e a checagem procurava o JID, então ninguém era
// bloqueado de fato.
type Store struct {
	js      *store.JSONStore[[]string]
	mu      sync.RWMutex
	ignored map[string]string // chave normalizada → valor original (exibição)
	once    sync.Once
}

func NewStore() *Store { return NewStoreAt(storeFile) }

// NewStoreAt cria o store com um caminho explícito (usado nos testes para
// não escrever no arquivo real).
func NewStoreAt(path string) *Store {
	return &Store{
		js: store.NewJSONStore[[]string](path),
	}
}

// key normaliza um identificador (número ou JID) para a chave de busca.
func key(id string) string {
	if digits := phone.NormalizeNumber(id); digits != "" {
		return digits
	}
	return strings.ToLower(strings.TrimSpace(id))
}

func (s *Store) load() {
	list, err := s.js.Read()
	if err != nil || list == nil {
		s.ignored = make(map[string]string)
		return
	}
	s.ignored = make(map[string]string, len(list))
	for _, jid := range list {
		s.ignored[key(jid)] = jid
	}
}

func (s *Store) persist() error {
	list := make([]string, 0, len(s.ignored))
	for _, jid := range s.ignored {
		list = append(list, jid)
	}
	return s.js.Write(list)
}

// IsIgnored informa se o identificador (número ou JID) está ignorado.
func (s *Store) IsIgnored(jid string) bool {
	s.once.Do(s.load)

	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.ignored[key(jid)]
	return ok
}

// Add ignora um contato. Aceita número ou JID.
func (s *Store) Add(jid string) error {
	s.once.Do(s.load)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.ignored[key(jid)] = jid
	return s.persist()
}

// Remove deixa de ignorar um contato. Aceita número ou JID.
func (s *Store) Remove(jid string) (bool, error) {
	s.once.Do(s.load)

	s.mu.Lock()
	defer s.mu.Unlock()
	k := key(jid)
	if _, ok := s.ignored[k]; !ok {
		return false, nil
	}
	delete(s.ignored, k)
	if err := s.persist(); err != nil {
		return true, err
	}
	return true, nil
}

// List devolve os valores originais (como foram cadastrados).
func (s *Store) List() []string {
	s.once.Do(s.load)

	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]string, 0, len(s.ignored))
	for _, jid := range s.ignored {
		list = append(list, jid)
	}
	return list
}
