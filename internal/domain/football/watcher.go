package football

import (
	"context"
	"fmt"
	"time"

	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/gosafe"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/configs"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.uber.org/zap"
)

// Use the configs package to avoid import not used error.
var _ = configs.PollIntervalConfig{}.IdleInterval

// Watcher watches for goals and sends notifications.
type Watcher struct {
	client    *whatsmeow.Client
	cfg       *Config
	logger    *zap.Logger
	provider  ScoreProvider
	notifyJID string
}

// NewWatcher creates a new football watcher.
func NewWatcher(client *whatsmeow.Client, cfg *Config, logger *zap.Logger, provider ScoreProvider) *Watcher {
	return &Watcher{
		client:    client,
		cfg:       cfg,
		logger:    logger,
		provider:  provider,
		notifyJID: cfg.NotifyJID,
	}
}

// Start begins watching for goals in a background goroutine.
func (w *Watcher) Start(ctx context.Context) {
	gosafe.Go(func() {
		w.run(ctx)
	})
}

func (w *Watcher) run(ctx context.Context) {
	w.logger.Info("Iniciando watcher de futebol",
		zap.Any("watched_teams", w.cfg.WatchedTeams),
		zap.String("notify_jid", w.notifyJID))

	// Log the idle interval once at startup.
	idleDuration, err := time.ParseDuration(w.cfg.PollInterval.IdleInterval)
	if err != nil {
		w.logger.Error("Erro ao parsear idle_interval", zap.Error(err))
		idleDuration = 5 * time.Minute // fallback
	}
	w.logger.Debug("Intervalo de polling ocioso configurado", zap.Duration("idle", idleDuration))

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Watcher de futebol encerrado")
			return
		default:
			// Process each watched team.
			for _, team := range w.cfg.WatchedTeams {
				w.processTeam(ctx, team)
			}

			// Wait for the next idle interval before checking again.
			// In a real implementation, we would adjust the interval based on whether we are in live mode.
			// For simplicity, we'll use the idle interval and check for live matches inside processTeam.
			// We'll implement a simple sleep and then loop.
			time.Sleep(idleDuration)
		}
	}
}

func (w *Watcher) processTeam(ctx context.Context, team Team) {
	w.logger.Debug("Processando time", zap.String("team", team.Name))

	// Check for live matches for this team.
	liveMatches, err := w.provider.GetLiveFixturesForTeam(ctx, team.APITeamID)
	if err != nil {
		w.logger.Error("Erro ao buscar partidas ao vivo", zap.Error(err))
		return
	}

	if len(liveMatches) == 0 {
		// No live matches, we can optionally check for upcoming matches to see if one is about to start.
		// For now, we just skip.
		return
	}

	// For each live match, check for new goals.
	for _, match := range liveMatches {
		w.processMatch(ctx, team, match)
	}
}

func (w *Watcher) processMatch(ctx context.Context, team Team, match Match) {
	w.logger.Debug("Processando partida",
		zap.Int("match_id", match.ID),
		zap.String("home_team", match.HomeTeam.Name),
		zap.String("away_team", match.AwayTeam.Name),
		zap.Int("home_score", match.HomeScore),
		zap.Int("away_score", match.AwayScore),
		zap.String("status", match.Status))

	// Fetch events for this match.
	events, err := w.provider.GetMatchEvents(ctx, match.ID)
	if err != nil {
		w.logger.Error("Erro ao buscar eventos da partida", zap.Error(err))
		return
	}

	// Get the last processed event ID for this match.
	lastEventID, err := GetLastEventID(match.ID)
	if err != nil {
		w.logger.Error("Erro ao obter último evento processado", zap.Error(err))
		return
	}

	// Find new events (with ID > lastEventID).
	var newEvents []GoalEvent
	for _, event := range events {
		if event.ID > lastEventID {
			newEvents = append(newEvents, event)
		}
	}

	if len(newEvents) == 0 {
		return
	}

	// We assume events are ordered by ID (which should be chronological).
	// Process each new event.
	for _, event := range newEvents {
		// Check if the event is a goal for the team we are watching.
		if event.Type != "Goal" && event.Type != "Penalty" {
			continue
		}
		// Determine if the goal is for the team we are watching.
		// We have the team name in event.Team (from the provider).
		// We'll check if it matches either the home or away team name.
		var isWatchedTeamGoal bool
		if event.Team == match.HomeTeam.Name {
			isWatchedTeamGoal = true
		} else if event.Team == match.AwayTeam.Name {
			isWatchedTeamGoal = true
		}
		if !isWatchedTeamGoal {
			continue
		}

		// Build the notification message.
		message := w.buildGoalMessage(team, match, event)

		// Send the WhatsApp notification.
		ctxSend, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		jid, err := types.ParseJID(w.notifyJID)
		if err != nil {
			w.logger.Error("Erro ao parsear JID para notificação", zap.Error(err))
			cancel()
			return
		}
		err = whatsapp.SendTextToJID(ctxSend, w.client, jid, message, nil)
		cancel()
		if err != nil {
			w.logger.Error("Erro ao enviar notificação de gol", zap.Error(err))
			// We still update the last event ID to avoid spamming on repeated failures.
		} else {
			w.logger.Info("Notificação de gol enviada",
				zap.String("team", team.Name),
				zap.String("player", event.Player),
				zap.Int("minute", event.Minute+event.ExtraTime))
		}

		// Update the last event ID to this event's ID to avoid duplicate notifications.
		if err := SetLastEventID(match.ID, event.ID); err != nil {
			w.logger.Error("Erro ao atualizar último evento processado", zap.Error(err))
		}
	}
}

func (w *Watcher) buildGoalMessage(team Team, match Match, event GoalEvent) string {
	// Determine the opposing team and score.
	var opponentTeam string
	var opponentScore int
	var teamScore int
	if event.Team == match.HomeTeam.Name {
		opponentTeam = match.AwayTeam.Name
		teamScore = match.HomeScore
		opponentScore = match.AwayScore
	} else {
		opponentTeam = match.HomeTeam.Name
		teamScore = match.AwayScore
		opponentScore = match.HomeScore
	}

	// Format the minute.
	minute := event.Minute
	if event.ExtraTime > 0 {
		minute = event.Minute + event.ExtraTime
		// Format as "45'+1" for example.
		// We'll just show the total minute for simplicity.
	}

	// Emojis based on the team's flag.
	flag := team.Flag
	if flag == "" {
		flag = "⚽"
	}

	return fmt.Sprintf("%s%s GOOOL DO %s! %s %d x %d %s — %s aos %d'",
		flag, flag, team.Name, team.Name, teamScore, opponentScore, opponentTeam, event.Player, minute)
}
