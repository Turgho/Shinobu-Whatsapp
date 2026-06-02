package history

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/gosafe"
)

// UserMemoryStore gerencia fatos extraídos sobre usuários.
// Cada fato é uma chave-valor simples (ex: "jogo favorito" = "Zelda")
// associada a um usuário dentro de um chat.
type UserMemoryStore struct {
	db *sql.DB
}

func NewUserMemoryStore(db *sql.DB) (*UserMemoryStore, error) {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS user_memory (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		chat       TEXT NOT NULL,
		user_jid   TEXT NOT NULL,
		key        TEXT NOT NULL,
		value      TEXT NOT NULL,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(chat, user_jid, key)
	)`)
	if err != nil {
		return nil, fmt.Errorf("user_memory schema: %w", err)
	}
	return &UserMemoryStore{db: db}, nil
}

// SaveFact salva ou atualiza um fato sobre um usuário.
func (m *UserMemoryStore) SaveFact(ctx context.Context, chat, userJID, key, value string) error {
	_, err := m.db.ExecContext(ctx, `
		INSERT INTO user_memory (chat, user_jid, key, value, updated_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(chat, user_jid, key) DO UPDATE SET
			value = excluded.value,
			updated_at = CURRENT_TIMESTAMP
	`, chat, userJID, key, value)
	return err
}

// GetFacts retorna todos os fatos conhecidos sobre um usuário no chat.
func (m *UserMemoryStore) GetFacts(ctx context.Context, chat, userJID string) (map[string]string, error) {
	rows, err := m.db.QueryContext(ctx, `
		SELECT key, value FROM user_memory
		WHERE chat = ? AND user_jid = ?
		ORDER BY updated_at DESC
	`, chat, userJID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	facts := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		facts[k] = v
	}
	return facts, rows.Err()
}

// FormatFacts formata os fatos como texto legível para incluir no prompt do sistema.
func FormatFacts(facts map[string]string) string {
	if len(facts) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("Fatos conhecidos sobre o usuário:\n")
	// ordem alfabética para consistência entre execuções
	keys := make([]string, 0, len(facts))
	for k := range facts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		b.WriteString("- ")
		b.WriteString(k)
		b.WriteString(": ")
		b.WriteString(facts[k])
		b.WriteString("\n")
	}
	return b.String()
}

// ExtractFactsFromPrompt tenta extrair fatos simples do prompt do usuário.
// Ex: "meu nome é João" → ("nome", "João")
// Usa padrões simples, sem chamar IA — o modelo principal já responde
// com base nesses fatos; nós só os armazenamos para persistência.
func ExtractFactsFromPrompt(prompt string) map[string]string {
	facts := make(map[string]string)
	lower := strings.ToLower(strings.TrimSpace(prompt))

	patterns := []struct {
		prefix string
		key    string
	}{
		{"meu nome é ", "nome"},
		{"me chamo ", "nome"},
		{"meu nome e ", "nome"},
		{"eu sou o ", "nome"},
		{"eu sou a ", "nome"},
		{"meu jogo favorito é ", "jogo favorito"},
		{"meu jogo favorito e ", "jogo favorito"},
		{"meu hobby é ", "hobby"},
		{"meu hobby e ", "hobby"},
		{"meu hobbie é ", "hobby"},
		{"meu hobbie e ", "hobby"},
		{"gosto de ", "gosto"},
		{"eu gosto de ", "gosto"},
		{"não gosto de ", "não gosto"},
		{"eu não gosto de ", "não gosto"},
		{"eu nao gosto de ", "não gosto"},
		{"minha idade é ", "idade"},
		{"minha idade e ", "idade"},
		{"tenho ", "idade"}, // "tenho 25 anos"
		{"moro em ", "cidade"},
		{"sou de ", "cidade"},
	}

	for _, p := range patterns {
		idx := strings.Index(lower, p.prefix)
		if idx < 0 {
			continue
		}
		rest := lower[idx+len(p.prefix):]
		// Pega até o primeiro espaço, ponto, vírgula, exclamação ou interrogação
		end := strings.IndexAny(rest, " .,!?;")
		if end < 0 {
			end = len(rest)
		}
		value := strings.TrimSpace(rest[:end])
		// Remove artigos/partículas no final
		value = strings.TrimSuffix(value, " mesmo")
		value = strings.TrimSuffix(value, " também")
		value = strings.TrimSuffix(value, " um")
		value = strings.TrimSuffix(value, " uma")
		value = strings.TrimSuffix(value, " o")
		value = strings.TrimSuffix(value, " a")
		if value != "" && len(value) < 60 {
			facts[p.key] = value
		}
	}

	// Extrai idade de "tenho X anos"
	if _, ok := facts["idade"]; !ok {
		idx := strings.Index(lower, "tenho ")
		if idx >= 0 {
			rest := lower[idx+6:]
			end := strings.IndexAny(rest, " .,!?;")
			if end < 0 {
				end = len(rest)
			}
			part := rest[:end]
			if strings.HasSuffix(part, "anos") || strings.HasSuffix(part, "ano") {
				age := strings.TrimSuffix(strings.TrimSuffix(part, "anos"), "ano")
				age = strings.TrimSpace(age)
				if age != "" {
					facts["idade"] = age
				}
			}
		}
	}

	return facts
}

// StartMemoryCleanup apaga memórias não atualizadas há mais de maxAge, a cada 24h.
func (m *UserMemoryStore) StartMemoryCleanup(ctx context.Context, maxAge time.Duration) {
	gosafe.Go(func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				_, _ = m.db.ExecContext(ctx,
					`DELETE FROM user_memory WHERE updated_at < ?`,
					time.Now().Add(-maxAge).Format("2006-01-02 15:04:05"),
				)
			case <-ctx.Done():
				return
			}
		}
	})
}
