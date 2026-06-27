package public

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Turgho/Shinobu-Whatsapp/internal/commands"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/scheduler"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
	"go.uber.org/zap"
)

func AgendaCommand(
	ctx context.Context,
	client *whatsmeow.Client,
	evt *events.Message,
	args []string,
	dynSched *scheduler.Scheduler,
	dynStore *scheduler.DynamicStore,
	logger *zap.Logger,
) error {
	if len(args) < 2 {
		return whatsapp.Reply(ctx, client, evt, "Uso: agenda <data> <mensagem>\nEx: agenda 2026-06-28T09:00 tomar remédio")
	}

	runAt, err := parseAgendaTime(args[0])
	if err != nil {
		return whatsapp.Reply(ctx, client, evt,
			"Data inválida. Exemplos válidos:\n\n"+
				"2026-06-28T09:00\n"+
				"28/06 09:00\n"+
				"5 de janeiro 14:00\n"+
				"28/06 (assume 08:00)")
	}

	now := time.Now()
	if runAt.Before(now) {
		return whatsapp.Reply(ctx, client, evt, "Não posso agendar lembretes no passado!")
	}

	maxAhead := now.AddDate(0, 0, 30)
	if runAt.After(maxAhead) {
		return whatsapp.Reply(ctx, client, evt, "Só posso agendar lembretes com até 30 dias de antecedência.")
	}

	message := strings.Join(args[1:], " ")
	chatJID := evt.Info.Chat.String()
	id := fmt.Sprintf("dyn_%d", time.Now().UnixNano())

	job := scheduler.NewDynamicJob(id, runAt, chatJID, message, client, logger)
	if err := dynStore.Save(job); err != nil {
		logger.Error("Erro ao persistir job dinâmico", zap.Error(err))
		return whatsapp.Reply(ctx, client, evt, "Erro ao salvar lembrete. Tente novamente.")
	}

	dynSched.Register(job)

	reply := fmt.Sprintf("✅ Lembrete agendado!\n\n📅 %s\n🕐 %s\n📝 %s",
		runAt.Format("02/01/2006"),
		runAt.Format("15:04"),
		message,
	)
	return whatsapp.Reply(ctx, client, evt, reply)
}

func AgendaHandler(
	sched *scheduler.Scheduler,
	dynStore *scheduler.DynamicStore,
	logger *zap.Logger,
) commands.HandlerFunc {
	l := logger.Named("AGENDA")
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		return AgendaCommand(ctx, client, evt, args, sched, dynStore, l)
	}
}

func parseRelativeDuration(input string) (time.Time, bool) {
	s := strings.ToLower(strings.TrimSpace(input))
	now := time.Now()

	patterns := []struct {
		prefixes []string
		unit     time.Duration
	}{
		{[]string{"minuto", "minutos", "min"}, time.Minute},
		{[]string{"hora", "horas", "h"}, time.Hour},
		{[]string{"dia", "dias"}, 24 * time.Hour},
	}

	re := regexp.MustCompile(`(\d+)\s*(\w+)`)
	matches := re.FindStringSubmatch(s)
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
				return now.Add(time.Duration(n) * p.unit), true
			}
		}
	}

	return time.Time{}, false
}

func parseAgendaTime(input string) (time.Time, error) {
	s := strings.TrimSpace(input)
	s = strings.Trim(s, "`\"'")

	if t, ok := parseRelativeDuration(s); ok {
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
		if t, err := time.Parse(layout, s); err == nil {
			if t.Year() == 0 {
				t = t.AddDate(time.Now().Year(), 0, 0)
			}
			if t.Before(time.Now()) {
				t = t.AddDate(1, 0, 0)
			}
			hasTime := strings.Contains(layout, "15:04")
			if !hasTime {
				t = time.Date(t.Year(), t.Month(), t.Day(), 8, 0, 0, 0, time.Local)
			}
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("formato inválido: %s", input)
}

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
