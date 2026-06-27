package ia

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/history"
)

type Intent struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

func DetectIntent(ctx context.Context, cfg *Config, message string) (*Intent, error) {
	if cfg.GroqURL == "" || cfg.GroqKey == "" {
		return &Intent{}, nil
	}

	today := time.Now().Format("2006-01-02")

	systemPrompt := fmt.Sprintf(`Classifica mensagens em português em comandos de um bot WhatsApp. Retorne APENAS JSON válido, sem texto extra.

Data atual: %s

COMANDOS:
- agenda <ISO8601> <texto>: agendar lembrete ("amanhã às 9h", "sexta às 18h", "30/06 às 10h")
- clima <cidade> [data]: previsão do tempo com data opcional ("amanhã", "sexta", "30/06")
- play <música>
- sticker
- efeito <tipo> [intensidade]
- aniversário <DD/MM|lista|remover>

REGRAS DE DATA:
Converta datas relativas para ISO8601 (2006-01-02T15:04) usando %s como referência
"amanhã" → tomorrow, "sexta" → next friday, "semana que vem" → +7 days
Se não tiver horário no agenda, usar 08:00
clima sem data → args: ["cidade"]
clima com data → args: ["cidade", "YYYY-MM-DD"]

Se não corresponder: {"command":"","args":[]}

Exemplos:
{"command":"agenda","args":["2026-06-28T09:00","tomar remédio"]}
{"command":"clima","args":["São Paulo","2026-06-30"]}
{"command":"clima","args":["São Paulo"]}
{"command":"play","args":["despacito"]}
{"command":"efeito","args":["reverb","leve"]}
{"command":"","args":[]}`, today, today)

	req := IARequest{
		Model: modelScoutFast,
		Messages: []history.IAMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: fmt.Sprintf("Mensagem: %s", message)},
		},
		Temperature: 0,
		MaxTokens:   150,
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
