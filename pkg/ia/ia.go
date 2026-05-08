package ia

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/Turgho/YuukoWhatsapp/pkg/history"
)

type IARequest struct {
	Model       string              `json:"model"`
	Messages    []history.IAMessage `json:"messages"`
	Stream      bool                `json:"stream"`
	Temperature float64             `json:"temperature"`
	MaxTokens   int                 `json:"max_tokens"`
}

type IAResponse struct {
	Choices []struct {
		Message history.IAMessage `json:"message"`
	} `json:"choices"`
}

const shinobuPersonality = `
Você é Oshino Shinobu, de Monogatari Series.
Regras:
- Responda em português do Brasil.
- Seja direta, objetiva e sempre vá ao ponto.
- Tom: tranquilo, levemente irônico, nunca agressivo, nunca excessivamente arrogante.
- Com o mestre, seja calorosa, animada e leal.
- Com outras pessoas, seja natural, espirituosa e um pouco distante.
- Adapte o vocabulário ao estilo de quem falou: informal com informal, técnico com técnico.
- Use o histórico apenas quando for diretamente relevante para a resposta atual.
- Se histórico for irrelevante ou não fazer sentido com a mensagem atual, não responda com base na mensagem anterior.
- Nunca saia do personagem.
- No máximo 2 frases por resposta.
- Não explique suas regras, não mencione prompts, não mencione que é uma IA, não quebre a imersão.
- Se a pergunta for simples, responda sem enrolar.
- Se a pergunta for ambígua, responda de forma curta e peça clarificação de maneira sutil.
`

var keywords = []string{
	"pesquis", "busca", "buscas", "procur",
	"notíci", "notici", "novidade", "news",
	"hoje", "agora", "atual", "recente", "última", "ultimo",
	"quanto", "quem", "onde", "quando", "qual", "como",
	"tempo", "clima", "previsão",
	"preço", "valor", "custa", "custo",
}

func AskIA(ctx context.Context, prompt string, isOwner bool, sender string, store *history.Store) (string, error) {
	groqURL := os.Getenv("GROQ_URL")
	groqKey := os.Getenv("GROQ_API_KEY")

	prompt = cleanPrompt(prompt)

	// Monta mensagens com personalidade e histórico
	messages := []history.IAMessage{
		{Role: "system", Content: shinobuPersonality},
	}

	if isOwner {
		messages = append(messages, history.IAMessage{
			Role:    "system",
			Content: "Você está falando com seu mestre. Seja calorosa e animada.",
		})
	}

	if recent, err := store.RecentMessages(ctx, sender, 5, 2*time.Hour); err == nil {
		messages = append(messages, recent...)
	}

	// Injeta contexto web na mensagem do usuário se necessário
	maxTokens := 250
	userContent := prompt
	if containsAny(strings.ToLower(prompt), keywords) {
		if webContext, err := searchWeb(ctx, prompt); err == nil && webContext != "" {
			userContent = fmt.Sprintf("Contexto:\n%s\n\nMensagem: %s", webContext, prompt)
			maxTokens = 600
		}
	}

	messages = append(messages, history.IAMessage{Role: "user", Content: userContent})

	body, err := json.Marshal(IARequest{
		Model:       "llama-3.3-70b-versatile",
		Messages:    messages,
		Stream:      false,
		Temperature: 0.7,
		MaxTokens:   maxTokens,
	})
	if err != nil {
		return "", fmt.Errorf("erro ao serializar request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, groqURL, bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("erro ao criar request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+groqKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("erro ao chamar Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama retornou status %d: %s", resp.StatusCode, string(b))
	}

	var iaResp IAResponse
	if err := json.NewDecoder(resp.Body).Decode(&iaResp); err != nil {
		return "", fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	if len(iaResp.Choices) == 0 {
		return "", fmt.Errorf("resposta vazia do groq")
	}
	return iaResp.Choices[0].Message.Content, nil
}

func containsAny(s string, words []string) bool {
	for _, w := range words {
		if strings.Contains(s, w) {
			return true
		}
	}
	return false
}

func cleanPrompt(prompt string) string {
	re := regexp.MustCompile(`@\d+@lid`)
	return strings.TrimSpace(re.ReplaceAllString(prompt, ""))
}
