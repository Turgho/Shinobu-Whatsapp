package app

import (
	"context"
	"database/sql"
	"fmt"
	"os/exec"
	"time"

	"github.com/Turgho/Shinobu-Whatsapp/internal/bot"
	"github.com/Turgho/Shinobu-Whatsapp/internal/commands"
	"github.com/Turgho/Shinobu-Whatsapp/internal/commands/admin"
	"github.com/Turgho/Shinobu-Whatsapp/internal/commands/public"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/birthday"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/geocoding"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/history"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/ia"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/music"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/scheduler"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/weather"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/weekday"
	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/configs"
	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/database"
	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/uptime"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
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

	client, err := bot.NewClient(ctx, db)
	if err != nil {
		return fmt.Errorf("erro ao criar client WhatsApp: %w", err)
	}

	if err := client.Connect(ctx); err != nil {
		return fmt.Errorf("erro ao conectar no WhatsApp: %w", err)
	}

	r := buildRouter(cfg, client.WAClient, logger, store)

	r.SetAIConfig(&ia.Config{
		GroqURL:   cfg.Groq.URL,
		GroqKey:   cfg.Groq.APIKey,
		TavilyKey: cfg.Tavily.APIKey,
		Log:       logger.Named("AI"),
	})

	r.StartRateLimitCleanup(ctx)

	// Inicializa scheduler e store de jobs dinâmicos
	dynLogger := logger.Named("DYNAMIC")
	dynStore := scheduler.NewDynamicStore("storage/dynamic_jobs.json", dynLogger)
	sched := scheduler.NewScheduler(logger.Named("SCHEDULER"))

	registerPublicCommands(r, cfg, logger, store, sched, dynStore)
	registerAdminCommands(r, cfg, store, logger)
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

		job := scheduler.NewDynamicJob(data.ID, runAt, data.ChatJID, data.Message, client.WAClient, dynLogger)
		sched.Register(job)
		dynLogger.Info("Job dinâmico restaurado",
			zap.String("id", data.ID),
			zap.Time("run_at", runAt),
		)
	}

	birthdayJob, err := birthday.NewBirthdayJob(client.WAClient, logger.Named("BIRTHDAY"))
	if err != nil {
		logger.Error("Erro ao criar job de aniversário", zap.Error(err))
	} else {
		sched.Register(birthdayJob)
	}

	for _, j := range cfg.ScheduledJobs {
		if job := weekday.NewFromConfig(client.WAClient, logger.Named(j.Name), weekday.Config{
			Name:         j.Name,
			Day:          j.Day,
			Enabled:      j.Enabled,
			Hour:         j.Hour,
			Minute:       j.Minute,
			AudioPath:    j.AudioPath,
			StickerName:  j.StickerName,
			TargetGroups: j.TargetGroups,
		}); job != nil {
			sched.Register(job)
		}
	}

	sched.Start(ctx)

	handler := bot.NewHandler(client.WAClient, r)
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

func buildRouter(cfg *configs.Config, waClient *whatsmeow.Client, logger *zap.Logger, store *history.Store) *commands.Router {
	r := commands.NewRouter(cfg.Bot.Prefix, waClient, logger.Named("ROUTER"), store)

	r.Use(commands.IgnoreSelfMiddleware)
	r.Use(commands.IgnoreOldMessagesMiddleware)
	r.Use(commands.CommandNotFoundMiddleware(r))
	r.Use(commands.PrivateCommandsMiddleware(r, cfg.UsersJID.Owner, cfg.UsersJID.Admins))

	return r
}

