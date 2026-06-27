// Package history guarda mensagens por chat e um resumo textual para contexto da IA.
package history

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/gosafe"
	"go.uber.org/zap"
	_ "modernc.org/sqlite"
)

// AssistantSenderName é o valor de `sender` gravado para respostas do bot (ver Save em shinobu).
// Precisa coincidir com o que o comando persiste para mapear papel "assistant" e rótulo na transcrição.
const AssistantSenderName = "Shinobu"

// Message é um registro persistido de chat (não confundir com IAMessage do fluxo Groq).
type Message struct {
	Sender string
	Text   string
	SentAt time.Time
}

// IAMessage representa uma mensagem no formato da API de chat (evita import cycle com internal/domain/ia).
type IAMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Store persiste mensagens, resumos e memória de usuário em SQLite.
type Store struct {
	db         *sql.DB
	UserMemory *UserMemoryStore
	log        *zap.Logger
}

// NewStore abre (ou cria) o arquivo SQLite e aplica o esquema mínimo de tabelas/índices.
func NewStore(path string, log *zap.Logger) (*Store, error) {
	if log == nil {
		log = zap.NewNop()
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("erro ao abrir history db: %w", err)
	}

	stmts := []string{
		`CREATE TABLE IF NOT EXISTS messages (
			id      INTEGER PRIMARY KEY AUTOINCREMENT,
			chat    TEXT NOT NULL,
			sender  TEXT NOT NULL,
			text    TEXT NOT NULL,
			sent_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS chat_summaries (
			chat       TEXT PRIMARY KEY,
			summary    TEXT NOT NULL,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_chat ON messages(chat)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_chat_sender ON messages(chat, sender)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_chat_sent_at ON messages(chat, sent_at)`,
	}
	for _, q := range stmts {
		if _, err := db.Exec(q); err != nil {
			return nil, fmt.Errorf("history schema: %w", err)
		}
	}

	memStore, err := NewUserMemoryStore(db, log)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar user memory store: %w", err)
	}

	return &Store{db: db, UserMemory: memStore, log: log}, nil
}

// StartCleanup apaga mensagens mais antigas que maxAge, a cada hora, até ctx cancelar.
func (s *Store) StartCleanup(ctx context.Context, maxAge time.Duration) {
	gosafe.Go(s.log, func() {
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
	})
}

