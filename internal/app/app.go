// Package app monta logger, banco, histórico, cliente WhatsApp, router e handlers.
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
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/weather"
	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/configs"
	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/database"
	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/uptime"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
	"go.uber.org/zap"
)

// Run inicializa dependências, conecta ao WhatsApp e bloqueia em Listen até encerrar.
func Run() error {
	uptime.Start()

	// Verifica dependências externas antes de qualquer conexão
	if err := checkDeps(); err != nil {
		return fmt.Errorf("dependências ausentes: %w", err)
	}

	cfg := configs.Load()

	// Logger
	logger, err := buildLogger()
	if err != nil {
		return fmt.Errorf("erro ao inicializar logger: %w", err)
	}
	defer logger.Sync()

	ctx := context.Background()

	// Database SQLite para o Whatsmeow
	db, err := connectDatabase(cfg, logger)
	if err != nil {
		return err
	}
	defer db.Close()

	// Histórico de mensagens por usuário — usado pela IA para contexto de conversa
	store, err := history.NewStore("storage/message_history.db")
	if err != nil {
		logger.Error("erro ao abrir history", zap.Error(err))
		return fmt.Errorf("erro ao abrir history: %w", err)
	}
	defer store.Close()

	store.StartCleanup(ctx, 24*time.Hour) // apaga mensagens com mais de 24h

	// Inicializa o cliente WhatsApp
	client, err := bot.NewClient(ctx, db)
	if err != nil {
		return fmt.Errorf("erro ao criar client WhatsApp: %w", err)
	}

	// Conecta ao WhatsApp
	if err := client.Connect(ctx); err != nil {
		return fmt.Errorf("erro ao conectar no WhatsApp: %w", err)
	}

	// Inicializa o router
	r := buildRouter(cfg, client.WAClient, logger, store)

	// Cadastra comandos públicos
	registerPublicCommands(r, cfg, logger, store)

	// Cadastra comandos privados
	registerAdminCommands(r, cfg)

	birthday.StartScheduler(client.WAClient)

	registry := ia.NewToolRegistry()
	registry.RegisterFromRouter(r)
	ia.SetRegistry(registry)

	// Inicializa o handler
	handler := bot.NewHandler(client.WAClient, r)
	client.RegisterHandlers(handler.EventHandler)

	// Bloqueia em Listen até encerrar
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

	// IgnoreSelfMiddleware: descarta mensagens enviadas pelo próprio bot (desativado por ora)
	// r.Use(commands.IgnoreSelfMiddleware)

	// Descarta mensagens antigas recebidas ao reconectar
	r.Use(commands.IgnoreOldMessagesMiddleware)

	// Notifica quando um comando não existe
	r.Use(commands.CommandNotFoundMiddleware(r))

	// Bloqueia comandos privados para quem não é owner/admin
	r.Use(commands.PrivateCommandsMiddleware(r, cfg.UsersJID.Owner, cfg.UsersJID.Admins))

	return r
}

// registerPublicCommands cadastra comandos abertos a qualquer usuário autorizado pelo middleware.
//
// Para adicionar um comando público: crie o handler em internal/commands/public/,
// chame r.RegisterCommand aqui; o !menu lista automaticamente.
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

	r.RegisterCommand(commands.CommandMeta{
		Name:        "play",
		Description: "Busca por uma música via nome ou URL",
		Type:        commands.CommandTypeDownload,
		Args:        []commands.ArgMeta{{Name: "nome da música ou URL", Required: true}},
	}, public.PlayCommand)

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

	// Shinobu: IA com personalidade, histórico por usuário e busca web sob demanda
	r.RegisterCommand(commands.CommandMeta{
		Name:        "shinobu",
		Description: "converse com shinobu",
		Type:        commands.CommandTypeAI,
		Args:        []commands.ArgMeta{{Name: "escreva algo", Required: false}},
	}, public.ShinobuCommand(store))

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

// registerAdminCommands cadastra comandos com Private: true (owner/admins no middleware).
func registerAdminCommands(r *commands.Router, cfg *configs.Config) {
	r.RegisterCommand(commands.CommandMeta{
		Name:        "stats",
		Description: "Exibe estatísticas de runtime do bot",
		Type:        commands.CommandTypeOwner,
		Private:     true,
	}, admin.StatsCommand)

	r.RegisterCommand(commands.CommandMeta{
		Name:        "shutdown",
		Description: "Desliga o bot",
		Type:        commands.CommandTypeOwner,
		Private:     true,
	}, admin.ShutdownCommand)

	r.RegisterCommand(commands.CommandMeta{
		Name:        "fig",
		Description: "Gerencia stickers salvos. Uso em DM: !fig salvar <nome>, !fig remover <nome>, !fig lista. Uso normal: !fig <nome>",
		Type:        commands.CommandTypeAdmin,
		Args: []commands.ArgMeta{
			{Name: "nome", Required: false},
		},
		Private: true,
	}, admin.SaveStickerCommand(cfg.UsersJID.Owner))
}

// weatherHandler envolve WeatherCommand injetando as dependências externas.
// Esse padrão mantém a assinatura de HandlerFunc sem poluir os tipos centrais.
func weatherHandler(geo *geocoding.GeoCoding, wc *weather.WeatherClient) commands.HandlerFunc {
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		return public.WeatherCommand(ctx, client, evt, args, geo, wc)
	}
}

// checkDeps garante que o processo encontra ffmpeg/webpmux nos caminhos usados
// pelo internal/infra/ffmpeg e internal/domain/sticker (execução típica: cwd = raiz do repositório).
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
