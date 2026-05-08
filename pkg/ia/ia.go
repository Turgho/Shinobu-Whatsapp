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
	Model    string              `json:"model"`
	Messages []history.IAMessage `json:"messages"`
	Stream   bool                `json:"stream"`
}

type IAResponse struct {
	Message history.IAMessage `json:"message"`
}

var keywords = []string{
	"pesquis", "busca", "buscas", "procur",
	"notíci", "notici", "novidade", "news",
	"hoje", "agora", "atual", "recente", "última", "ultimo",
	"quanto", "quem", "onde", "quando", "qual", "como",
	"tempo", "clima", "previsão",
	"preço", "valor", "custa", "custo",
}

func AskIA(ctx context.Context, prompt string, isOwner bool, sender string, store *history.Store) (string, error) {
	ollamaURL := os.Getenv("OLLAMA_URL")

	prompt = cleanPrompt(prompt)

	// Monta mensagens com histórico
	messages := []history.IAMessage{}
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
	userContent := prompt
	if containsAny(strings.ToLower(prompt), keywords) {
		if webContext, err := searchWeb(ctx, prompt); err == nil && webContext != "" {
			userContent = fmt.Sprintf("Contexto:\n%s\n\nMensagem: %s", webContext, prompt)
		}
	}

	messages = append(messages, history.IAMessage{Role: "user", Content: userContent})

	body, err := json.Marshal(IARequest{
		Model:    "shinobu",
		Messages: messages,
		Stream:   false,
	})
	if err != nil {
		return "", fmt.Errorf("erro ao serializar request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ollamaURL, bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("erro ao criar request: %w", err)
	}
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

	return iaResp.Message.Content, nil
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
