package birthday

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/store"
)

const storeFile = "assets/info/birthdays.json"

type Entry struct {
	JID   string `json:"jid"`
	Name  string `json:"name"`
	Day   int    `json:"day"`
	Month int    `json:"month"`
}

type Store map[string][]Entry

var js = store.NewJSONStore[Store](storeFile)

func Set(groupJID, userJID, name string, day, month int) error {
	return js.Update(func(s Store) (Store, error) {
		if s == nil {
			s = make(Store)
		}
		entries := s[groupJID]

		for i, e := range entries {
			if e.JID == userJID {
				entries[i] = Entry{JID: userJID, Name: name, Day: day, Month: month}
				s[groupJID] = entries
				return s, nil
			}
		}

		s[groupJID] = append(entries, Entry{JID: userJID, Name: name, Day: day, Month: month})
		return s, nil
	})
}

func Remove(groupJID, userJID string) (bool, error) {
	var removed bool
	err := js.Update(func(s Store) (Store, error) {
		if s == nil {
			return s, nil
		}
		entries := s[groupJID]
		for i, e := range entries {
			if e.JID == userJID {
				s[groupJID] = append(entries[:i], entries[i+1:]...)
				removed = true
				return s, nil
			}
		}
		return s, nil
	})
	return removed, err
}

func TodayEntries(day, month int) map[string][]Entry {
	s, err := js.Read()
	if err != nil || s == nil {
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

func ListGroup(groupJID string) []Entry {
	s, err := js.Read()
	if err != nil || s == nil {
		return nil
	}
	return s[groupJID]
}

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
