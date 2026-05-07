package app

import (
	"context"
	"database/sql"
	"fmt"
	"os/exec"

	"github.com/Turgho/YuukoWhatsapp/internal/bot"
	"github.com/Turgho/YuukoWhatsapp/internal/commands"
	"github.com/Turgho/YuukoWhatsapp/internal/commands/admin"
	"github.com/Turgho/YuukoWhatsapp/internal/commands/public"
	"github.com/Turgho/YuukoWhatsapp/internal/configs"
	"github.com/Turgho/YuukoWhatsapp/internal/database"
	"github.com/Turgho/YuukoWhatsapp/internal/utils"
	"github.com/Turgho/YuukoWhatsapp/pkg/geocoding"
	"github.com/Turgho/YuukoWhatsapp/pkg/weather"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
	"go.uber.org/zap"
)

// Run é o ponto de entrada da aplicação.
func Run() error {
	utils.StartUptime()

	// Verifica dependências externas antes de qualquer conexão
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

	client, err := bot.NewClient(ctx, db)
	if err != nil {
		return fmt.Errorf("erro ao criar client WhatsApp: %w", err)
	}

	if err := client.Connect(ctx); err != nil {
		return fmt.Errorf("erro ao conectar no WhatsApp: %w", err)
	}

	r := buildRouter(cfg, client.WAClient, logger)
	registerCommands(r, cfg, logger)

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

func buildRouter(cfg *configs.Config, waClient *whatsmeow.Client, logger *zap.Logger) *commands.Router {
	r := commands.NewRouter(cfg.Bot.Prefix, waClient, logger.Named("ROUTER"))

	// r.Use(commands.IgnoreSelfMiddleware)
	r.Use(commands.IgnoreOldMessagesMiddleware)
	r.Use(commands.CommandNotFoundMiddleware(r))
	r.Use(commands.PrivateCommandsMiddleware(r, cfg.UsersJID.Owner, cfg.UsersJID.Admins))

	return r
}

// registerCommands é o único lugar onde comandos são cadastrados.
//
// Para adicionar um novo comando:
//  1. Crie o handler em internal/commands/public/ ou admin/
//  2. Chame r.RegisterCommand com os metadados e o handler
//  3. O comando aparece automaticamente no !menu — sem mais nada.
func registerCommands(r *commands.Router, cfg *configs.Config, logger *zap.Logger) {
	geoClient := geocoding.NewGeoCoding(cfg.ApiURLs.Geocoding, logger.Named("GEOCODING"))
	weatherClient := weather.NewWeatherClient(cfg.ApiURLs.Weather, logger.Named("WEATHER"))

	// ─── Públicos ──────────────────────────────────────────────────────────

	r.RegisterCommand(commands.CommandMeta{
		Name:        "menu",
		Description: "Lista todos os comandos disponíveis",
	}, public.MenuCommand(r))

	r.RegisterCommand(commands.CommandMeta{
		Name:        "ping",
		Description: "Verifica se o bot está online",
	}, public.PingCommand)

	r.RegisterCommand(commands.CommandMeta{
		Name:        "weather",
		Description: "Mostra o clima atual de uma cidade",
		Args:        []commands.ArgMeta{{Name: "cidade", Required: true}},
	}, weatherHandler(geoClient, weatherClient))

	r.RegisterCommand(commands.CommandMeta{
		Name:        "sticker",
		Description: "Gera uma figurinha com base em uma imagem ou vídeo",
	}, public.StickerCommand)

	r.RegisterCommand(commands.CommandMeta{
		Name:        "play",
		Description: "Busca por uma música via nome ou URL",
		Args:        []commands.ArgMeta{{Name: "nome da música ou URL", Required: true}},
	}, public.PlayCommand)

	r.RegisterCommand(commands.CommandMeta{
		Name:        "mambo",
		Description: "M A M B O 🏇",
	}, public.MamboCommand)

	// ─── Privados (apenas owner/admins) ───────────────────────────────────

	r.RegisterCommand(commands.CommandMeta{
		Name:        "stats",
		Description: "Exibe estatísticas de runtime do bot",
		Private:     true,
	}, admin.StatsCommand)

	r.RegisterCommand(commands.CommandMeta{
		Name:        "shutdown",
		Description: "Desliga o bot",
		Private:     true,
	}, admin.ShutdownCommand)

	r.RegisterCommand(commands.CommandMeta{
		Name:        "shinobu",
		Description: "S H I N O B U",
		Private:     true,
	}, admin.ShinobuCommand)
}

// weatherHandler envolve WeatherCommand injetando as dependências externas.
// Esse padrão mantém a assinatura de HandlerFunc sem poluir os tipos centrais.
func weatherHandler(geo *geocoding.GeoCoding, wc *weather.WeatherClient) commands.HandlerFunc {
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		return public.WeatherCommand(ctx, client, evt, args, geo, wc)
	}
}

func checkDeps() error {
	if _, err := exec.LookPath("yt-dlp"); err != nil {
		return fmt.Errorf(
			"yt-dlp não encontrado no PATH\n" +
				"  → Linux:   sudo apt install yt-dlp\n" +
				"  → macOS:   brew install yt-dlp\n" +
				"  → Windows: https://github.com/yt-dlp/yt-dlp/releases",
		)
	}

	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return fmt.Errorf(
			"ffmpeg não encontrado no PATH\n" +
				"  → Linux:   sudo apt install ffmpeg\n" +
				"  → macOS:   brew install ffmpeg\n" +
				"  → Windows: https://ffmpeg.org/download.html",
		)
	}

	return nil
}
