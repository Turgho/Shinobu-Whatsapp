package birthday

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.uber.org/zap"
)

const notifyHour = 8

// BirthdayJob implementa scheduler.Job para notificações de aniversário.
type BirthdayJob struct {
	client   *whatsmeow.Client
	logger   *zap.Logger
	location *time.Location
}

// NewBirthdayJob cria um novo job de aniversário.
func NewBirthdayJob(client *whatsmeow.Client, logger *zap.Logger, loc *time.Location) *BirthdayJob {
	return &BirthdayJob{
		client:   client,
		logger:   logger.Named("BIRTHDAY"),
		location: loc,
	}
}

// Name retorna o nome do job.
func (j *BirthdayJob) Name() string {
	return "birthday"
}

// Next retorna o próximo horário de execução (próximas 8h no timezone configurado).
func (j *BirthdayJob) Next(now time.Time) time.Time {
	now = now.In(j.location)
	next := time.Date(now.Year(), now.Month(), now.Day(), notifyHour, 0, 0, 0, j.location)
	if now.After(next) {
		next = next.Add(24 * time.Hour)
	}
	return next
}

// Run executa a verificação e notificação de aniversários.
func (j *BirthdayJob) Run(ctx context.Context) error {
	now := time.Now().In(j.location)
	birthdays := TodayEntries(now.Day(), int(now.Month()))

	if len(birthdays) == 0 {
		j.logger.Debug("Nenhum aniversariante hoje")
		return nil
	}

	for groupJID, entries := range birthdays {
		jid, err := types.ParseJID(groupJID)
		if err != nil {
			j.logger.Error("JID inválido", zap.String("jid", groupJID), zap.Error(err))
			continue
		}

		mentions, msg := buildMessage(entries)

		sendCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		err = whatsapp.SendTextToJID(sendCtx, j.client, jid, msg, mentions)
		cancel()

		if err != nil {
			j.logger.Error("Erro ao enviar notificação de aniversário",
				zap.String("group", groupJID),
				zap.Error(err),
			)
			continue
		}

		j.logger.Info("Notificação de aniversário enviada",
			zap.String("group", groupJID),
			zap.Int("count", len(entries)),
		)
	}

	return nil
}

func buildMessage(entries []Entry) (mentions []string, msg string) {
	var sb strings.Builder
	sb.WriteString("🎂 *Parabéns!* 🎉\n\n")

	for _, e := range entries {
		phone := phoneFromJID(e.JID)
		fmt.Fprintf(&sb, "🎈 @%s faz aniversário hoje!\n", phone)
		mentions = append(mentions, e.JID)
	}

	sb.WriteString("\n@all venha parabenizar! 🥳")
	mentions = append(mentions, "all@broadcast")

	return mentions, sb.String()
}

// phoneFromJID extrai o número de telefone de um JID (ex: "5511999999999@s.whatsapp.net" → "5511999999999").
func phoneFromJID(jid string) string {
	if idx := strings.Index(jid, "@"); idx > 0 {
		return jid[:idx]
	}
	return jid
}
