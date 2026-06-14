package football

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/httpclient"
	"go.uber.org/zap"
)

// APIFootballProvider implementa ScoreProvider usando a API-Football (api-football.com).
type APIFootballProvider struct {
	BaseURL string
	APIKey  string
	Logger  *zap.Logger
	client  *http.Client
}

// NewAPIFootballProvider cria provedor para API-Football.
func NewAPIFootballProvider(baseURL, apiKey string, logger *zap.Logger) *APIFootballProvider {
	return &APIFootballProvider{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Logger:  logger,
		client:  httpclient.Client,
	}
}

// apiResponse é o wrapper padrão de resposta da API-Football.
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

// apiFixture representa uma partida retornada pela API-Football (lista).
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

// apiFixtureDetail representa resposta completa de GET /fixtures?id={id}.
type apiFixtureDetail struct {
	Fixture apiFixture `json:"fixture"`
	Events  []apiEvent `json:"events"`
}

// apiEvent representa um evento de partida da API-Football.
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

// apiTeamResponse representa resposta de GET /teams.
type apiTeamResponse struct {
	Team TeamInfo `json:"team"`
}

// GetWorldCupFixturesForTeam busca jogos do time na Copa 2026.
// Chama GET /fixtures?league=1&season=2026&team={teamID}
// Usado no modo idle (1x/dia) para descobrir fixture_id e horário do próximo jogo.
func (p *APIFootballProvider) GetWorldCupFixturesForTeam(ctx context.Context, teamID int) ([]Match, error) {
	endpoint := fmt.Sprintf("%s/fixtures", p.BaseURL)
	params := url.Values{}
	params.Set("league", strconv.Itoa(WorldCupLeagueID))
	params.Set("season", strconv.Itoa(WorldCupSeason))
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
		startTime, err := time.Parse(time.RFC3339, f.Fixture.Date)
		if err != nil {
			p.Logger.Warn("Erro ao parsear data do jogo, usando time zero",
				zap.Error(err), zap.String("date", f.Fixture.Date), zap.Int("fixture_id", f.Fixture.ID))
			startTime = time.Time{}
		}

		matches = append(matches, Match{
			ID:          f.Fixture.ID,
			HomeTeam:    Team{Name: f.Teams.Home.Name, APITeamID: f.Teams.Home.ID},
			AwayTeam:    Team{Name: f.Teams.Away.Name, APITeamID: f.Teams.Away.ID},
			HomeScore:   f.Goals.Home,
			AwayScore:   f.Goals.Away,
			Status:      f.Fixture.Status.Long,
			StatusShort: f.Fixture.Status.Short,
			StartTime:   startTime,
			ElapsedTime: f.Fixture.Status.Elapsed,
			League:      f.League.Name,
			Season:      f.League.Season,
			Round:       f.League.Round,
		})
	}

	return matches, nil
}

