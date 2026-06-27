package ia

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/history"
)

// Intent representa um comando identificado pela NLU.
type Intent struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

// DetectIntent classifica a mensagem em um comando disponível e extrai argumentos.
// Retorna Intent{Command: ""} se nenhum comando corresponder.
// Apenas comandos públicos da whitelist são retornados.
func DetectIntent(ctx context.Context, cfg *Config, message string) (*Intent, error) {
	if cfg.GroqURL == "" || cfg.GroqKey == "" {
		return &Intent{}, nil
	}

	systemPrompt := `Classifica mensagens em português em comandos de um bot WhatsApp. Retorne APENAS JSON válido, sem texto extra.

COMANDOS:
- clima <cidade>
- play <música>
- sticker
- efeito <tipo> [intensidade]
- aniversário <DD/MM|lista|remover>

Se não corresponder: {"command":"","args":[]}

Exemplos:
{"command":"clima","args":["São Paulo"]}
{"command":"play","args":["despacito"]}
{"command":"efeito","args":["reverb","leve"]}
{"command":"","args":[]}`

	req := IARequest{
		Model: modelScoutFast,
		Messages: []history.IAMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: fmt.Sprintf("Mensagem: %s", message)},
		},
		Temperature: 0,
		MaxTokens:   100,
		Stream:      false,
	}

	resp, err := groqChat(ctx, cfg.GroqURL, cfg.GroqKey, req)
	if err != nil {
		return nil, fmt.Errorf("detect intent: groq: %w", err)
	}
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("detect intent: sem choices")
	}

	content := strings.TrimSpace(resp.Choices[0].Message.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var intent Intent
	if err := json.Unmarshal([]byte(content), &intent); err != nil {
		return nil, fmt.Errorf("detect intent: parse JSON: %w", err)
	}

	intent.Command = strings.ToLower(intent.Command)
	return &intent, nil
}
