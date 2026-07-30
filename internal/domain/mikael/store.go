package mikael

import (
	"context"
	"database/sql"
	"strings"

	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/history"
	"go.mau.fi/whatsmeow/types/events"
	"go.uber.org/zap"
)

// trackedWord é a palavra monitorada nas mensagens.
const trackedWord = "pix"

// Store gerencia a contagem de palavras do Mikael: armazena mensagens de
// grupos-alvo e conta ocorrências de uma palavra por remetente específico.
// A contagem é persistida na tabela word_counts, que não é afetada pela
// limpeza periódica da tabela messages.
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

	s := &Store{
		historyStore: historyStore,
		db:           db,
		log:          log,
		groups:       g,
		targetSender: sender,
	}

	s.initSchema()
	s.migrateLegacyData()

	return s
}

// initSchema cria a tabela word_counts se não existir.
func (s *Store) initSchema() {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS word_counts (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			chat       TEXT NOT NULL,
			sender     TEXT NOT NULL,
			word       TEXT NOT NULL,
			count      INTEGER NOT NULL DEFAULT 0,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(chat, sender, word)
		)
	`)
	if err != nil {
		s.log.Error("Erro ao criar tabela word_counts", zap.Error(err))
	}
}

// migrateLegacyData faz uma migração única dos dados existentes na tabela
// messages para a word_counts, evitando perda do histórico após a introdução
// da tabela persistente.
func (s *Store) migrateLegacyData() {
	ctx := context.Background()

	for chat := range s.groups {
		var existing int
		err := s.db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM word_counts WHERE chat = ? AND sender = ? AND word = ?`,
			chat, s.targetSender, trackedWord,
		).Scan(&existing)
		if err != nil || existing > 0 {
			continue
		}

		rows, err := s.db.QueryContext(ctx,
			`SELECT text FROM messages WHERE chat = ? AND sender = ?`,
			chat, s.targetSender,
		)
		if err != nil {
			s.log.Warn("Erro ao consultar mensagens para migração",
				zap.String("chat", chat),
				zap.Error(err),
			)
			continue
		}

		var total int
		for rows.Next() {
			var text string
			if err := rows.Scan(&text); err != nil {
				rows.Close()
				s.log.Warn("Erro ao escanear mensagem na migração", zap.Error(err))
				return
			}
			total += countOccurrences(text, trackedWord)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			s.log.Warn("Erro ao iterar mensagens na migração",
				zap.String("chat", chat),
				zap.Error(err),
			)
			continue
		}

		if total > 0 {
			_, err = s.db.ExecContext(ctx,
				`INSERT INTO word_counts (chat, sender, word, count) VALUES (?, ?, ?, ?)`,
				chat, s.targetSender, trackedWord, total,
			)
			if err != nil {
				s.log.Warn("Erro ao inserir contagem migrada",
					zap.String("chat", chat),
					zap.Error(err),
				)
			}
		}
	}
}

// SaveMessageHook retorna uma função que salva mensagens de grupos-alvo no
// histórico e atualiza a contagem persistente da palavra monitorada.
func (s *Store) SaveMessageHook() func(ctx context.Context, evt *events.Message, msg string) {
	return func(ctx context.Context, evt *events.Message, msg string) {
		chat := evt.Info.Chat.String()
		if !s.groups[chat] {
			return
		}
		sender := evt.Info.Sender.ToNonAD().String()

		// Save to history as before (outros usos dependem da tabela messages)
		if err := s.historyStore.Save(ctx, chat, sender, msg); err != nil {
			s.log.Error("Erro ao salvar mensagem mikael",
				zap.String("chat", chat),
				zap.String("sender", sender),
				zap.Error(err),
			)
		}

		// Incrementa contagem persistente na word_counts apenas se o remetente for o alvo
		count := countOccurrences(msg, trackedWord)
		if count > 0 && sender == s.targetSender {
			_, err := s.db.ExecContext(ctx, `
				INSERT INTO word_counts (chat, sender, word, count, updated_at)
				VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
				ON CONFLICT(chat, sender, word) DO UPDATE SET
					count = count + excluded.count,
					updated_at = CURRENT_TIMESTAMP
			`, chat, sender, trackedWord, count)
			if err != nil {
				s.log.Error("Erro ao atualizar contagem persistente",
					zap.String("word", trackedWord),
					zap.String("chat", chat),
					zap.Error(err),
				)
			}
		}
	}
}

// CountWord retorna a contagem persistente de word para o remetente alvo no chat.
func (s *Store) CountWord(ctx context.Context, chat, word string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		`SELECT count FROM word_counts WHERE chat = ? AND sender = ? AND word = ?`,
		chat, s.targetSender, word,
	).Scan(&count)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return count, nil
}

// countOccurrences conta quantas vezes word aparece como palavra inteira em
// text, ignorando pontuação adjacente. Case-insensitive.
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
