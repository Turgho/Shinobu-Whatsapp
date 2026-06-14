package football

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/httpclient"
	"go.uber.org/zap"
)

// APIFootballProvider implements ScoreProvider using the API-Football service.
type APIFootballProvider struct {
	BaseURL string
	APIKey  string
	Logger  *zap.Logger
	client  *http.Client
}

// NewAPIFootballProvider creates a new API-Football provider.
func NewAPIFootballProvider(baseURL, apiKey string, logger *zap.Logger) *APIFootballProvider {
	return &APIFootballProvider{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Logger:  logger,
		client:  httpclient.Client,
	}
}

// apiResponse represents the standard API-Football response wrapper.
type apiResponse[T any] struct {
	Get        string            `json:"get"`
	Parameters map[string]string `json:"parameters"`
	Errors     []string          `json:"errors"`
	Results    int               `json:"results"`
	Paging     struct {
		Current int `json:"current"`
		Total   int `json:"total"`
	} `json:"paging"`
	Response []T `json:"response"`
}

// apiFixture represents a fixture from the API.
type apiFixture struct {
	Fixture struct {
		ID     int `json:"id"`
		Status struct {
			Long    string `json:"long"`
			Short   string `json:"short"`
			Elapsed int    `json:"elapsed"`
		} `json:"status"`
		Date  string `json:"date"`
		Venue struct {
			Name string `json:"name"`
		} `json:"venue"`
	} `json:"fixture"`
	League struct {
		ID      int    `json:"id"`
		Name    string `json:"name"`
		Country string `json:"country"`
		Logo    string `json:"logo"`
		Flag    string `json:"flag"`
		Season  int    `json:"season"`
		Round   string `json:"round"`
	} `json:"league"`
	Teams struct {
		Home struct {
			ID     int    `json:"id"`
			Name   string `json:"name"`
			Logo   string `json:"logo"`
			Winner *bool  `json:"winner"`
		} `json:"home"`
		Away struct {
			ID     int    `json:"id"`
			Name   string `json:"name"`
			Logo   string `json:"logo"`
			Winner *bool  `json:"winner"`
		} `json:"away"`
	} `json:"teams"`
	Goals struct {
		Home int `json:"home"`
		Away int `json:"away"`
	} `json:"goals"`
	Score struct {
		Halftime  map[string]int `json:"halftime"`
		Fulltime  map[string]int `json:"fulltime"`
		Extratime map[string]int `json:"extratime"`
		Penalty   map[string]int `json:"penalty"`
	} `json:"score"`
}

// apiEvent represents a match event from the API.
type apiEvent struct {
	Time struct {
		Elapsed int  `json:"elapsed"`
		Extra   *int `json:"extra"`
	} `json:"time"`
	Team struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
		Logo string `json:"logo"`
	} `json:"team"`
	Player struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"player"`
	Assist struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"assist"`
	Type     string `json:"type"`
	Detail   string `json:"detail"`
	Comments string `json:"comments"`
}

// GetLiveFixturesForTeam returns live matches for the given team ID.
func (p *APIFootballProvider) GetLiveFixturesForTeam(ctx context.Context, teamID int) ([]Match, error) {
	endpoint := fmt.Sprintf("%s/fixtures", p.BaseURL)
	params := url.Values{}
	params.Set("live", "all")
	params.Set("team", strconv.Itoa(teamID))

	var resp apiResponse[apiFixture]
	if err := p.doRequest(ctx, endpoint, params, &resp); err != nil {
		return nil, err
	}

	if len(resp.Errors) > 0 {
		p.Logger.Error("API-Football retornou erros", zap.Strings("errors", resp.Errors))
		return nil, fmt.Errorf("football: erros da API: %v", resp.Errors)
	}

	matches := make([]Match, 0, len(resp.Response))
	for _, f := range resp.Response {
		if f.Fixture.Status.Short != "LIVE" && f.Fixture.Status.Short != "1H" && f.Fixture.Status.Short != "2H" && f.Fixture.Status.Short != "HT" && f.Fixture.Status.Short != "ET" && f.Fixture.Status.Short != "BT" && f.Fixture.Status.Short != "P" && f.Fixture.Status.Short != "SUSP" {
			continue
		}

		startTime, _ := time.Parse(time.RFC3339, f.Fixture.Date)

		matches = append(matches, Match{
			ID: f.Fixture.ID,
			HomeTeam: Team{
				Name:      f.Teams.Home.Name,
				APITeamID: f.Teams.Home.ID,
			},
			AwayTeam: Team{
				Name:      f.Teams.Away.Name,
				APITeamID: f.Teams.Away.ID,
			},
			HomeScore:   f.Goals.Home,
			AwayScore:   f.Goals.Away,
			Status:      f.Fixture.Status.Long,
			StartTime:   startTime,
			ElapsedTime: f.Fixture.Status.Elapsed,
			League:      f.League.Name,
			Season:      f.League.Season,
			Round:       f.League.Round,
		})
	}

	return matches, nil
}

