package mikael

import (
	"context"
	"database/sql"
	"strings"

	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/history"
	"go.mau.fi/whatsmeow/types/events"
	"go.uber.org/zap"
)

// Store gerencia a contagem de palavras do Mikael: armazena mensagens de
// grupos-alvo e conta ocorrências de uma palavra por remetente específico.
type Store struct {
	historyStore *history.Store
	db           *sql.DB
	log          *zap.Logger
	groups       map[string]bool
	targetSender string // JID completo (ex: 46549275017279@lid)
}

// NewStore cria uma Store. groups são os JIDs de grupo onde monitorar mensagens;
// lid é o identificador do Mikael (com ou sem domínio).
func NewStore(historyStore *history.Store, db *sql.DB, groups []string, lid string, log *zap.Logger) *Store {
	g := make(map[string]bool, len(groups))
	for _, grp := range groups {
		g[grp] = true
	}
	sender := lid
	if !strings.Contains(sender, "@") {
		sender = sender + "@lid"
	}
	return &Store{
		historyStore: historyStore,
		db:           db,
		log:          log,
		groups:       g,
		targetSender: sender,
	}
}

// SaveMessageHook retorna uma função que salva mensagens de grupos-alvo no
// histórico, para ser registrada como hook no router.
func (s *Store) SaveMessageHook() func(ctx context.Context, evt *events.Message, msg string) {
	return func(ctx context.Context, evt *events.Message, msg string) {
		chat := evt.Info.Chat.String()
		if !s.groups[chat] {
			return
		}
		sender := evt.Info.Sender.ToNonAD().String()
		if err := s.historyStore.Save(ctx, chat, sender, msg); err != nil {
			s.log.Error("Erro ao salvar mensagem mikael",
				zap.String("chat", chat),
				zap.String("sender", sender),
				zap.Error(err),
			)
		}
	}
}

// CountWord conta quantas vezes word aparece como palavra inteira nas
// mensagens do Mikael no chat. Pontuação adjacente (.,!? etc) é ignorada.
func (s *Store) CountWord(ctx context.Context, chat, word string) (int, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT text FROM messages WHERE chat = ? AND sender = ?
	`, chat, s.targetSender)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	lower := strings.ToLower(word)
	var total int
	for rows.Next() {
		var text string
		if err := rows.Scan(&text); err != nil {
			return 0, err
		}
		total += countOccurrences(text, lower)
	}
	return total, rows.Err()
}

// countOccurrences conta quantas vezes word (lowercase) aparece como palavra
// inteira em text, ignorando pontuação adjacente.
func countOccurrences(text, word string) int {
	var count int
	for _, field := range strings.Fields(text) {
		field = strings.Trim(field, ".,!?;:()[]{}'\"-")
		if strings.EqualFold(field, word) {
			count++
		}
	}
	return count
}
