package scheduler

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.uber.org/zap"
)

type DynamicJob struct {
	ID      string
	RunAt   time.Time
	ChatJID string
	Message string

	client *whatsmeow.Client
	logger *zap.Logger
	done   atomic.Bool
}

func NewDynamicJob(id string, runAt time.Time, chatJID, message string, client *whatsmeow.Client, logger *zap.Logger) *DynamicJob {
	return &DynamicJob{
		ID:      id,
		RunAt:   runAt,
		ChatJID: chatJID,
		Message: message,
		client:  client,
		logger:  logger,
	}
}

func (j *DynamicJob) Name() string { return j.ID }

func (j *DynamicJob) Next(now time.Time) time.Time {
	if j.done.Load() {
		return time.Time{}
	}
	return j.RunAt
}

func (j *DynamicJob) Run(ctx context.Context) error {
	j.done.Store(true)

	j.logger.Info("Executando job dinâmico",
		zap.String("job_id", j.ID),
		zap.String("chat", j.ChatJID),
	)

	jid, err := types.ParseJID(j.ChatJID)
	if err != nil {
		return fmt.Errorf("dynamic_job: JID inválido %s: %w", j.ChatJID, err)
	}

	sendCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	err = whatsapp.SendTextToJID(sendCtx, j.client, jid, j.Message, nil)
	if err != nil {
		return fmt.Errorf("dynamic_job: enviar mensagem: %w", err)
	}

	j.logger.Info("Lembrete enviado",
		zap.String("job_id", j.ID),
		zap.String("chat", j.ChatJID),
		zap.String("message", j.Message),
	)
	return nil
}
