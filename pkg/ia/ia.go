package ia

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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

// AskIA monta o contexto completo (personalidade, histórico, web se necessário)
// e envia para o modelo adequado no Groq.
// Usa Scout 17B para conversa casual e 70B Versatile quando há contexto web.
// chat = chave da conversa (use o JID do privado ou do grupo).
func AskIA(ctx context.Context, chat, prompt string, isOwner bool, sender string, store *history.Store) (string, bool, error) {
	groqURL := os.Getenv("GROQ_URL")
	groqKey := os.Getenv("GROQ_API_KEY")

	prompt = cleanPrompt(prompt)

	mode := classifyPromptMode(prompt)

	// Monta mensagens com personalidade base da Shinobu
	messages := []history.IAMessage{
		{Role: "system", Content: buildSystemPrompt(mode)},
	}

	// Injeta contexto especial quando for o dono do bot
	if isOwner {
		messages = append(messages, history.IAMessage{
			Role:    "system",
			Content: "Você está falando com seu mestre. Seja calorosa e animada.",
		})
	}

	// Injeta resumo persistente da conversa, se existir
	if store != nil && chat != "" {
		if summary, err := store.GetSummary(ctx, chat); err == nil && strings.TrimSpace(summary) != "" {
			summary = truncateText(strings.TrimSpace(summary), 1200)
			messages = append(messages, history.IAMessage{
				Role: "system",
				Content: "Resumo persistente da conversa anterior:\n" +
					summary,
			})
		}
	}

	// Injeta histórico recente para manter continuidade da conversa
	if store != nil && chat != "" {
		if recent, err := store.RecentMessages(ctx, chat, 10, 12*time.Hour); err == nil {
			messages = append(messages, recent...)
		}
	}

	// Modelo e tokens padrão para conversa casual
	// Scout 17B: rápido e eficiente para respostas curtas do dia a dia
	model := "meta-llama/llama-4-scout-17b-16e-instruct"
	maxTokens := 150 // Suficiente para 2 frases em PT-BR com folga
	temperature := 0.7
	userContent := buildUserContent(prompt, mode, "")
	usedSearch := false

	// Decide dinamicamente se a pergunta precisa de busca web
	// Primeiro tenta keywords óbvias — sem chamada extra ao modelo
	// Se não encaixar, usa o Scout para classificar
	if shouldSearch(ctx, prompt) {
		if webContext, err := searchWeb(ctx, prompt); err == nil && webContext != "" {
			usedSearch = true
			mode = ModeWeb

			// Limita o contexto antes de injetar — evita prompt gigante mas mantém informação suficiente
			webContext = truncateText(webContext, 1500)

			// Histórico irrelevante quando tem contexto web — remove pra não confundir
			messages = []history.IAMessage{
				{Role: "system", Content: buildSystemPrompt(mode)},
			}
			if isOwner {
				messages = append(messages, history.IAMessage{
					Role:    "system",
					Content: "Você está falando com seu mestre. Seja calorosa e animada.",
				})
			}

			userContent = buildUserContent(prompt, mode, webContext)
			maxTokens = 600 // Espaço suficiente para respostas completas com contexto web
			temperature = 0.5
			model = "llama-3.3-70b-versatile"
		}
	}

	messages = append(messages, history.IAMessage{Role: "user", Content: userContent})

	answer, err := callGroq(ctx, groqURL, groqKey, model, messages, temperature, maxTokens)
	if err != nil {
		return "", usedSearch, err
	}

	// Atualiza o resumo da conversa em segundo plano para melhorar o contexto futuro
	if store != nil && chat != "" {
		go refreshChatSummary(chat, prompt, answer, store)
	}

	return answer, usedSearch, nil
}

// callGroq faz a chamada ao Groq e devolve o texto da primeira resposta.
func callGroq(ctx context.Context, groqURL, groqKey, model string, messages []history.IAMessage, temperature float64, maxTokens int) (string, error) {
	if groqURL == "" {
		return "", fmt.Errorf("GROQ_URL não definido")
	}
	if groqKey == "" {
		return "", fmt.Errorf("GROQ_API_KEY não definido")
	}

	body, err := json.Marshal(IARequest{
		Model:       model,
		Messages:    messages,
		Stream:      false,
		Temperature: temperature,
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
		return "", fmt.Errorf("erro ao chamar Groq: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("groq retornou status %d: %s", resp.StatusCode, string(b))
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
