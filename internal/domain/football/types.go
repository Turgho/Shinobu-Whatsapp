package football

import (
	"context"
	"time"

	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/configs"
)

// Constantes da Copa do Mundo 2026 (API-Football).
const (
	WorldCupLeagueID = 1
	WorldCupSeason   = 2026
)

// Team representa um time monitorado (nome, ID na API-Football, emoji da bandeira).
type Team struct {
	Name      string `json:"name"`        // Ex: "Brasil"
	APITeamID int    `json:"api_team_id"` // ID externo na API-Football
	Flag      string `json:"flag"`        // Emoji da bandeira, ex: "🇧🇷"
}

// Match representa uma partida de futebol.
type Match struct {
	ID          int       `json:"id"`
	HomeTeam    Team      `json:"home_team"`
	AwayTeam    Team      `json:"away_team"`
	HomeScore   int       `json:"home_score"`
	AwayScore   int       `json:"away_score"`
	Status      string    `json:"status"`       // Ex: "TBD", "LIVE", "FINISHED", "NS"
	StatusShort string    `json:"status_short"` // Ex: "1H", "HT", "2H", "ET", "FT", "PEN"
	StartTime   time.Time `json:"start_time"`
	ElapsedTime int       `json:"elapsed_time"` // Minutos decorridos (jogo ao vivo)
	League      string    `json:"league"`
	Season      int       `json:"season"`
	Round       string    `json:"round"`
	LastEventID int       `json:"last_event_id"` // Para deduplicação de gols
}

// GoalEvent representa um evento de gol/pênalti.
type GoalEvent struct {
	ID        int    `json:"id"`
	MatchID   int    `json:"match_id"`
	TeamID    int    `json:"team_id"`    // ID do time que marcou (para comparar com watched_team.api_team_id)
	TeamName  string `json:"team_name"`  // Nome do time que marcou
	Player    string `json:"player"`     // Nome do jogador
	Assist    string `json:"assist"`     // Nome do assistente (se houver)
	Minute    int    `json:"minute"`     // Minuto do gol (time.elapsed)
	ExtraTime int    `json:"extra_time"` // Acréscimos (time.extra)
	Type      string `json:"type"`       // "Goal" ou "Penalty"
	Detail    string `json:"detail"`     // Tipo: "Normal Goal", "Penalty", "Own Goal"
}

// TeamInfo representa informações básicas do time da API.
type TeamInfo struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Logo string `json:"logo"`
}

// Config carrega configuração do watcher (config.yaml → Viper → mapstructure).
type Config struct {
	Enabled      bool                       `mapstructure:"enabled"`
	APIKey       string                     `mapstructure:"api_key"`
	NotifyJID    string                     `mapstructure:"notify_jid"`
	PollInterval configs.PollIntervalConfig `mapstructure:"poll"`
	WatchedTeams []Team                     `mapstructure:"watched_teams"`
}

// ScoreProvider define interface para buscar dados de futebol.
// Implementação padrão: APIFootballProvider (chama api-football.com).
type ScoreProvider interface {
	// GetWorldCupFixturesForTeam busca jogos do Brasil na Copa 2026.
	// Chama GET /fixtures?league=1&season=2026&team={teamID}
	GetWorldCupFixturesForTeam(ctx context.Context, teamID int) ([]Match, error)

	// GetMatchDetails busca detalhes completos da partida (placar + eventos).
	// Chama GET /fixtures?id={fixture_id} - retorna fixture + events numa call.
	GetMatchDetails(ctx context.Context, fixtureID int) (*MatchDetails, error)

	// GetTeamIDByName busca o ID do time na liga/temporada.
	// Chama GET /teams?league=1&season=2026&search={name}
	GetTeamIDByName(ctx context.Context, leagueID, season int, name string) (int, error)
}

// MatchDetails contém dados completos da partida (retorno de GET /fixtures?id={id}).
type MatchDetails struct {
	Fixture Match       `json:"fixture"`
	Events  []GoalEvent `json:"events"`
}

// LiveStatus representa status considerados "ao vivo" na Copa 2026.
var LiveStatuses = []string{"1H", "HT", "2H", "ET", "BT", "P", "LIVE"}

// FinalStatus representa status que indicam fim da partida.
var FinalStatuses = []string{"FT", "AET", "PEN"}
