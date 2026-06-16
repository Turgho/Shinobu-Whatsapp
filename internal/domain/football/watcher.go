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

const (
	// preMatchWindow define quanto tempo antes do jogo o watcher entra em modo live.
	preMatchWindow = 30 * time.Minute
	// sendRetryAttempts define quantas tentativas de envio WhatsApp antes de desistir.
	sendRetryAttempts = 3
	// sendRetryBackoff define o tempo de espera entre tentativas de envio.
	sendRetryBackoff = 2 * time.Second
)

// Watcher monitora gols dos times configurados e envia notificação via WhatsApp.
type Watcher struct {
	client   *whatsmeow.Client
	cfg      *Config
	logger   *zap.Logger
	provider ScoreProvider
}

// NewWatcher cria watcher de gols.
func NewWatcher(client *whatsmeow.Client, cfg *Config, logger *zap.Logger, provider ScoreProvider) *Watcher {
	return &Watcher{
		client:   client,
		cfg:      cfg,
		logger:   logger,
		provider: provider,
	}
}

// Start inicia o loop de monitoramento em background (protegido por gosafe).
func (w *Watcher) Start(ctx context.Context) {
	gosafe.Go(func() {
		w.run(ctx)
	})
}

// run executa polling adaptativo para Copa 2026:
//   - Idle: busca jogos via GET /fixtures?league=1&season=2026&team={id}
//     entra em modo live se o jogo começa em menos de preMatchWindow ou já está rolando.
//   - Live: GET /fixtures?id={fixture_id} no intervalo configurado
//     (15s plano pago, 60s free tier) até status final (FT, AET, PEN).
func (w *Watcher) run(ctx context.Context) {
	w.logger.Info("Iniciando watcher de futebol (Copa 2026)",
		zap.Any("watched_teams", w.cfg.WatchedTeams),
		zap.String("notify_jid", w.cfg.NotifyJID))

	idleDuration, err := time.ParseDuration(w.cfg.PollInterval.IdleInterval)
	if err != nil {
		w.logger.Error("Erro ao parsear idle_interval, usando fallback 1h", zap.Error(err))
		idleDuration = time.Hour
	}
	liveDuration, err := time.ParseDuration(w.cfg.PollInterval.LiveInterval)
	if err != nil {
		w.logger.Error("Erro ao parsear live_interval, usando fallback 60s", zap.Error(err))
		liveDuration = 60 * time.Second
	}

	var currentFixtureID int
	var currentTeam Team
	var inLiveWindow bool

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Watcher de futebol encerrado")
			return
		default:
		}

		start := time.Now()

		if !inLiveWindow {
			// MODO IDLE: busca próximos jogos de TODOS os times monitorados na Copa 2026.
			for _, team := range w.cfg.WatchedTeams {
				fixtures, err := w.provider.GetWorldCupFixturesForTeam(ctx, team.APITeamID)
				if err != nil {
					w.logger.Error("Erro ao buscar jogos da Copa 2026",
						zap.Error(err), zap.String("team", team.Name))
					continue
				}

				for _, match := range fixtures {
					if w.isFinalStatus(match.StatusShort) {
						continue
					}

					// Só entra em live se o jogo começa em menos de preMatchWindow ou já está rolando.
					timeUntilStart := time.Until(match.StartTime)
					if timeUntilStart > preMatchWindow && !w.isLiveStatus(match.StatusShort) {
						w.logger.Info("Próximo jogo ainda distante - permanecendo idle",
							zap.String("team", team.Name),
							zap.String("opponent", match.AwayTeam.Name),
							zap.Duration("time_until_start", timeUntilStart),
							zap.Time("start_time", match.StartTime))
						continue
					}

					currentFixtureID = match.ID
					currentTeam = team
					inLiveWindow = true
					w.logger.Info("Jogo encontrado - entrando em modo live",
						zap.Int("fixture_id", match.ID),
						zap.String("status", match.StatusShort),
						zap.Time("start_time", match.StartTime),
						zap.String("round", match.Round),
						zap.String("team", team.Name))
					break
				}

				if inLiveWindow {
					break
				}
			}

			if !inLiveWindow {
				w.logger.Debug("Nenhum jogo ativo na Copa 2026 - modo idle")
			}
		}

		if inLiveWindow {
			// MODO LIVE: polling no fixture atual.
			// FIX: retorno de processLiveMatch verificado para sair do modo live ao fim da partida.
			finished := w.processLiveMatch(ctx, currentTeam, currentFixtureID)
			if !finished {
				w.logger.Info("Partida encerrada - voltando para modo idle",
					zap.Int("fixture_id", currentFixtureID))
				inLiveWindow = false
				currentFixtureID = 0
				currentTeam = Team{}
			}
		}

		currentDuration := idleDuration
		if inLiveWindow {
			currentDuration = liveDuration
		}

		elapsed := time.Since(start)
		if elapsed < currentDuration {
			select {
			case <-ctx.Done():
				w.logger.Info("Watcher de futebol encerrado")
				return
			case <-time.After(currentDuration - elapsed):
			}
		}
	}
}

