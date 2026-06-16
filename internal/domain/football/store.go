package football

import (
	"strconv"

	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/store"
)

const storeFile = "assets/info/football_events.json"

// Store mapeia matchID (como string) -> último eventID processado.
type Store map[string]int

var js = store.NewJSONStore[Store](storeFile)

// SetLastEventID salva o último eventID processado para a partida.
// Usado para deduplicação: evita notificar o mesmo gol duas vezes (mesmo após restart).
func SetLastEventID(matchID, eventID int) error {
	return js.Update(func(s Store) (Store, error) {
		if s == nil {
			s = make(Store)
		}
		s[strconv.Itoa(matchID)] = eventID
		return s, nil
	})
}

// GetLastEventID lê o último eventID processado da partida.
// Retorna 0 se nenhum evento foi processado ainda para essa partida.
func GetLastEventID(matchID int) (int, error) {
	s, err := js.Read()
	if err != nil {
		return 0, err
	}
	if s == nil {
		return 0, nil
	}
	return s[strconv.Itoa(matchID)], nil
}
