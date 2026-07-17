// Package app orquestra a inicialização de todas as dependências do bot:
// config, logger, banco, stores, scheduler e router de comandos.
package app

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os/exec"
	"time"

	"github.com/Turgho/Shinobu-Whatsapp/internal/bot"
	"github.com/Turgho/Shinobu-Whatsapp/internal/commands"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/birthday"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/history"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/ia"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/ignore"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/mikael"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/scheduler"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/sticker"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/weekday"
	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/configs"
	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/database"
	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/uptime"
	"go.mau.fi/whatsmeow"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func Run() error {
	uptime.Start()

	if err := checkDeps(); err != nil {
		return fmt.Errorf("dependências ausentes: %w", err)
	}

	cfg := configs.Load()

	logger, err := buildLogger(cfg)
	if err != nil {
		return fmt.Errorf("erro ao inicializar logger: %w", err)
	}
	defer logger.Sync()

	ctx := context.Background()

	db, err := connectDatabase(cfg, logger)
	if err != nil {
		return err
	}
	defer db.Close()

	store, err := history.NewStore("storage/message_history.db", logger.Named("HISTORY"))
	if err != nil {
		logger.Error("erro ao abrir history", zap.Error(err))
		return fmt.Errorf("erro ao abrir history: %w", err)
	}
	defer store.Close()

	store.StartCleanup(ctx, 24*time.Hour)

	mikaelStore := mikael.NewStore(store, store.DB(), cfg.Mikael.Groups, cfg.Mikael.LID, logger.Named("MIKAEL"))

	client, err := bot.NewClient(ctx, db, logger.Named("CLIENT"))
	if err != nil {
		return fmt.Errorf("erro ao criar client WhatsApp: %w", err)
	}

	if err := client.Connect(ctx); err != nil {
		return fmt.Errorf("erro ao conectar no WhatsApp: %w", err)
	}

	ignoreStore := ignore.NewStore()
	stickerStore := sticker.NewStore()
	r := buildRouter(cfg, client.WAClient, logger, store, ignoreStore)
	r.AddMessageHook(mikaelStore.SaveMessageHook())

	r.SetAIConfig(&ia.Config{
		GroqURL:    cfg.Groq.URL,
		GroqKey:    cfg.Groq.APIKey,
		TavilyKey:  cfg.Tavily.APIKey,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
		Log:        logger.Named("AI"),
	})

	r.StartRateLimitCleanup(ctx)

	// Inicializa scheduler e store de jobs dinâmicos
	dynLogger := logger.Named("DYNAMIC")
	dynStore := scheduler.NewDynamicStore("storage/dynamic_jobs.json", dynLogger)
	sched := scheduler.NewScheduler(logger.Named("SCHEDULER"))

	registerPublicCommands(r, cfg, logger, store, sched, dynStore, stickerStore, mikaelStore)
	registerAdminCommands(r, cfg, store, logger, ignoreStore, stickerStore, client.WAClient, sched)
	registerAliases(r)

	// Carrega jobs dinâmicos persistidos, ignorando expirados
	now := time.Now()
	for _, data := range dynStore.LoadAll() {
		runAt, err := time.Parse(time.RFC3339, data.RunAt)
		if err != nil || runAt.Before(now) {
			dynLogger.Info("Job expirado ou inválido, removendo",
				zap.String("id", data.ID),
				zap.String("run_at", data.RunAt),
				zap.Error(err),
			)
			dynStore.Delete(data.ID)
			continue
		}

		job := scheduler.NewDynamicJob(data.ID, runAt, data.ChatJID, data.Message, data.MentionAll, client.WAClient, dynLogger)
		sched.Register(job)
		dynLogger.Info("Job dinâmico restaurado",
			zap.String("id", data.ID),
			zap.Time("run_at", runAt),
		)
	}

	loc, errLoc := time.LoadLocation(cfg.Bot.Timezone)
	if errLoc != nil {
		logger.Warn("timezone inválida, usando Local", zap.String("timezone", cfg.Bot.Timezone), zap.Error(errLoc))
		loc = time.Local
	}

	birthdayJob := birthday.NewBirthdayJob(client.WAClient, logger.Named("BIRTHDAY"), loc)
	sched.Register(birthdayJob)

	for _, j := range cfg.ScheduledJobs {
		if job := weekday.NewFromConfig(client.WAClient, logger.Named(j.Name), weekday.Config{
			Name:         j.Name,
			Day:          j.Day,
			Enabled:      j.Enabled,
			Hour:         j.Hour,
			Minute:       j.Minute,
			Message:      j.Message,
			AudioPath:    j.AudioPath,
			StickerName:  j.StickerName,
			TargetGroups: j.TargetGroups,
			Location:     loc,
		}, stickerStore); job != nil {
			sched.Register(job)
		}
	}

	sched.Start(ctx)

	handler := bot.NewHandler(client.WAClient, r, logger.Named("HANDLER"))
	client.RegisterHandlers(handler.EventHandler)

	client.Listen()
	return nil
}

func buildLogger(cfg *configs.Config) (*zap.Logger, error) {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		loc = time.UTC
	}

	logCfg := zap.NewDevelopmentConfig()
	logCfg.DisableStacktrace = true
	logCfg.EncoderConfig.TimeKey = "time"
	logCfg.EncoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.In(loc).Format("2006-01-02 15:04:05"))
	}

	if cfg.Log.Level != "" {
		lvl := zap.NewAtomicLevel()
		if err := lvl.UnmarshalText([]byte(cfg.Log.Level)); err == nil {
			logCfg.Level = lvl
		}
	}

	return logCfg.Build()
}

func connectDatabase(cfg *configs.Config, logger *zap.Logger) (*sql.DB, error) {
	conn := database.NewDatabase(cfg.Database.Driver, cfg.Database.Dsn, logger.Named("DATABASE"))
	db, err := conn.Connect()
	if err != nil {
		return nil, fmt.Errorf("erro ao conectar no banco: %w", err)
	}
	return db, nil
}

func buildRouter(cfg *configs.Config, waClient *whatsmeow.Client, logger *zap.Logger, store *history.Store, ignoreStore *ignore.Store) *commands.Router {
	r := commands.NewRouter(cfg.Bot.Prefix, waClient, logger.Named("ROUTER"), store, ignoreStore)

	r.Use(commands.IgnoreSelfMiddleware)
	r.Use(commands.IgnoreOldMessagesMiddleware)
	r.Use(commands.CommandNotFoundMiddleware(r))
	r.Use(commands.PrivateCommandsMiddleware(r, cfg.UsersJID.Owner, cfg.UsersJID.Admins))
	r.SetNLPGroupTrigger(cfg.Bot.NLPGroupTrigger)
	r.SetIntentEnabled(cfg.Bot.IntentEnabled)

	return r
}

func checkDeps() error {
	deps := []struct {
		path string
		name string
	}{
		{"./bin/ffmpeg", "ffmpeg"},
		{"./bin/webpmux", "webpmux"},
	}
	for _, d := range deps {
		if _, err := exec.LookPath(d.path); err != nil {
			return fmt.Errorf("%s não encontrado em %s (execute a partir da raiz do projeto ou ajuste o path)", d.name, d.path)
		}
	}
	return nil
}