// processLiveMatch processa partida em andamento.
// Chama GET /fixtures?id={fixture_id} para obter placar + events.
// Retorna true se a partida ainda está em andamento, false se terminou.
func (w *Watcher) processLiveMatch(ctx context.Context, team Team, fixtureID int) bool {
	details, err := w.provider.GetMatchDetails(ctx, fixtureID)
	if err != nil {
		w.logger.Error("Erro ao buscar detalhes da partida ao vivo",
			zap.Error(err), zap.Int("fixture_id", fixtureID))
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

	// Partida encerrada: loga placar final e sinaliza saída do modo live.
	if w.isFinalStatus(match.StatusShort) {
		w.logger.Info("Partida finalizada - saindo do modo live",
			zap.Int("fixture_id", match.ID),
			zap.String("final_status", match.StatusShort),
			zap.String("score", fmt.Sprintf("%s %d x %d %s",
				match.HomeTeam.Name, match.HomeScore,
				match.AwayScore, match.AwayTeam.Name)))
		return false
	}

	// Partida ainda não iniciou: aguarda sem processar eventos.
	if !w.isLiveStatus(match.StatusShort) {
		w.logger.Debug("Partida não iniciada ainda - aguardando",
			zap.Int("fixture_id", match.ID),
			zap.String("status", match.StatusShort))
		return true
	}

	// Lê último event ID processado (persistido em JSON para sobreviver a restart).
	lastEventID, err := GetLastEventID(match.ID)
	if err != nil {
		w.logger.Error("Erro ao obter último evento processado", zap.Error(err))
		return true
	}

	// Filtra apenas eventos novos (ID > último processado).
	var newEvents []GoalEvent
	for _, event := range events {
		if event.ID > lastEventID {
			newEvents = append(newEvents, event)
		}
	}

	if len(newEvents) == 0 {
		return true
	}

	for _, event := range newEvents {
		if event.Type != "Goal" && event.Type != "Penalty" {
			continue
		}

		// Gol a favor: marcado pelo Brasil OU gol contra do adversário.
		isBrazilGoal := event.TeamID == team.APITeamID ||
			(event.Detail == "Own Goal" && event.TeamID != team.APITeamID)
		if !isBrazilGoal {
			// Persiste mesmo gols do adversário para não reprocessar.
			if err := SetLastEventID(match.ID, event.ID); err != nil {
				w.logger.Error("Erro ao atualizar último evento processado", zap.Error(err))
			}
			continue
		}

		message := w.buildGoalMessage(team, match, event)
		w.sendWithRetry(ctx, message, event, team)

		// Persiste último event ID para deduplicação.
		if err := SetLastEventID(match.ID, event.ID); err != nil {
			w.logger.Error("Erro ao atualizar último evento processado", zap.Error(err))
		}
	}

	return true
}

// sendWithRetry tenta enviar a notificação WhatsApp até sendRetryAttempts vezes.
func (w *Watcher) sendWithRetry(ctx context.Context, message string, event GoalEvent, team Team) {
	jid, err := types.ParseJID(w.cfg.NotifyJID)
	if err != nil {
		w.logger.Error("Erro ao parsear JID para notificação", zap.Error(err))
		return
	}

	for attempt := 1; attempt <= sendRetryAttempts; attempt++ {
		ctxSend, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		err = whatsapp.SendTextToJID(ctxSend, w.client, jid, message, nil)
		cancel()

		if err == nil {
			w.logger.Info("Notificação de gol enviada",
				zap.String("team", team.Name),
				zap.String("player", event.Player),
				zap.String("detail", event.Detail),
				zap.String("minute", fmt.Sprintf("%d+%d", event.Minute, event.ExtraTime)),
				zap.String("assist", event.Assist))
			return
		}

		w.logger.Warn("Falha ao enviar notificação de gol, tentando novamente",
			zap.Error(err),
			zap.Int("attempt", attempt),
			zap.Int("max_attempts", sendRetryAttempts))

		if attempt < sendRetryAttempts {
			select {
			case <-ctx.Done():
				return
			case <-time.After(sendRetryBackoff):
			}
		}
	}

	w.logger.Error("Falha ao enviar notificação de gol após todas as tentativas",
		zap.String("player", event.Player),
		zap.Int("minute", event.Minute))
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
// Para gol contra: "🇧🇷🇧🇷 GOOOL DO BRASIL! Brasil 1 x 0 Croácia - 🇧🇷 Gol contra aos 45+1'"
func (w *Watcher) buildGoalMessage(team Team, match Match, event GoalEvent) string {
	var opponentTeam string
	var teamScore, opponentScore int

	isHomeGoal := event.TeamID == match.HomeTeam.APITeamID ||
		(event.Detail == "Own Goal" && event.TeamID == match.AwayTeam.APITeamID)
	if isHomeGoal {
		opponentTeam = match.AwayTeam.Name
		teamScore = match.HomeScore
		opponentScore = match.AwayScore
	} else {
		opponentTeam = match.HomeTeam.Name
		teamScore = match.AwayScore
		opponentScore = match.HomeScore
	}

	minuteStr := fmt.Sprintf("%d", event.Minute)
	if event.ExtraTime > 0 {
		minuteStr = fmt.Sprintf("%d+%d", event.Minute, event.ExtraTime)
	}

	flag := team.Flag
	if flag == "" {
		flag = "⚽"
	}

	playerName := event.Player
	if event.Detail == "Own Goal" {
		playerName = "Gol contra"
	}

	assistStr := ""
	if event.Assist != "" {
		assistStr = fmt.Sprintf(" (assist: %s)", event.Assist)
	}

	// FIX: fmt.Sprintf sem índices posicionais para evitar confusão de argumentos.
	return fmt.Sprintf("%s%s GOOOL DO %s! %s %d x %d %s - %s %s aos %s'%s",
		flag, flag, team.Name, team.Name, teamScore, opponentScore,
		opponentTeam, flag, playerName, minuteStr, assistStr)
}
