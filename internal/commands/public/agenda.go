package public

import (
	"context"
	"fmt"
	"regexp"
	"sort"
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
	if len(args) == 0 {
		return whatsapp.Reply(ctx, client, evt,
			"📋 *Agenda*\n\n"+
				"`agenda <data> <mensagem>` — agendar lembrete\n"+
				"`agenda lista` — listar lembretes\n"+
				"`agenda remover <n>` — remover lembrete",
		)
	}

	switch strings.ToLower(args[0]) {
	case "lista", "list":
		return agendaLista(ctx, client, evt, dynStore)
	case "remover", "remove":
		return agendaRemover(ctx, client, evt, args, dynStore, dynSched, logger)
	}

	runAt, err := parseAgendaTime(args[0])
	if err != nil {
		return whatsapp.Reply(ctx, client, evt,
			"Data inválida. Exemplos válidos:\n\n"+
				"2026-06-28T09:00\n"+
				"28/06 09:00\n"+
				"5 de janeiro 14:00\n"+
				"daqui 5 minutos\n"+
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

func agendaLista(
	ctx context.Context,
	client *whatsmeow.Client,
	evt *events.Message,
	dynStore *scheduler.DynamicStore,
) error {
	all := dynStore.LoadAll()
	if len(all) == 0 {
		return whatsapp.Reply(ctx, client, evt, "Nenhum lembrete agendado.")
	}

	now := time.Now()
	var future []scheduler.DataWithTime
	for _, d := range all {
		t, err := time.Parse(time.RFC3339, d.RunAt)
		if err != nil || t.Before(now) {
			continue
		}
		future = append(future, scheduler.DataWithTime{Data: d, ParsedAt: t})
	}

	if len(future) == 0 {
		return whatsapp.Reply(ctx, client, evt, "Nenhum lembrete agendado.")
	}

	sort.Slice(future, func(i, j int) bool {
		return future[i].ParsedAt.Before(future[j].ParsedAt)
	})

	var b strings.Builder
	b.WriteString("📋 *Lembretes agendados:*\n\n")
	for i, f := range future {
		b.WriteString(fmt.Sprintf("%d. 📅 %s — %s\n",
			i+1,
			f.ParsedAt.Format("02/01 15:04"),
			f.Data.Message,
		))
	}

	return whatsapp.Reply(ctx, client, evt, b.String())
}

func agendaRemover(
	ctx context.Context,
	client *whatsmeow.Client,
	evt *events.Message,
	args []string,
	dynStore *scheduler.DynamicStore,
	dynSched *scheduler.Scheduler,
	logger *zap.Logger,
) error {
	if len(args) < 2 {
		return whatsapp.Reply(ctx, client, evt, "Use: `agenda remover <número>`")
	}

	idx, err := strconv.Atoi(args[1])
	if err != nil || idx < 1 {
		return whatsapp.Reply(ctx, client, evt, "Número inválido.")
	}

	all := dynStore.LoadAll()
	now := time.Now()
	var future []scheduler.DataWithTime
	for _, d := range all {
		t, err := time.Parse(time.RFC3339, d.RunAt)
		if err != nil || t.Before(now) {
			continue
		}
		future = append(future, scheduler.DataWithTime{Data: d, ParsedAt: t})
	}

	sort.Slice(future, func(i, j int) bool {
		return future[i].ParsedAt.Before(future[j].ParsedAt)
	})

	if idx > len(future) {
		return whatsapp.Reply(ctx, client, evt, "Número inválido.")
	}

	target := future[idx-1]
	if err := dynStore.Delete(target.Data.ID); err != nil {
		logger.Error("Erro ao deletar job", zap.Error(err))
		return whatsapp.Reply(ctx, client, evt, "Erro ao remover lembrete.")
	}

	dynSched.Unregister(target.Data.ID)

	return whatsapp.Reply(ctx, client, evt,
		fmt.Sprintf("🗑️ Lembrete removido: %s", target.Data.Message))
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

	type unitPattern struct {
		prefixes []string
		unit     time.Duration
	}

	patterns := []unitPattern{
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
