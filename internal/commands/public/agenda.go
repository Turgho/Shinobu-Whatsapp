// Package public contém handlers de comandos públicos do bot.
package public

import (
	"context"
	"fmt"
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

// AgendaCommand agenda lembretes com data/hora natural em PT-BR.
// Subcomandos: lista (jobs futuros), remover <n>, ou <data> <mensagem>.
func AgendaCommand(
	ctx context.Context,
	client *whatsmeow.Client,
	evt *events.Message,
	args []string,
	dynSched *scheduler.Scheduler,
	dynStore *scheduler.DynamicStore,
	logger *zap.Logger,
	loc *time.Location,
) error {
	if len(args) == 0 {
		return whatsapp.Reply(ctx, client, evt,
			"📋 *Agenda*\n\n"+
				"`agenda <data> <mensagem>` — agendar lembrete\n"+
				"`agenda <data> todos <mensagem>` — agendar com @all\n"+
				"`agenda lista` — listar lembretes\n"+
				"`agenda remover <n>` — remover lembrete",
		)
	}

	switch strings.ToLower(args[0]) {
	case "lista", "list":
		return agendaLista(ctx, client, evt, dynStore, loc)
	case "remover", "remove":
		return agendaRemover(ctx, client, evt, args, dynStore, dynSched, logger, loc)
	}

	runAt, err := parseAgendaTime(args[0], loc)
	if err != nil {
		return whatsapp.Reply(ctx, client, evt, msgInvalidDate)
	}

	now := time.Now().In(loc)
	if runAt.Before(now) {
		return whatsapp.Reply(ctx, client, evt, msgPastDate)
	}

	maxAhead := now.AddDate(0, 0, 30)
	if runAt.After(maxAhead) {
		return whatsapp.Reply(ctx, client, evt, msgDateLimit)
	}

	// Detecta flag @all: "@all" (NLU) ou "todos" (digitação manual) na mensagem
	rawMsg := strings.Join(args[1:], " ")
	mentionAll := false
	for _, marker := range []string{"@all", "todos"} {
		if idx := strings.Index(strings.ToLower(rawMsg), marker); idx >= 0 {
			mentionAll = true
			before := rawMsg[:idx]
			after := rawMsg[idx+len(marker):]
			rawMsg = strings.TrimSpace(before + " " + after)
			rawMsg = trimLeadingPreps(rawMsg)
			break
		}
	}
	if rawMsg == "" {
		return whatsapp.Reply(ctx, client, evt, msgEmptyReminder)
	}

	chatJID := evt.Info.Chat.String()
	id := fmt.Sprintf("dyn_%d", time.Now().UnixNano())

	job := scheduler.NewDynamicJob(id, runAt, chatJID, rawMsg, mentionAll, client, logger)
	if err := dynStore.Save(job); err != nil {
		logger.Error("Erro ao persistir job dinâmico", zap.Error(err))
		return whatsapp.Reply(ctx, client, evt, msgSaveReminderFail)
	}

	dynSched.Register(job)

	// Força uma verificação imediata para jobs de curta duração
	// (ex: "daqui 1 minuto"), reduzindo o delay máximo para 1 tick (15s).
	dynSched.RunCheck()

	allTag := ""
	if mentionAll {
		allTag = "\n📢 @all"
	}

	reply := fmt.Sprintf("✅ Lembrete agendado!\n\n📅 %s\n🕐 %s\n📝 %s%s",
		runAt.Format("02/01/2006"),
		runAt.Format("15:04"),
		rawMsg,
		allTag,
	)
	return whatsapp.Reply(ctx, client, evt, reply)
}

// agendaLista exibe todos os lembretes futuros ordenados por data.
func agendaLista(
	ctx context.Context,
	client *whatsmeow.Client,
	evt *events.Message,
	dynStore *scheduler.DynamicStore,
	loc *time.Location,
) error {
	all := dynStore.LoadAll()
	if len(all) == 0 {
		return whatsapp.Reply(ctx, client, evt, msgNoReminders)
	}

	now := time.Now().In(loc)
	var future []scheduler.DataWithTime
	for _, d := range all {
		t, err := time.Parse(time.RFC3339, d.RunAt)
		if err != nil || t.Before(now) {
			continue
		}
		future = append(future, scheduler.DataWithTime{Data: d, ParsedAt: t})
	}

	if len(future) == 0 {
		return whatsapp.Reply(ctx, client, evt, msgNoReminders)
	}

	sort.Slice(future, func(i, j int) bool {
		return future[i].ParsedAt.Before(future[j].ParsedAt)
	})

	var b strings.Builder
	b.WriteString("📋 *Lembretes agendados:*\n\n")
	for i, f := range future {
		tag := ""
		if f.Data.MentionAll {
			tag = " 📢@all"
		}
		fmt.Fprintf(&b, "%d. 📅 %s — %s%s\n",
			i+1,
			f.ParsedAt.Format("02/01 15:04"),
			f.Data.Message,
			tag,
		)
	}

	return whatsapp.Reply(ctx, client, evt, b.String())
}

// agendaRemover remove um lembrete pelo índice mostrado no lista.
func agendaRemover(
	ctx context.Context,
	client *whatsmeow.Client,
	evt *events.Message,
	args []string,
	dynStore *scheduler.DynamicStore,
	dynSched *scheduler.Scheduler,
	logger *zap.Logger,
	loc *time.Location,
) error {
	if len(args) < 2 {
		return whatsapp.Reply(ctx, client, evt, msgReminderUsage)
	}

	idx, err := strconv.Atoi(args[1])
	if err != nil || idx < 1 {
		return whatsapp.Reply(ctx, client, evt, msgInvalidNumber)
	}

	all := dynStore.LoadAll()
	now := time.Now().In(loc)
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
		return whatsapp.Reply(ctx, client, evt, msgInvalidNumber)
	}

	target := future[idx-1]
	if err := dynStore.Delete(target.Data.ID); err != nil {
		logger.Error("Erro ao deletar job", zap.Error(err))
		return whatsapp.Reply(ctx, client, evt, msgRemoveReminderFail)
	}

	dynSched.Unregister(target.Data.ID)

	return whatsapp.Reply(ctx, client, evt,
		fmt.Sprintf("🗑️ Lembrete removido: %s", target.Data.Message))
}

func AgendaHandler(
	sched *scheduler.Scheduler,
	dynStore *scheduler.DynamicStore,
	logger *zap.Logger,
	loc *time.Location,
) commands.HandlerFunc {
	l := logger.Named("AGENDA")
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		return AgendaCommand(ctx, client, evt, args, sched, dynStore, l, loc)
	}
}

func trimLeadingPreps(s string) string {
	preps := []string{"do", "da", "de", "o", "a", "os", "das", "dos", "um", "uma"}
	for {
		trimmed := false
		fields := strings.Fields(s)
		if len(fields) == 0 {
			break
		}
		for _, p := range preps {
			if strings.EqualFold(fields[0], p) {
				s = strings.Join(fields[1:], " ")
				trimmed = true
				break
			}
		}
		if !trimmed {
			break
		}
	}
	return s
}
