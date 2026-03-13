package app

import (
	"context"
	"fmt"

	"github.com/Turgho/YuukoWhatsapp/internal/bot"
	"github.com/Turgho/YuukoWhatsapp/internal/commands"
	admin "github.com/Turgho/YuukoWhatsapp/internal/commands/admin"
	public "github.com/Turgho/YuukoWhatsapp/internal/commands/public"
	"github.com/Turgho/YuukoWhatsapp/internal/configs"
	"github.com/Turgho/YuukoWhatsapp/internal/database"
	"github.com/Turgho/YuukoWhatsapp/internal/utils"
	"github.com/Turgho/YuukoWhatsapp/pkg/geocoding"
	"github.com/Turgho/YuukoWhatsapp/pkg/weather"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
	"go.uber.org/zap"
)

// Run inicia todo o bot
func Run() error {
	// Inicia uptime
	utils.StartUptime()

	// Carrega as configurações
	cfg := configs.Load()

	// Inicia logger sem stack trace
	logCfg := zap.NewDevelopmentConfig()
	logCfg.DisableStacktrace = true
	logger, _ := logCfg.Build()
	defer logger.Sync()

	// Banco de dados
	dbLogger := logger.Named("DATABASE")
	conn := database.NewDatabase(cfg.Database.Driver, cfg.Database.Dsn, dbLogger)
	db, err := conn.Connect()
	if err != nil {
		return fmt.Errorf("erro ao conectar no banco: %w", err)
	}
	defer db.Close()

	// Contexto base
	ctx := context.Background()

	// Client WhatsApp
	client, err := bot.NewClient(ctx, db)
	if err != nil {
		return fmt.Errorf("erro ao criar client: %w", err)
	}

	if err := client.Connect(ctx); err != nil {
		return fmt.Errorf("erro ao conectar no WhatsApp: %w", err)
	}

	// Router
	routerLogger := logger.Named("ROUTER")
	r := commands.NewRouter(cfg.Bot.Prefix, client.WAClient, routerLogger)

	privateCommands := map[string]bool{
		"shutdown": true,
		"stats":    true,
	}

	// r.Use(commands.IgnoreSelf)
	r.Use(commands.IgnoreOldMessagesMiddleware)
	r.Use(commands.CommandNotFoundMiddleware(r))
	r.Use(commands.PrivateCommandsMiddleware(cfg.UsersJID.Owner, cfg.UsersJID.Admins, privateCommands))

	geoLogger := logger.Named("GEOCODING")
	geoClient := geocoding.NewGeoCoding(cfg.ApiURLs.Geocoding, geoLogger)

	weatherLogger := logger.Named("WEATHER")
	weatherClient := weather.NewWeatherClient(cfg.ApiURLs.Weather, weatherLogger)

	// Comandos públicos
	r.RegisterCommand("ping", public.PingCommand)
	r.RegisterCommand("weather", func(client *whatsmeow.Client, evt *events.Message, args []string) error {
		return public.WeatherCommand(client, evt, args, geoClient, weatherClient)
	})

	// Comandos Privados
	r.RegisterCommand("stats", admin.StatsCommand)
	r.RegisterCommand("shutdown", admin.ShutdownCommand)

	// Handler
	handler := bot.NewHandler(client.WAClient, r)
	client.RegisterHandlers(handler.EventHandler)

	// Mantém o bot rodando
	client.Listen()
	return nil
}