// GetMatchDetails busca detalhes completos da partida (placar + eventos).
// Chama GET /fixtures?id={fixture_id} - retorna fixture + events numa call.
// Usado no modo live (intervalo configurado: 15s pago, 60s free).
func (p *APIFootballProvider) GetMatchDetails(ctx context.Context, fixtureID int) (*MatchDetails, error) {
	endpoint := fmt.Sprintf("%s/fixtures", p.BaseURL)
	params := url.Values{}
	params.Set("id", strconv.Itoa(fixtureID))

	var resp apiResponse[apiFixtureDetail]
	if err := p.doRequest(ctx, endpoint, params, &resp); err != nil {
		return nil, err
	}

	if len(resp.Errors) > 0 {
		p.Logger.Error("API-Football retornou erros", zap.Strings("errors", resp.Errors))
		return nil, fmt.Errorf("football: erros da API: %v", resp.Errors)
	}

	if len(resp.Response) == 0 {
		return nil, fmt.Errorf("football: partida %d não encontrada", fixtureID)
	}

	data := resp.Response[0]

	// Converte eventos para GoalEvent
	events := make([]GoalEvent, 0, len(data.Events))
	for _, e := range data.Events {
		extraTime := 0
		if e.Time.Extra != nil {
			extraTime = *e.Time.Extra
		}

		// Gera ID único para deduplicação (API não fornece ID de evento)
		// Fórmula: matchID*1000000 + minuto*10000 + extraTime*100 + typeHash + playerID
		// typeHash: Goal=1, Penalty=2, Other=0 (para diferenciar mesmo minuto/jogador)
		typeHash := 0
		switch e.Type {
		case "Goal":
			typeHash = 1
		case "Penalty":
			typeHash = 2
		}

		eventID := fixtureID*1000000 + e.Time.Elapsed*10000 + extraTime*100 + typeHash
		if e.Player.ID > 0 {
			eventID += e.Player.ID
		}

		events = append(events, GoalEvent{
			ID:        eventID,
			MatchID:   fixtureID,
			TeamID:    e.Team.ID,
			TeamName:  e.Team.Name,
			Player:    e.Player.Name,
			Assist:    e.Assist.Name,
			Minute:    e.Time.Elapsed,
			ExtraTime: extraTime,
			Type:      e.Type,
			Detail:    e.Detail,
		})
	}

	// Atualiza fixture com dados mais recentes
	f := data.Fixture
	startTime, _ := time.Parse(time.RFC3339, f.Fixture.Date)

	match := Match{
		ID:          f.Fixture.ID,
		HomeTeam:    Team{Name: f.Teams.Home.Name, APITeamID: f.Teams.Home.ID},
		AwayTeam:    Team{Name: f.Teams.Away.Name, APITeamID: f.Teams.Away.ID},
		HomeScore:   f.Goals.Home,
		AwayScore:   f.Goals.Away,
		Status:      f.Fixture.Status.Long,
		StatusShort: f.Fixture.Status.Short,
		StartTime:   startTime,
		ElapsedTime: f.Fixture.Status.Elapsed,
		League:      f.League.Name,
		Season:      f.League.Season,
		Round:       f.League.Round,
	}

	return &MatchDetails{
		Fixture: match,
		Events:  events,
	}, nil
}

// GetTeamIDByName busca o ID do time na liga/temporada.
// Chama GET /teams?league={leagueID}&season={season}&search={name}
// Usado para confirmar o api_team_id do Brasil antes de fixar no config.
func (p *APIFootballProvider) GetTeamIDByName(ctx context.Context, leagueID, season int, name string) (int, error) {
	endpoint := fmt.Sprintf("%s/teams", p.BaseURL)
	params := url.Values{}
	params.Set("league", strconv.Itoa(leagueID))
	params.Set("season", strconv.Itoa(season))
	params.Set("search", name)

	var resp apiResponse[apiTeamResponse]
	if err := p.doRequest(ctx, endpoint, params, &resp); err != nil {
		return 0, err
	}

	if len(resp.Errors) > 0 {
		p.Logger.Error("API-Football retornou erros", zap.Strings("errors", resp.Errors))
		return 0, fmt.Errorf("football: erros da API: %v", resp.Errors)
	}

	if len(resp.Response) == 0 {
		return 0, fmt.Errorf("football: time '%s' não encontrado na liga %d temporada %d", name, leagueID, season)
	}

	// Procura match exato (case-insensitive)
	for _, t := range resp.Response {
		if strings.EqualFold(t.Team.Name, name) {
			return t.Team.ID, nil
		}
	}

	// Fallback: retorna o primeiro resultado
	return resp.Response[0].Team.ID, nil
}

// doRequest executa GET HTTP na API-Football com header x-apisports-key.
func (p *APIFootballProvider) doRequest(ctx context.Context, endpoint string, params url.Values, result interface{}) error {
	reqURL := endpoint + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		p.Logger.Error("Erro ao criar request", zap.Error(err), zap.String("url", reqURL))
		return fmt.Errorf("football: criar request: %w", err)
	}

	req.Header.Set("x-apisports-key", p.APIKey)
	req.Header.Set("Accept", "application/json")

	p.Logger.Debug("Chamando API-Football", zap.String("url", reqURL))

	resp, err := p.client.Do(req)
	if err != nil {
		p.Logger.Error("Erro na chamada HTTP", zap.Error(err), zap.String("url", reqURL))
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
