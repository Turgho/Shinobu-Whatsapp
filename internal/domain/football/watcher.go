package football

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/gosafe"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.uber.org/zap"
)

// Watcher monitora gols dos times configurados e envia notificação via WhatsApp.
type Watcher struct {
	client    *whatsmeow.Client
	cfg       *Config
	logger    *zap.Logger
	provider  ScoreProvider
	notifyJID string
}

// NewWatcher cria watcher de gols.
func NewWatcher(client *whatsmeow.Client, cfg *Config, logger *zap.Logger, provider ScoreProvider) *Watcher {
	return &Watcher{
		client:    client,
		cfg:       cfg,
		logger:    logger,
		provider:  provider,
		notifyJID: cfg.NotifyJID,
	}
}

// Start inicia o loop de monitoramento em background (protegido por gosafe).
func (w *Watcher) Start(ctx context.Context) {
	gosafe.Go(func() {
		w.run(ctx)
	})
}

// run executa polling adaptativo para Copa 2026:
//   - Idle: 1x/dia (ou few horas) via GET /fixtures?league=1&season=2026&team={id}
//     descobre fixture_id e horário do próximo jogo do Brasil
//   - Live: GET /fixtures?id={fixture_id} no intervalo configurado
//     (15s plano pago, 60s free tier) até status final (FT, AET, PEN)
func (w *Watcher) run(ctx context.Context) {
	w.logger.Info("Iniciando watcher de futebol (Copa 2026)",
		zap.Any("watched_teams", w.cfg.WatchedTeams),
		zap.String("notify_jid", w.notifyJID))

	idleDuration, err := time.ParseDuration(w.cfg.PollInterval.IdleInterval)
	if err != nil {
		w.logger.Error("Erro ao parsear idle_interval", zap.Error(err))
		idleDuration = 24 * time.Hour // fallback: 1x/dia
	}
	liveDuration, err := time.ParseDuration(w.cfg.PollInterval.LiveInterval)
	if err != nil {
		w.logger.Error("Erro ao parsear live_interval", zap.Error(err))
		liveDuration = 60 * time.Second // fallback: free tier 1 call/min
	}

	var currentFixtureID int
	var currentTeam Team
	var inLiveWindow bool
	currentDuration := idleDuration

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Watcher de futebol encerrado")
			return
		default:
			start := time.Now()

			if !inLiveWindow {
				// MODO IDLE: busca próximos jogos de TODOS os times monitorados na Copa 2026
				foundMatch := false
				for _, team := range w.cfg.WatchedTeams {
					fixtures, err := w.provider.GetWorldCupFixturesForTeam(ctx, team.APITeamID)
					if err != nil {
						w.logger.Error("Erro ao buscar jogos da Copa 2026", zap.Error(err), zap.String("team", team.Name))
						continue
					}

					// Procura jogo que ainda não terminou (status != FT, AET, PEN)
					for _, match := range fixtures {
						if w.isFinalStatus(match.StatusShort) {
							continue
						}

						// Encontrou jogo relevante (próximo ou em andamento)
						currentFixtureID = match.ID
						currentTeam = team
						inLiveWindow = true
						foundMatch = true
						w.logger.Info("Jogo encontrado - entrando em modo live",
							zap.Int("fixture_id", match.ID),
							zap.String("status", match.StatusShort),
							zap.Time("start_time", match.StartTime),
							zap.String("round", match.Round),
							zap.String("team", team.Name))
						break
					}

					if foundMatch {
						break
					}
				}

				if !inLiveWindow {
					w.logger.Debug("Nenhum jogo ativo na Copa 2026 - modo idle")
				}
			}

			if inLiveWindow {
				// MODO LIVE: polling no fixture atual via GET /fixtures?id={id}
				w.processLiveMatch(ctx, currentTeam, currentFixtureID)

				// Verifica se partida terminou
				// (o processLiveMatch atualiza inLiveWindow via retorno)
			}

			// Ajusta intervalo
			if inLiveWindow {
				currentDuration = liveDuration
			} else {
				currentDuration = idleDuration
			}

			elapsed := time.Since(start)
			if elapsed < currentDuration {
				time.Sleep(currentDuration - elapsed)
			}
		}
	}
}