func registerPublicCommands(r *commands.Router, cfg *configs.Config, logger *zap.Logger, store *history.Store, sched *scheduler.Scheduler, dynStore *scheduler.DynamicStore) {
	geoClient := geocoding.NewGeoCoding(cfg.ApiURLs.Geocoding, cfg.ApiURLs.OpenMeteoGeo, logger.Named("GEOCODING"))
	weatherClient := weather.NewWeatherClient(cfg.ApiURLs.Weather, logger.Named("WEATHER"))

	r.RegisterCommand(commands.CommandMeta{
		Name:        "menu",
		Description: "Lista todos os comandos disponíveis",
		Type:        commands.CommandTypeUtility,
	}, public.MenuCommand(r))

	r.RegisterCommand(commands.CommandMeta{
		Name:        "ping",
		Description: "Verifica se o bot está online",
		Type:        commands.CommandTypeUtility,
	}, public.PingCommand)

	r.RegisterCommand(commands.CommandMeta{
		Name:        "clima",
		Description: "Mostra o clima atual de uma cidade",
		Type:        commands.CommandTypeUtility,
		Args:        []commands.ArgMeta{{Name: "cidade", Required: true}},
	}, weatherHandler(geoClient, weatherClient))

	r.RegisterCommand(commands.CommandMeta{
		Name:        "sticker",
		Description: "Gera uma figurinha com base em uma imagem ou vídeo",
		Type:        commands.CommandTypeUtility,
	}, public.StickerCommand)

	musicCfg := &music.Config{
		ServerURL: cfg.Music.ServerURL,
		APIToken:  cfg.Music.APIToken,
	}

	r.RegisterCommand(commands.CommandMeta{
		Name:        "play",
		Description: "Busca por uma música via nome ou URL",
		Type:        commands.CommandTypeDownload,
		Args:        []commands.ArgMeta{{Name: "nome da música ou URL", Required: true}},
	}, public.PlayCommand(musicCfg))

	r.RegisterCommand(commands.CommandMeta{
		Name:        "mambo",
		Description: "M A M B O 🏇",
		Type:        commands.CommandTypeFun,
	}, public.FixedBundledAudioCommand("assets/audios/mambo.ogg", ""))

	r.RegisterCommand(commands.CommandMeta{
		Name:        "dio",
		Description: "Talvez o tempo pare...",
		Type:        commands.CommandTypeFun,
	}, public.FixedBundledAudioCommand("assets/audios/zawarudo.ogg", "zawarudo"))

	r.RegisterCommand(commands.CommandMeta{
		Name:        "cafe",
		Description: "Não importa a hora!",
		Type:        commands.CommandTypeFun,
	}, public.FixedBundledAudioCommand("assets/audios/hora_cafe.ogg", "hora_cafe"))

	r.RegisterCommand(commands.CommandMeta{
		Name:        "shinobu",
		Description: "converse com shinobu",
		Type:        commands.CommandTypeAI,
		Args:        []commands.ArgMeta{{Name: "escreva algo", Required: false}},
	}, public.ShinobuCommand(store, cfg))

	r.RegisterCommand(commands.CommandMeta{
		Name:        "aniversário",
		Description: "Gerencia aniversário de grupos",
		Type:        commands.CommandTypeGroup,
	}, public.BirthdayCommand(cfg.UsersJID.Owner, cfg.UsersJID.Admins))

	r.RegisterCommand(commands.CommandMeta{
		Name:        "efeito",
		Description: "Aplica efeitos em um áudio. Use !efeito para ver os disponíveis.",
		Type:        commands.CommandTypeMedia,
		Args: []commands.ArgMeta{
			{Name: "efeito", Required: false},
			{Name: "intensidade", Required: false},
		},
	}, public.AudioEffectsCommand)

	r.RegisterCommand(commands.CommandMeta{
		Name:        "agenda",
		Description: "Agenda um lembrete. Ex: agenda 2026-06-28T09:00 tomar remédio",
		Type:        commands.CommandTypeUtility,
		Args: []commands.ArgMeta{
			{Name: "data ISO8601", Required: true},
			{Name: "mensagem", Required: true},
		},
	}, public.AgendaHandler(sched, dynStore, logger))
}

func registerAdminCommands(r *commands.Router, cfg *configs.Config, store *history.Store, logger *zap.Logger) {
	musicCfg := &music.Config{
		ServerURL: cfg.Music.ServerURL,
		APIToken:  cfg.Music.APIToken,
	}

	r.RegisterCommand(commands.CommandMeta{
		Name:        "stats",
		Description: "Exibe estatísticas de runtime do bot",
		Type:        commands.CommandTypeOwner,
		Private:     true,
	}, admin.StatsCommand(musicCfg))

	r.RegisterCommand(commands.CommandMeta{
		Name:        "shutdown",
		Description: "Desliga o bot",
		Type:        commands.CommandTypeOwner,
		Private:     true,
	}, admin.ShutdownCommand)

	r.RegisterCommand(commands.CommandMeta{
		Name:        "restart",
		Description: "Reinicia o bot",
		Type:        commands.CommandTypeOwner,
		Private:     true,
	}, admin.RestartCommand)

	r.RegisterCommand(commands.CommandMeta{
		Name:        "fig",
		Description: "Gerencia stickers salvos. Uso em DM: !fig salvar <nome>, !fig remover <nome>, !fig lista. Uso normal: !fig <nome>",
		Type:        commands.CommandTypeAdmin,
		Args: []commands.ArgMeta{
			{Name: "nome", Required: false},
		},
		Private: true,
	}, admin.SaveStickerCommand(cfg.UsersJID.Owner))

	r.RegisterCommand(commands.CommandMeta{
		Name:        "ignorar",
		Description: "Ignorar mensagens de un número",
		Type:        commands.CommandTypeAdmin,
		Private:     true,
	}, admin.IgnoreCommand())

	r.RegisterCommand(commands.CommandMeta{
		Name:        "testjob",
		Description: "Testa o job semanal (audio+@all+sticker) no chat atual",
		Type:        commands.CommandTypeOwner,
		Private:     true,
	}, admin.TestJobCommand())

	r.RegisterCommand(commands.CommandMeta{
		Name:        "manutencao",
		Description: "Ativa/desativa modo manutenção (comandos bloqueados)",
		Type:        commands.CommandTypeOwner,
		Private:     true,
	}, admin.ManutencaoCommand(r))

	r.RegisterCommand(commands.CommandMeta{
		Name:        "memoria",
		Description: "Gerencia a memória da IA no chat. Subcomandos: ver, limpar [@user], resumo",
		Type:        commands.CommandTypeAdmin,
		Private:     true,
	}, admin.MemoriaHandler(store, logger))
}

func weatherHandler(geo *geocoding.GeoCoding, wc *weather.WeatherClient) commands.HandlerFunc {
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		return public.WeatherCommand(ctx, client, evt, args, geo, wc)
	}
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

func registerAliases(r *commands.Router) {
	aliases := map[string]string{
		// Shortcuts
		"p": "play",
		"s": "sticker",
		"m": "menu",
		"c": "clima",
		"e": "efeito",
		"a": "aniversário",

		// Common misspellings
		"plau":      "play",
		"plei":      "play",
		"stiker":    "sticker",
		"figurinha": "sticker",
		"clim":      "clima",
		"tempo":     "clima",
		"aniver":    "aniversário",
		"aniversario": "aniversário",
		"lembrete":  "agenda",
	}

	for alias, target := range aliases {
		r.RegisterAlias(alias, target)
	}
}
