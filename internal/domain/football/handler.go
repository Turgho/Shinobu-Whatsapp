package football

import (
	"context"

	"github.com/Turgho/Shinobu-Whatsapp/internal/bot"
	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/configs"
	"go.uber.org/zap"
)

// Start initializes and starts the football goal watcher.
// It should be called from the app's initialization.
func Start(ctx context.Context, waClient *bot.Client) {
	// Load configuration.
	cfg := configs.Load()
	if cfg == nil {
		zap.L().Error("config não carregado")
		return
	}

	// Check if football is enabled.
	if !cfg.Football.Enabled {
		zap.L().Info("football watcher desativado na configuração")
		return
	}

	// Validate required fields.
	if cfg.Football.APIKey == "" {
		zap.L().Error("API key para futebol não configurada")
		return
	}
	if cfg.Football.NotifyJID == "" {
		zap.L().Error("JID para notificação de futebol não configurado")
		return
	}
	if len(cfg.Football.WatchedTeams) == 0 {
		zap.L().Error("nenhum time configurado para assistir")
		return
	}

	// Convert configs.FootballTeam to football.Team.
	var watchedTeams []Team
	for _, t := range cfg.Football.WatchedTeams {
		watchedTeams = append(watchedTeams, Team{
			Name:      t.Name,
			APITeamID: t.APITeamID,
			Flag:      t.Flag,
		})
	}

	// Create a logger for the football domain.
	footballLogger := zap.L().Named("FOOTBALL")

	// Create the API-Football provider.
	provider := NewAPIFootballProvider(
		"https://v3.football.api-sports.io ",
		cfg.Football.APIKey,
		footballLogger,
	)

	// Create the watcher.
	watcher := NewWatcher(waClient.WAClient, &Config{
		Enabled:      cfg.Football.Enabled,
		APIKey:       cfg.Football.APIKey,
		NotifyJID:    cfg.Football.NotifyJID,
		PollInterval: cfg.Football.PollInterval,
		WatchedTeams: watchedTeams,
	}, footballLogger, provider)

	// Start the watcher in a background goroutine.
	watcher.Start(ctx)
}
