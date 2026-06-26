package weekday

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/sticker"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.uber.org/zap"
)

// Config contém as opções do WeekdayJob.
type Config struct {
	Name         string
	Day          string // "sunday", "monday", ..., "saturday"
	Enabled      bool
	Hour         int
	Minute       int
	AudioPath    string
	StickerName  string
	TargetGroups []string
}

// WeekdayJob envia áudio + menção + sticker para grupos em um dia específico.
type WeekdayJob struct {
	client       *whatsmeow.Client
	logger       *zap.Logger
	location     *time.Location
	targetGroups []types.JID
	audioPath    string
	stickerName  string
	name         string
	day          time.Weekday
	hour         int
	minute       int
}

// NewFromConfig cria um WeekdayJob a partir da Config.
// Retorna nil se desabilitado, dia inválido, ou sem grupos válidos.
func NewFromConfig(client *whatsmeow.Client, logger *zap.Logger, cfg Config) *WeekdayJob {
	if !cfg.Enabled || len(cfg.TargetGroups) == 0 {
		return nil
	}

	day, ok := parseDay(cfg.Day)
	if !ok {
		logger.Warn("Dia inválido", zap.String("day", cfg.Day))
		return nil
	}

	groups := make([]types.JID, 0, len(cfg.TargetGroups))
	for _, jidStr := range cfg.TargetGroups {
		jid, err := types.ParseJID(jidStr)
		if err != nil {
			logger.Warn("JID inválido ignorado", zap.String("jid", jidStr), zap.Error(err))
			continue
		}
		groups = append(groups, jid)
	}

	if len(groups) == 0 {
		return nil
	}

	name := cfg.Name
	if name == "" {
		name = cfg.Day
	}

	job, err := newWeekdayJob(client, logger, groups, cfg.AudioPath, cfg.StickerName, name, day, cfg.Hour, cfg.Minute)
	if err != nil {
		logger.Error("Erro ao criar WeekdayJob", zap.Error(err))
		return nil
	}
	return job
}

func newWeekdayJob(client *whatsmeow.Client, logger *zap.Logger, targetGroups []types.JID, audioPath, stickerName, name string, day time.Weekday, hour, minute int) (*WeekdayJob, error) {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		return nil, fmt.Errorf("carregar timezone: %w", err)
	}

	groupStrs := make([]string, len(targetGroups))
	for i, g := range targetGroups {
		groupStrs[i] = g.String()
	}

	logger.Info("Job configurado",
		zap.String("name", name),
		zap.String("day", day.String()),
		zap.Int("hour", hour),
		zap.Int("minute", minute),
		zap.Strings("groups", groupStrs),
		zap.String("audio", audioPath),
		zap.String("sticker", stickerName),
	)

	return &WeekdayJob{
		client:       client,
		logger:       logger.Named(strings.ToUpper(name)),
		location:     loc,
		targetGroups: targetGroups,
		audioPath:    audioPath,
		stickerName:  stickerName,
		name:         name,
		day:          day,
		hour:         hour,
		minute:       minute,
	}, nil
}

// parseDay converte string nome do dia para time.Weekday.
func parseDay(s string) (time.Weekday, bool) {
	switch strings.ToLower(s) {
	case "sunday":
		return time.Sunday, true
	case "monday":
		return time.Monday, true
	case "tuesday":
		return time.Tuesday, true
	case "wednesday":
		return time.Wednesday, true
	case "thursday":
		return time.Thursday, true
	case "friday":
		return time.Friday, true
	case "saturday":
		return time.Saturday, true
	}
	return 0, false
}

func (j *WeekdayJob) Name() string {
	return j.name
}

func (j *WeekdayJob) Next(now time.Time) time.Time {
	now = now.In(j.location)
	next := time.Date(now.Year(), now.Month(), now.Day(), j.hour, j.minute, 0, 0, j.location)
	for next.Weekday() != j.day || next.Before(now) {
		next = next.Add(24 * time.Hour)
	}
	return next
}

func (j *WeekdayJob) Run(ctx context.Context) error {
	for _, groupJID := range j.targetGroups {
		// 1. Envia áudio como PTT
		func() {
			sendCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			if err := whatsapp.SendAudioFileToJID(sendCtx, j.client, groupJID, j.audioPath); err != nil {
				j.logger.Error("Erro ao enviar áudio",
					zap.String("group", groupJID.String()),
					zap.Error(err),
				)
			}
		}()

		// 2. Envia texto com @all nativo
		func() {
			sendCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			msg := fmt.Sprintf("🎉 %s! @all", j.day.String())
			if err := whatsapp.SendAllToJID(sendCtx, j.client, groupJID, msg); err != nil {
				j.logger.Error("Erro ao enviar menção @all",
					zap.String("group", groupJID.String()),
					zap.Error(err),
				)
			}
		}()

		// 3. Envia sticker se configurado
		if j.stickerName != "" {
			func() {
				sendCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
				defer cancel()
				d, ok := sticker.Get(j.stickerName)
				if !ok {
					j.logger.Warn("Sticker não encontrado",
						zap.String("sticker", j.stickerName),
					)
					return
				}
				uploaded := &whatsmeow.UploadResponse{
					URL:           d.URL,
					DirectPath:    d.DirectPath,
					MediaKey:      d.MediaKey,
					FileEncSHA256: d.FileEncSHA256,
					FileSHA256:    d.FileSHA256,
					FileLength:    d.FileLength,
				}
				if err := whatsapp.SendStickerToJID(sendCtx, j.client, groupJID, uploaded, d.IsAnimated); err != nil {
					j.logger.Error("Erro ao enviar sticker",
						zap.String("group", groupJID.String()),
						zap.Error(err),
					)
				}
			}()
		}
	}
	return nil
}
