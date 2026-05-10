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
...
- Quando tiver contexto de pesquisa disponível, priorize as informações dele na resposta — mas mantenha seu tom e estilo.
- Nunca invente informações; se não souber algo, diga de forma direta e no personagem.
`

// AskIA monta o contexto completo (personalidade, histórico, web se necessário)
// e envia para o modelo adequado no Groq.
// Usa Scout 17B para conversa casual e 70B Versatile quando há contexto web.
func AskIA(ctx context.Context, prompt string, isOwner bool, sender string, store *history.Store) (string, bool, error) {
	groqURL := os.Getenv("GROQ_URL")
	groqKey := os.Getenv("GROQ_API_KEY")

	prompt = cleanPrompt(prompt)

	// Monta mensagens com personalidade base da Shinobu
	messages := []history.IAMessage{
		{Role: "system", Content: shinobuPersonality},
	}

	// Injeta contexto especial quando for o dono do bot
	if isOwner {
		messages = append(messages, history.IAMessage{
			Role:    "system",
			Content: "Você está falando com seu mestre. Seja calorosa e animada.",
		})
	}

	// Injeta histórico recente para manter continuidade da conversa (máx 5 mensagens, janela de 2h)
	if recent, err := store.RecentMessages(ctx, sender, 5, 2*time.Hour); err == nil {
		messages = append(messages, recent...)
	}

	// Modelo e tokens padrão para conversa casual
	// Scout 17B: rápido e eficiente para respostas curtas do dia a dia
	model := "meta-llama/llama-4-scout-17b-16e-instruct"
	maxTokens := 150 // Suficiente para 2 frases em PT-BR com folga
	userContent := prompt
	usedSearch := false

	// Decide dinamicamente se a pergunta precisa de busca web
	// Usa uma chamada leve ao Scout para classificar, evitando keyword matching frágil
	if shouldSearch(ctx, prompt) {
		if webContext, err := searchWeb(ctx, prompt); err == nil && webContext != "" {
			usedSearch = true

			// Limita o contexto antes de injetar — evita resposta longa
			if len(webContext) > 600 {
				webContext = webContext[:600]
			}

			// Histórico irrelevante quando tem contexto web — remove pra não confundir
			messages = []history.IAMessage{
				{Role: "system", Content: shinobuPersonality},
			}
			if isOwner {
				messages = append(messages, history.IAMessage{
					Role:    "system",
					Content: "Você está falando com seu mestre. Seja calorosa e animada.",
				})
			}

			userContent = fmt.Sprintf(
				"Contexto:\n%s\n\nMensagem: %s\n\nIMPORTANTE: Responda em no máximo 2 frases curtas.",
				webContext, prompt,
			)
			maxTokens = 300
			model = "llama-3.3-70b-versatile"
		}
	}

	messages = append(messages, history.IAMessage{Role: "user", Content: userContent})

	body, err := json.Marshal(IARequest{
		Model:       model,
		Messages:    messages,
		Stream:      false,
		Temperature: 0.7,
		MaxTokens:   maxTokens,
	})
	if err != nil {
		return "", usedSearch, fmt.Errorf("erro ao serializar request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, groqURL, bytes.NewBuffer(body))
	if err != nil {
		return "", usedSearch, fmt.Errorf("erro ao criar request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+groqKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", usedSearch, fmt.Errorf("erro ao chamar Groq: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", usedSearch, fmt.Errorf("groq retornou status %d: %s", resp.StatusCode, string(b))
	}

	var iaResp IAResponse
	if err := json.NewDecoder(resp.Body).Decode(&iaResp); err != nil {
		return "", usedSearch, fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	if len(iaResp.Choices) == 0 {
		return "", usedSearch, fmt.Errorf("resposta vazia do groq")
	}

	return iaResp.Choices[0].Message.Content, usedSearch, nil
}

// shouldSearch faz uma chamada leve ao Scout para decidir se a pergunta
// precisa de informações atuais da internet. Retorna true se sim.
// MaxTokens=5 e Temperature=0 garantem resposta mínima e determinística.
func shouldSearch(ctx context.Context, prompt string) bool {
	groqURL := os.Getenv("GROQ_URL")
	groqKey := os.Getenv("GROQ_API_KEY")

	body, err := json.Marshal(IARequest{
		Model: "meta-llama/llama-4-scout-17b-16e-instruct",
		Messages: []history.IAMessage{
			{
				Role: "system",
				Content: "Responda apenas 'sim' ou 'nao', sem pontuação. " +
					"Esta pergunta precisa de informações atuais ou específicas da internet para ser respondida corretamente?",
			},
			{Role: "user", Content: prompt},
		},
		Stream:      false,
		Temperature: 0, // Determinístico: sem criatividade na classificação
		MaxTokens:   5, // Só precisa de "sim" ou "nao"
	})
	if err != nil {
		return false
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, groqURL, bytes.NewBuffer(body))
	if err != nil {
		return false
	}
	req.Header.Set("Authorization", "Bearer "+groqKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	var iaResp IAResponse
	if err := json.NewDecoder(resp.Body).Decode(&iaResp); err != nil {
		return false
	}

	if len(iaResp.Choices) == 0 {
		return false
	}

	answer := strings.ToLower(strings.TrimSpace(iaResp.Choices[0].Message.Content))
	return strings.HasPrefix(answer, "sim")
}

// cleanPrompt remove menções do WhatsApp (@número@lid) e espaços extras do prompt
func cleanPrompt(prompt string) string {
	re := regexp.MustCompile(`@\d+@lid`)
	return strings.TrimSpace(re.ReplaceAllString(prompt, ""))
}
