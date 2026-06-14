package football

import (
	"context"
	"time"

	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/configs"
)

// Team represents a football team being watched.
type Team struct {
	Name      string `json:"name"`        // e.g., "Brasil"
	APITeamID int    `json:"api_team_id"` // External API ID for the team
	Flag      string `json:"flag"`        // Emoji flag, e.g., "🇧🇷"
}

// Match represents a football match.
type Match struct {
	ID          int       `json:"id"`
	HomeTeam    Team      `json:"home_team"`
	AwayTeam    Team      `json:"away_team"`
	HomeScore   int       `json:"home_score"`
	AwayScore   int       `json:"away_score"`
	Status      string    `json:"status"` // e.g., "TBD", "LIVE", "FINISHED"
	StartTime   time.Time `json:"start_time"`
	ElapsedTime int       `json:"elapsed_time"` // in minutes for live matches
	League      string    `json:"league"`
	Season      int       `json:"season"`
	Round       string    `json:"round"`
	LastEventID int       `json:"last_event_id"` // for deduping events (goals)
}

// GoalEvent represents a goal event.
type GoalEvent struct {
	ID        int    `json:"id"`
	MatchID   int    `json:"match_id"`
	Team      string `json:"team"` // "home" or "away"
	Player    string `json:"player"`
	Minute    int    `json:"minute"`
	ExtraTime int    `json:"extra_time"` // e.g., 1 for 45'+1
	Type      string `json:"type"`       // e.g., "Goal", "Penalty"
}

// Config holds the configuration for the football watcher.
type Config struct {
	Enabled      bool                       `mapstructure:"enabled"`
	APIKey       string                     `mapstructure:"api_key"`
	NotifyJID    string                     `mapstructure:"notify_jid"`
	PollInterval configs.PollIntervalConfig `mapstructure:"poll"`
	WatchedTeams []Team                     `mapstructure:"watched_teams"`
}

// ScoreProvider defines the interface for fetching football data.
type ScoreProvider interface {
	// GetLiveFixturesForTeam returns live matches for the given team ID.
	GetLiveFixturesForTeam(ctx context.Context, teamID int) ([]Match, error)
	// GetUpcomingFixturesForTeam returns upcoming matches for the given team ID.
	GetUpcomingFixturesForTeam(ctx context.Context, teamID int) ([]Match, error)
	// GetMatchEvents returns events (goals, etc.) for a given match ID.
	GetMatchEvents(ctx context.Context, matchID int) ([]GoalEvent, error)
}
