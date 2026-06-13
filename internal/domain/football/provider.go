package football

import (
	"context"
	"go.uber.org/zap"
)

// APIFootballProvider implements ScoreProvider using the API-Football service.
type APIFootballProvider struct {
	BaseURL string
	APIKey  string
	Logger  *zap.Logger
}

// NewAPIFootballProvider creates a new API-Football provider.
func NewAPIFootballProvider(baseURL, apiKey string, logger *zap.Logger) *APIFootballProvider {
	return &APIFootballProvider{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Logger:  logger,
	}
}

// GetLiveFixturesForTeam returns live matches for the given team ID.
func (p *APIFootballProvider) GetLiveFixturesForTeam(ctx context.Context, teamID int) ([]Match, error) {
	return nil, nil
}

// GetUpcomingFixturesForTeam returns upcoming matches for the given team ID.
func (p *APIFootballProvider) GetUpcomingFixturesForTeam(ctx context.Context, teamID int) ([]Match, error) {
	return nil, nil
}

// GetMatchEvents returns events (goals, etc.) for a given match ID.
func (p *APIFootballProvider) GetMatchEvents(ctx context.Context, matchID int) ([]GoalEvent, error) {
	return nil, nil
}
