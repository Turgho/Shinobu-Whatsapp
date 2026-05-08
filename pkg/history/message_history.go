package history

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type Message struct {
	Sender string
	Text   string
	SentAt time.Time
}

// IAMessage definida aqui para evitar import circular com pkg/ia
type IAMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Store struct {
	db *sql.DB
}

func NewStore(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("erro ao abrir history db: %w", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS messages (
			id      INTEGER PRIMARY KEY AUTOINCREMENT,
			chat    TEXT NOT NULL,
			sender  TEXT NOT NULL,
			text    TEXT NOT NULL,
			sent_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar tabela: %w", err)
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_sender ON messages(sender)`)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar índice: %w", err)
	}

	return &Store{db: db}, nil
}

// Limpa mensagens muito antigas
func (s *Store) StartCleanup(ctx context.Context, maxAge time.Duration) {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.db.ExecContext(ctx,
					`DELETE FROM messages WHERE sent_at < ?`,
					time.Now().Add(-maxAge).Format("2006-01-02 15:04:05"),
				)
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (s *Store) Save(ctx context.Context, chat, sender, text string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO messages (chat, sender, text) VALUES (?, ?, ?)`,
		chat, sender, text,
	)
	return err
}

func (s *Store) RecentMessages(ctx context.Context, sender string, limit int, maxAge time.Duration) ([]IAMessage, error) {
	since := time.Now().Add(-maxAge).Format("2006-01-02 15:04:05")

	rows, err := s.db.QueryContext(ctx, `
		SELECT sender, text FROM (
			SELECT sender, text, sent_at FROM messages
			WHERE sender = ?
			AND sent_at >= ?
			ORDER BY sent_at DESC
			LIMIT ?
		) ORDER BY sent_at ASC
	`, sender, since, limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []IAMessage
	for rows.Next() {
		var sender, text string
		rows.Scan(&sender, &text)

		role := "user"
		if sender == "Shinobu" {
			role = "assistant"
		}
		messages = append(messages, IAMessage{Role: role, Content: text})
	}
	return messages, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}
