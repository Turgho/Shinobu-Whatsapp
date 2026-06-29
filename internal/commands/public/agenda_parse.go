package public

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var relativeTimeRe = regexp.MustCompile(`(\d+)\s*(\w+)`)

// parseRelativeDuration converte "daqui 5 minutos", "em 2 horas" etc. em time.Time.
// Retorna false se não reconhecer o formato.
func parseRelativeDuration(input string, loc *time.Location) (time.Time, bool) {
	s := strings.ToLower(strings.TrimSpace(input))
	now := time.Now().In(loc)

	type unitPattern struct {
		prefixes []string
		unit     time.Duration
	}

	patterns := []unitPattern{
		{[]string{"minuto", "minutos", "min"}, time.Minute},
		{[]string{"hora", "horas", "h"}, time.Hour},
		{[]string{"dia", "dias"}, 24 * time.Hour},
	}

	matches := relativeTimeRe.FindStringSubmatch(s)
	if len(matches) < 3 {
		return time.Time{}, false
	}

	n, err := strconv.Atoi(matches[1])
	if err != nil {
		return time.Time{}, false
	}

	word := matches[2]
	for _, p := range patterns {
		for _, prefix := range p.prefixes {
			if strings.HasPrefix(word, prefix) {
				return now.Add(time.Duration(n) * p.unit).Truncate(time.Second), true
			}
		}
	}

	return time.Time{}, false
}

// parseAgendaTime aceita ISO8601, DD/MM HH:MM, DD/MM/YYYY HH:MM, "D de mês HH:MM",
// "D de mês de YYYY HH:MM", ou tempo relativo ("daqui 5 minutos").
// Tenta 14 layouts em ordem de especificidade. Sem hora → assume 08:00 local.
// Sem ano → assume ano atual; se já passou, avança para o próximo.
func parseAgendaTime(input string, loc *time.Location) (time.Time, error) {
	s := strings.TrimSpace(input)
	s = strings.Trim(s, "`\"'")

	if t, ok := parseRelativeDuration(s, loc); ok {
		return t, nil
	}

	s = normalizePtMonths(s)

	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04",
		"02/01/2006 15:04",
		"02/01 15:04",
		"2/1/2006 15:04",
		"2/1 15:04",
		"2 de January de 2006 15:04",
		"2 de January 15:04",
		"02/01/2006",
		"02/01",
		"2/1/2006",
		"2/1",
		"2 de January de 2006",
		"2 de January",
		"2006-01-02",
	}

	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, loc); err == nil {
			if t.Year() == 0 {
				t = t.AddDate(time.Now().In(loc).Year(), 0, 0)
			}
			if t.Before(time.Now()) {
				t = t.AddDate(1, 0, 0)
			}
			hasTime := strings.Contains(layout, "15:04")
			if !hasTime {
				t = time.Date(t.Year(), t.Month(), t.Day(), 8, 0, 0, 0, loc)
			}
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("formato inválido: %s", input)
}

// normalizePtMonths substitui meses em português para inglês (time.Parse usa nomes EN).
// Ex: "janeiro" → "January", "de" → "".
func normalizePtMonths(s string) string {
	s = strings.ToLower(s)

	repl := map[string]string{
		"janeiro":   "january",
		"fevereiro": "february",
		"março":     "march",
		"abril":     "april",
		"maio":      "may",
		"junho":     "june",
		"julho":     "july",
		"agosto":    "august",
		"setembro":  "september",
		"outubro":   "october",
		"novembro":  "november",
		"dezembro":  "december",
	}

	for pt, en := range repl {
		s = strings.ReplaceAll(s, pt, en)
	}

	return s
}
