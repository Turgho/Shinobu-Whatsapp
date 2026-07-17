package admin

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/scheduler"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
	"go.uber.org/zap"
)

func JobsCommand(
	ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string,
	sched *scheduler.Scheduler, logger *zap.Logger,
) error {
	sub := ""
	if len(args) > 0 {
		sub = strings.ToLower(strings.TrimSpace(args[0]))
	}

	switch sub {
	case "forcar", "force":
		return forceJob(ctx, client, evt, args, sched, logger)
	default:
		return listJobs(ctx, client, evt, sched)
	}
}

func listJobs(
	ctx context.Context, client *whatsmeow.Client, evt *events.Message,
	sched *scheduler.Scheduler,
) error {
	jobs := sched.ListJobs()
	if len(jobs) == 0 {
		return whatsapp.Reply(ctx, client, evt, "Nenhum job registrado no scheduler.")
	}

	var sb strings.Builder
	sb.WriteString("*Jobs do Scheduler:*\n\n")
	for _, j := range jobs {
		runAt := j.NextRun.In(time.Local).Format("02/01 15:04 Mon")
		sb.WriteString(fmt.Sprintf("• *%s* → %s\n", j.Name, runAt))
	}
	sb.WriteString("\nPara forçar: `!jobs forcar <nome>`")

	return whatsapp.Reply(ctx, client, evt, sb.String())
}

func forceJob(
	ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string,
	sched *scheduler.Scheduler, logger *zap.Logger,
) error {
	if len(args) < 2 {
		return whatsapp.Reply(ctx, client, evt, "Uso: `!jobs forcar <nome>`\nUse `!jobs` para listar os disponíveis.")
	}

	name := strings.ToLower(strings.TrimSpace(args[1]))

	// Verifica se existe antes de tentar forçar
	found := false
	for _, j := range sched.ListJobs() {
		if j.Name == name {
			found = true
			break
		}
	}
	if !found {
		return whatsapp.Reply(ctx, client, evt, fmt.Sprintf("Job *%q* não encontrado. Use `!jobs` para listar.", name))
	}

	logger.Info("Job forçado via comando admin",
		zap.String("job", name),
		zap.String("sender", evt.Info.Sender.User),
	)

	if err := sched.ForceRun(ctx, name); err != nil {
		logger.Error("Erro ao forçar job", zap.String("job", name), zap.Error(err))
		return whatsapp.Reply(ctx, client, evt, fmt.Sprintf("❌ Erro ao executar *%s*: %v", name, err))
	}

	return whatsapp.Reply(ctx, client, evt, fmt.Sprintf("✅ Job *%s* executado com sucesso!", name))
}

func JobsHandler(sched *scheduler.Scheduler, logger *zap.Logger) func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
	l := logger.Named("JOBS")
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		return JobsCommand(ctx, client, evt, args, sched, l)
	}
}
