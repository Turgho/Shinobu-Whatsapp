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
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/football"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/geocoding"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/history"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/music"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/weather"
	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/configs"
	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/database"
	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/uptime"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
	"go.uber.org/zap"
)

func Run() error {
	uptime.Start()

	if err := checkDeps(); err != nil {
		return fmt.Errorf("dependências ausentes: %w", err)
	}

	cfg := configs.Load()

	logger, err := buildLogger()
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

	store, err := history.NewStore("storage/message_history.db")
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

	registerPublicCommands(r, cfg, logger, store)
	registerAdminCommands(r, cfg)

	birthday.StartScheduler(client.WAClient)
	football.Start(ctx, client)

	handler := bot.NewHandler(client.WAClient, r)
	client.RegisterHandlers(handler.EventHandler)

	client.Listen()
	return nil
}

func buildLogger() (*zap.Logger, error) {
	logCfg := zap.NewDevelopmentConfig()
	logCfg.DisableStacktrace = true
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

	r.Use(commands.IgnoreOldMessagesMiddleware)
	r.Use(commands.CommandNotFoundMiddleware(r))
	r.Use(commands.PrivateCommandsMiddleware(r, cfg.UsersJID.Owner, cfg.UsersJID.Admins))

	return r
}

func registerPublicCommands(r *commands.Router, cfg *configs.Config, logger *zap.Logger, store *history.Store) {
	geoClient := geocoding.NewGeoCoding(cfg.ApiURLs.Geocoding, logger.Named("GEOCODING"))
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
}

func registerAdminCommands(r *commands.Router, cfg *configs.Config) {
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
