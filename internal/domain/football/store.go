package football

import (
	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/store"
)

const storeFile = "assets/info/football_events.json"

// Store maps matchID to the last processed event ID.
type Store map[int]int

var js = store.NewJSONStore[Store](storeFile)

// SetLastEventID sets the last processed event ID for a match.
func SetLastEventID(matchID, eventID int) error {
	return js.Update(func(s Store) (Store, error) {
		if s == nil {
			s = make(Store)
		}
		s[matchID] = eventID
		return s, nil
	})
}

// GetLastEventID returns the last processed event ID for a match.
func GetLastEventID(matchID int) (int, error) {
	s, err := js.Read()
	if err != nil {
		return 0, err
	}
	if s == nil {
		return 0, nil
	}
	return s[matchID], nil
}