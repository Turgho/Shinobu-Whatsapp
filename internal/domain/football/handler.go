package football

import (
	"context"

	"github.com/Turgho/Shinobu-Whatsapp/internal/bot"
	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/configs"
	"go.uber.org/zap"
)

// Start inicializa e inicia o watcher de gols da Copa 2026.
// Chamado em internal/app/app.go na inicialização do bot.
func Start(ctx context.Context, waClient *bot.Client) {
	cfg := configs.Load()
	if cfg == nil {
		zap.L().Error("config não carregado")
		return
	}

	if !cfg.Football.Enabled {
		zap.L().Info("watcher de futebol desativado na configuração")
		return
	}

	if cfg.Football.APIKey == "" {
		zap.L().Error("API key do futebol não configurada")
		return
	}
	if cfg.Football.NotifyJID == "" {
		zap.L().Error("JID de notificação não configurado")
		return
	}
	if len(cfg.Football.WatchedTeams) == 0 {
		zap.L().Error("nenhum time configurado para monitorar")
		return
	}

	footballLogger := zap.L().Named("FOOTBALL")

	footballLogger.Debug("validações de configuração concluídas")

	// Cria provedor que chama a API-Football (api-football.com)
	provider := NewAPIFootballProvider(
		"https://v3.football.api-sports.io",
		cfg.Football.APIKey,
		footballLogger,
	)

	// Converte config.FootballTeam para football.Team
	var watchedTeams []Team
	for _, t := range cfg.Football.WatchedTeams {
		watchedTeams = append(watchedTeams, Team{
			Name:      t.Name,
			APITeamID: t.APITeamID,
			Flag:      t.Flag,
		})
	}

	// Para cada time, confirma o ID na API (Copa 2026: league=1, season=2026)
	validTeams := make([]Team, 0, len(watchedTeams))
	for _, team := range watchedTeams {
		if team.APITeamID == 0 {
			footballLogger.Info("api_team_id não configurado, buscando na API-Football",
				zap.String("team", team.Name),
				zap.Int("league", WorldCupLeagueID),
				zap.Int("season", WorldCupSeason))

			confirmedID, err := provider.GetTeamIDByName(ctx, WorldCupLeagueID, WorldCupSeason, team.Name)
			if err != nil {
				footballLogger.Error("Falha ao confirmar ID do time na Copa 2026 - time será ignorado",
					zap.String("team", team.Name),
					zap.Error(err))
				continue
			}
			team.APITeamID = confirmedID
			footballLogger.Info("ID do time confirmado na Copa 2026",
				zap.String("team", team.Name),
				zap.Int("api_team_id", confirmedID))
		}
		validTeams = append(validTeams, team)
	}

	if len(validTeams) == 0 {
		footballLogger.Error("nenhum time válido para monitorar após validação de IDs")
		return
	}
	watchedTeams = validTeams

	// Cria e inicia watcher em background (gosafe.Go)
	watcher := NewWatcher(waClient.WAClient, &Config{
		Enabled:      cfg.Football.Enabled,
		APIKey:       cfg.Football.APIKey,
		NotifyJID:    cfg.Football.NotifyJID,
		PollInterval: cfg.Football.PollInterval,
		WatchedTeams: watchedTeams,
	}, footballLogger, provider)

	watcher.Start(ctx)

	footballLogger.Info("Watcher de futebol iniciado com sucesso",
		zap.Int("watched_teams", len(watchedTeams)),
		zap.String("notify_jid", cfg.Football.NotifyJID),
		zap.String("idle_interval", cfg.Football.PollInterval.IdleInterval),
		zap.String("live_interval", cfg.Football.PollInterval.LiveInterval))
}