// GetUpcomingFixturesForTeam returns upcoming matches for the given team ID.
func (p *APIFootballProvider) GetUpcomingFixturesForTeam(ctx context.Context, teamID int) ([]Match, error) {
	endpoint := fmt.Sprintf("%s/fixtures", p.BaseURL)
	params := url.Values{}
	params.Set("team", strconv.Itoa(teamID))
	params.Set("next", "10")
	params.Set("status", "NS,TBD")

	var resp apiResponse[apiFixture]
	if err := p.doRequest(ctx, endpoint, params, &resp); err != nil {
		return nil, err
	}

	if len(resp.Errors) > 0 {
		p.Logger.Error("API-Football retornou erros", zap.Strings("errors", resp.Errors))
		return nil, fmt.Errorf("football: erros da API: %v", resp.Errors)
	}

	matches := make([]Match, 0, len(resp.Response))
	for _, f := range resp.Response {
		startTime, _ := time.Parse(time.RFC3339, f.Fixture.Date)

		matches = append(matches, Match{
			ID: f.Fixture.ID,
			HomeTeam: Team{
				Name:      f.Teams.Home.Name,
				APITeamID: f.Teams.Home.ID,
			},
			AwayTeam: Team{
				Name:      f.Teams.Away.Name,
				APITeamID: f.Teams.Away.ID,
			},
			HomeScore: f.Goals.Home,
			AwayScore: f.Goals.Away,
			Status:    f.Fixture.Status.Long,
			StartTime: startTime,
			League:    f.League.Name,
			Season:    f.League.Season,
			Round:     f.League.Round,
		})
	}

	return matches, nil
}

// GetMatchEvents returns events (goals, etc.) for a given match ID.
func (p *APIFootballProvider) GetMatchEvents(ctx context.Context, matchID int) ([]GoalEvent, error) {
	endpoint := fmt.Sprintf("%s/fixtures/events", p.BaseURL)
	params := url.Values{}
	params.Set("fixture", strconv.Itoa(matchID))

	var resp apiResponse[apiEvent]
	if err := p.doRequest(ctx, endpoint, params, &resp); err != nil {
		return nil, err
	}

	if len(resp.Errors) > 0 {
		p.Logger.Error("API-Football retornou erros", zap.Strings("errors", resp.Errors))
		return nil, fmt.Errorf("football: erros da API: %v", resp.Errors)
	}

	events := make([]GoalEvent, 0, len(resp.Response))
	for _, e := range resp.Response {
		extraTime := 0
		if e.Time.Extra != nil {
			extraTime = *e.Time.Extra
		}

		// Generate a unique ID for the event based on matchID, minute, team, player
		// This is used for deduplication since API doesn't provide event IDs
		eventID := matchID*10000 + e.Time.Elapsed*100 + extraTime
		if e.Player.ID > 0 {
			eventID += e.Player.ID
		}

		events = append(events, GoalEvent{
			ID:        eventID,
			MatchID:   matchID,
			Team:      e.Team.Name,
			Player:    e.Player.Name,
			Minute:    e.Time.Elapsed,
			ExtraTime: extraTime,
			Type:      e.Type,
		})
	}

	return events, nil
}

// doRequest performs an HTTP GET request to the API-Football API.
func (p *APIFootballProvider) doRequest(ctx context.Context, endpoint string, params url.Values, result interface{}) error {
	reqURL := endpoint + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		p.Logger.Error("Erro ao criar request", zap.Error(err), zap.String("url", reqURL))
		return fmt.Errorf("football: criar request: %w", err)
	}

	req.Header.Set("x-apisports-key", p.APIKey)
	req.Header.Set("Accept", "application/json")

	p.Logger.Debug("Fazendo request à API-Football", zap.String("url", reqURL))

	resp, err := p.client.Do(req)
	if err != nil {
		p.Logger.Error("Erro ao fazer request HTTP", zap.Error(err), zap.String("url", reqURL))
		return fmt.Errorf("football: erro na requisição: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		p.Logger.Error("API-Football retornou status inesperado", zap.Int("status", resp.StatusCode), zap.String("url", reqURL))
		return fmt.Errorf("football: status %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		p.Logger.Error("Erro ao decodificar resposta", zap.Error(err), zap.String("url", reqURL))
		return fmt.Errorf("football: erro ao decodificar resposta: %w", err)
	}

	return nil
}
