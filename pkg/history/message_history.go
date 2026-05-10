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
		return nil, fmt.Errorf("erro ao criar tabela messages: %w", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS chat_summaries (
			chat       TEXT PRIMARY KEY,
			summary    TEXT NOT NULL,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar tabela chat_summaries: %w", err)
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_messages_chat ON messages(chat)`)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar índice idx_messages_chat: %w", err)
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_messages_chat_sender ON messages(chat, sender)`)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar índice idx_messages_chat_sender: %w", err)
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_messages_chat_sent_at ON messages(chat, sent_at)`)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar índice idx_messages_chat_sent_at: %w", err)
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
				_, _ = s.db.ExecContext(ctx,
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

func (s *Store) RecentMessages(ctx context.Context, chat string, limit int, maxAge time.Duration) ([]IAMessage, error) {
	if limit <= 0 {
		limit = 10
	}

	since := time.Now().Add(-maxAge).Format("2006-01-02 15:04:05")

	rows, err := s.db.QueryContext(ctx, `
		SELECT sender, text FROM (
			SELECT sender, text, sent_at FROM messages
			WHERE chat = ?
			AND sent_at >= ?
			ORDER BY sent_at DESC
			LIMIT ?
		) ORDER BY sent_at ASC
	`, chat, since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []IAMessage
	for rows.Next() {
		var sender, text string
		if err := rows.Scan(&sender, &text); err != nil {
			return nil, err
		}

		role := "user"
		if sender == "Shinobu" {
			role = "assistant"
		}

		messages = append(messages, IAMessage{
			Role:    role,
			Content: text,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}

func (s *Store) GetSummary(ctx context.Context, chat string) (string, error) {
	var summary string
	err := s.db.QueryRowContext(ctx,
		`SELECT summary FROM chat_summaries WHERE chat = ?`,
		chat,
	).Scan(&summary)

	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}

	return summary, nil
}

func (s *Store) SetSummary(ctx context.Context, chat, summary string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO chat_summaries (chat, summary, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(chat) DO UPDATE SET
			summary = excluded.summary,
			updated_at = CURRENT_TIMESTAMP
	`, chat, summary)
	return err
}

func (s *Store) CountRecentMessages(ctx context.Context, chat string, maxAge time.Duration) (int, error) {
	since := time.Now().Add(-maxAge).Format("2006-01-02 15:04:05")

	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM messages
		WHERE chat = ?
		AND sent_at >= ?
	`, chat, since).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (s *Store) NeedsSummary(ctx context.Context, chat string, minMessages int, maxAge time.Duration) (bool, error) {
	count, err := s.CountRecentMessages(ctx, chat, maxAge)
	if err != nil {
		return false, err
	}

	return count >= minMessages, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}