// Save grava uma mensagem de chat.
func (s *Store) Save(ctx context.Context, chat, sender, text string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO messages (chat, sender, text) VALUES (?, ?, ?)`,
		chat, sender, text,
	)
	return err
}

// RecentMessages retorna até limit mensagens recentes dentro de maxAge, em ordem cronológica.
func (s *Store) RecentMessages(ctx context.Context, chat string, limit int, maxAge time.Duration) ([]IAMessage, error) {
	lines, err := s.loadRecentChatLines(ctx, chat, limit, maxAge)
	if err != nil {
		return nil, err
	}
	msgs := make([]IAMessage, 0, len(lines))
	for _, ln := range lines {
		role := "user"
		if ln.Sender == AssistantSenderName {
			role = "assistant"
		}
		msgs = append(msgs, IAMessage{Role: role, Content: ln.Text})
	}
	return msgs, nil
}

// TranscriptRecent formata mensagens recentes em texto com tempo relativo.
// Mensagens consecutivas do mesmo remetente são mescladas com " | " para
// reduzir overhead de formatação e tokens de metadados (label + timestamp).
// Ex: "[agora] Usuário: oi | tudo bem | viu o jogo"
// em vez de 3 linhas separadas.
func (s *Store) TranscriptRecent(ctx context.Context, chat string, limit int, maxAge time.Duration) (string, error) {
	lines, err := s.loadRecentChatLines(ctx, chat, limit, maxAge)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	var lastLabel string
	var lineBuf strings.Builder

	flush := func() {
		if lineBuf.Len() == 0 {
			return
		}
		b.WriteString(lineBuf.String())
		b.WriteByte('\n')
		lineBuf.Reset()
	}

	for _, ln := range lines {
		label := "Usuário"
		if ln.Sender == AssistantSenderName {
			label = AssistantSenderName
		}
		text := strings.TrimSpace(strings.ReplaceAll(ln.Text, "\n", " "))
		if text == "" {
			continue
		}
		ts := formatRelativeConversationTime(ln.At)

		if label != lastLabel {
			flush()
			lineBuf.WriteString(fmt.Sprintf("[%s] %s: %s", ts, label, text))
		} else {
			lineBuf.WriteString(" | ")
			lineBuf.WriteString(text)
		}
		lastLabel = label
	}
	flush()

	return strings.TrimSpace(b.String()), nil
}

type chatLine struct {
	Sender string
	Text   string
	At     time.Time
}

// loadRecentChatLines carrega as últimas linhas de conversa recentes.
func (s *Store) loadRecentChatLines(ctx context.Context, chat string, limit int, maxAge time.Duration) ([]chatLine, error) {
	if limit <= 0 {
		limit = 10
	}
	since := time.Now().Add(-maxAge).Format("2006-01-02 15:04:05")

	rows, err := s.db.QueryContext(ctx, `
		WITH ranked AS (
			SELECT id, sender, text, sent_at,
				ROW_NUMBER() OVER (ORDER BY sent_at DESC, id DESC) AS rn
			FROM messages
			WHERE chat = ? AND sent_at >= ?
		)
		SELECT sender, text, sent_at FROM ranked WHERE rn <= ?
		ORDER BY sent_at ASC, id ASC
	`, chat, since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []chatLine
	for rows.Next() {
		var sender, text, sentAtStr string
		if err := rows.Scan(&sender, &text, &sentAtStr); err != nil {
			return nil, err
		}
		out = append(out, chatLine{
			Sender: sender,
			Text:   text,
			At:     parseSQLiteSentAt(sentAtStr),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// parseSQLiteSentAt converte o timestamp SQLite para time.Time.
func parseSQLiteSentAt(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	for _, layout := range []string{
		"2006-01-02 15:04:05",
		time.RFC3339,
		"2006-01-02 15:04:05.999999999-07:00",
	} {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t
		}
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t
	}
	return time.Time{}
}

// formatRelativeConversationTime formata o tempo relativo da conversa.
func formatRelativeConversationTime(t time.Time) string {
	if t.IsZero() {
		return "?"
	}
	d := time.Since(t)
	if d < 0 {
		d = 0
	}
	switch {
	case d < time.Minute: // Se a diferença de tempo for menor que 1 minuto, retorna "agora"
		return "agora"
	case d < time.Hour: // Se a diferença de tempo for menor que 1 hora, retorna o número de minutos
		m := int(d / time.Minute)
		if m < 1 {
			m = 1
		}
		return fmt.Sprintf("há %dm", m)
	case d < 48*time.Hour: // Se a diferença de tempo for menor que 48 horas, retorna o número de horas
		h := int(d / time.Hour)
		if h < 1 {
			h = 1
		}
		return fmt.Sprintf("há %dh", h)
	default: // Se a diferença de tempo for maior que 48 horas, retorna a data e hora local
		return t.In(time.Local).Format("02/01 15:04")
	}
}

// GetSummary retorna o resumo persistido do chat ou string vazia se não houver.
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

// SetSummary grava ou atualiza o resumo do chat.
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

// CountRecentMessages conta mensagens no intervalo maxAge.
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

// NeedsSummary indica se já há mensagens suficientes para justificar regenerar o resumo.
func (s *Store) NeedsSummary(ctx context.Context, chat string, minMessages int, maxAge time.Duration) (bool, error) {
	count, err := s.CountRecentMessages(ctx, chat, maxAge)
	if err != nil {
		return false, err
	}

	return count >= minMessages, nil
}

// Close libera a conexão SQLite.
func (s *Store) Close() error {
	return s.db.Close()
}
