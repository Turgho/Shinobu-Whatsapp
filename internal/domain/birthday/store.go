package birthday

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
)

const storeFile = "assets/info/birthdays.json"

// Entry representa o aniversário de uma pessoa em um grupo.
type Entry struct {
	JID   string `json:"jid"`  // ex: 5511999999999@s.whatsapp.net
	Name  string `json:"name"` // nome de exibição
	Day   int    `json:"day"`
	Month int    `json:"month"`
}

// Store é o mapa de groupJID → lista de aniversários.
type Store map[string][]Entry

var mu sync.RWMutex

// load lê o JSON do disco.
func load() (Store, error) {
	s := make(Store)
	data, err := os.ReadFile(storeFile)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("birthday: ler arquivo: %w", err)
	}
	if len(data) == 0 {
		return s, nil
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("birthday: parse json: %w", err)
	}
	return s, nil
}

// save persiste o Store no disco.
func save(s Store) error {
	if err := os.MkdirAll("assets", 0755); err != nil {
		return fmt.Errorf("birthday: mkdir: %w", err)
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("birthday: marshal: %w", err)
	}
	return os.WriteFile(storeFile, data, 0644)
}

// Set salva ou atualiza o aniversário de um usuário em um grupo.
func Set(groupJID, userJID, name string, day, month int) error {
	mu.Lock()
	defer mu.Unlock()

	s, err := load()
	if err != nil {
		return err
	}

	entries := s[groupJID]

	// Atualiza se já existir, senão adiciona.
	for i, e := range entries {
		if e.JID == userJID {
			entries[i] = Entry{JID: userJID, Name: name, Day: day, Month: month}
			s[groupJID] = entries
			return save(s)
		}
	}

	s[groupJID] = append(entries, Entry{JID: userJID, Name: name, Day: day, Month: month})
	return save(s)
}

// Remove apaga o aniversário de um usuário em um grupo.
func Remove(groupJID, userJID string) (bool, error) {
	mu.Lock()
	defer mu.Unlock()

	s, err := load()
	if err != nil {
		return false, err
	}

	entries := s[groupJID]
	for i, e := range entries {
		if e.JID == userJID {
			s[groupJID] = append(entries[:i], entries[i+1:]...)
			return true, save(s)
		}
	}
	return false, nil
}

// TodayEntries retorna todos os aniversariantes do dia (dia e mês) por grupo.
func TodayEntries(day, month int) map[string][]Entry {
	mu.RLock()
	defer mu.RUnlock()

	s, err := load()
	if err != nil {
		return nil
	}

	result := make(map[string][]Entry)
	for groupJID, entries := range s {
		for _, e := range entries {
			if e.Day == day && e.Month == month {
				result[groupJID] = append(result[groupJID], e)
			}
		}
	}
	return result
}

// ListGroup retorna todos os aniversários de um grupo.
func ListGroup(groupJID string) []Entry {
	mu.RLock()
	defer mu.RUnlock()

	s, err := load()
	if err != nil {
		return nil
	}
	return s[groupJID]
}

// ─── Helpers compartilhados ────────────────────────────────────────────────────

// NormalizeNumber remove todos os caracteres não numéricos de um telefone.
func NormalizeNumber(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// parseDate converte "DD/MM" em day e month inteiros.
func parseDate(s string) (day, month int, err error) {
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("formato inválido")
	}
	day, err = strconv.Atoi(parts[0])
	if err != nil || day < 1 || day > 31 {
		return 0, 0, fmt.Errorf("dia inválido")
	}
	month, err = strconv.Atoi(parts[1])
	if err != nil || month < 1 || month > 12 {
		return 0, 0, fmt.Errorf("mês inválido")
	}
	return day, month, nil
}