// processLiveMatch processa partida em andamento.
// Chama GET /fixtures?id={fixture_id} para obter placar + events.
// Retorna false se partida terminou (para sair do modo live).
func (w *Watcher) processLiveMatch(ctx context.Context, team Team, fixtureID int) bool {
	details, err := w.provider.GetMatchDetails(ctx, fixtureID)
	if err != nil {
		w.logger.Error("Erro ao buscar detalhes da partida ao vivo", zap.Error(err), zap.Int("fixture_id", fixtureID))
		return true // continua tentando
	}

	match := details.Fixture
	events := details.Events

	w.logger.Debug("Processando partida ao vivo",
		zap.Int("fixture_id", match.ID),
		zap.String("status", match.StatusShort),
		zap.Int("home_score", match.HomeScore),
		zap.Int("away_score", match.AwayScore),
		zap.Int("elapsed", match.ElapsedTime))

	// Verifica se partida terminou
	if w.isFinalStatus(match.StatusShort) {
		w.logger.Info("Partida finalizada - saindo do modo live",
			zap.Int("fixture_id", match.ID),
			zap.String("final_status", match.StatusShort),
			zap.String("score", fmt.Sprintf("%d x %d", match.HomeScore, match.AwayScore)))
		return false // sai do modo live
	}

	// Verifica se está em status "ao vivo"
	if !w.isLiveStatus(match.StatusShort) {
		w.logger.Debug("Partida não iniciada ainda - aguardando",
			zap.Int("fixture_id", match.ID),
			zap.String("status", match.StatusShort))
		return true // continua no modo live, aguardando início
	}

	// Lê último event ID processado (persistido em JSON)
	lastEventID, err := GetLastEventID(match.ID)
	if err != nil {
		w.logger.Error("Erro ao obter último evento processado", zap.Error(err))
		return true
	}

	// Filtra apenas eventos novos (ID > último processado)
	var newEvents []GoalEvent
	for _, event := range events {
		if event.ID > lastEventID {
			newEvents = append(newEvents, event)
		}
	}

	if len(newEvents) == 0 {
		return true
	}

	// Processa cada evento novo: verifica se é gol DO Brasil (team.id == brazil_id)
	for _, event := range newEvents {
		// Considera apenas gols e pênaltis
		if event.Type != "Goal" && event.Type != "Penalty" {
			continue
		}

		// Verifica se o gol é A FAVOR do Brasil:
		// - event.TeamID == team.APITeamID (gol direto do Brasil)
		// - OU: event.Detail == "Own Goal" E event.TeamID != team.APITeamID (gol contra do adversário)
		isBrazilGoal := event.TeamID == team.APITeamID ||
			(event.Detail == "Own Goal" && event.TeamID != team.APITeamID)

		if !isBrazilGoal {
			continue
		}

		message := w.buildGoalMessage(team, match, event)

		// Envia notificação WhatsApp
		ctxSend, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		jid, err := types.ParseJID(w.notifyJID)
		if err != nil {
			w.logger.Error("Erro ao parsear JID para notificação", zap.Error(err))
			cancel()
			continue
		}
		err = whatsapp.SendTextToJID(ctxSend, w.client, jid, message, nil)
		cancel()
		if err != nil {
			w.logger.Error("Erro ao enviar notificação de gol", zap.Error(err))
		} else {
			w.logger.Info("Notificação de gol enviada",
				zap.String("team", team.Name),
				zap.String("player", event.Player),
				zap.String("detail", event.Detail),
				zap.Int("minute", event.Minute+event.ExtraTime),
				zap.String("assist", event.Assist))
		}

		// Persiste último event ID para deduplicação (sobrevive a restart)
		if err := SetLastEventID(match.ID, event.ID); err != nil {
			w.logger.Error("Erro ao atualizar último evento processado", zap.Error(err))
		}
	}

	return true // continua no modo live
}

// isLiveStatus verifica se status indica partida em andamento.
// Status ao vivo na Copa 2026: 1H, HT, 2H, ET, BT, P, LIVE
func (w *Watcher) isLiveStatus(status string) bool {
	for _, s := range LiveStatuses {
		if strings.EqualFold(s, status) {
			return true
		}
	}
	return false
}

// isFinalStatus verifica se status indica fim da partida.
// Status finais: FT, AET, PEN
func (w *Watcher) isFinalStatus(status string) bool {
	for _, s := range FinalStatuses {
		if strings.EqualFold(s, status) {
			return true
		}
	}
	return false
}

// buildGoalMessage monta mensagem de gol.
// Ex: "🇧🇷🇧🇷 GOOOL DO BRASIL! Brasil 1 x 0 Croácia - 🇧🇷 Neymar aos 23' (assist: Vini Jr.)"
// Para gol contra: "🇧🇷🇧🇷 GOOOL DO BRASIL! Brasil 1 x 0 Croácia - 🇧🇷 Gol contra (adversário) aos 45+1'"
func (w *Watcher) buildGoalMessage(team Team, match Match, event GoalEvent) string {
	var opponentTeam string
	var opponentScore int
	var teamScore int

	// Determina placar e adversário baseado em quem marcou
	isHomeGoal := event.TeamID == match.HomeTeam.APITeamID || (event.Detail == "Own Goal" && event.TeamID == match.AwayTeam.APITeamID)
	if isHomeGoal {
		opponentTeam = match.AwayTeam.Name
		teamScore = match.HomeScore
		opponentScore = match.AwayScore
	} else {
		opponentTeam = match.HomeTeam.Name
		teamScore = match.AwayScore
		opponentScore = match.HomeScore
	}

	// Formata minuto: 23 ou 45+1
	minuteStr := fmt.Sprintf("%d", event.Minute)
	if event.ExtraTime > 0 {
		minuteStr = fmt.Sprintf("%d+%d", event.Minute, event.ExtraTime)
	}

	flag := team.Flag
	if flag == "" {
		flag = "⚽"
	}

	// Nome do jogador ou "Gol contra" para own goal
	playerName := event.Player
	if event.Detail == "Own Goal" {
		playerName = "Gol contra"
	}

	assistStr := ""
	if event.Assist != "" {
		assistStr = fmt.Sprintf(" (assist: %s)", event.Assist)
	}

	return fmt.Sprintf("%[1]s%[1]s GOOOL DO %[3]s! %[3]s %[5]d x %[6]d %[7]s - %[1]s %[9]s aos %[10]s%[11]s",
		flag, flag, team.Name, team.Name, teamScore, opponentScore, opponentTeam, flag, playerName, minuteStr, assistStr)
}
