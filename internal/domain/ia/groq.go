package ia

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"

	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/history"
)

const (
	maxRetries       = 2
	baseBackoff      = 2 * time.Second
	maxBackoff       = 10 * time.Second
)

// RateLimitError indica que a API Groq retornou HTTP 429 (rate limited).
type RateLimitError struct {
	Model  string
	Reason string
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("Groq ainda tá de castigo (rate limit) — modelo %s: %s", e.Model, e.Reason)
}

// callGroq prepara o request IARequest e chama groqChat, retornando só o texto.
func callGroq(ctx context.Context, httpClient *http.Client, groqURL, groqKey, model string, messages []history.IAMessage, temperature float64, maxTokens int) (string, error) {
	resp, err := groqChat(ctx, httpClient, groqURL, groqKey, IARequest{
		Model:       model,
		Messages:    messages,
		Stream:      false,
		Temperature: temperature,
		MaxTokens:   maxTokens,
	})
	if err != nil {
		return "", err
	}
	return resp.Choices[0].Message.Content, nil
}

// groqChat é a camada pública que envolve groqChatRaw com retry em caso de 429.
func groqChat(ctx context.Context, httpClient *http.Client, groqURL, groqKey string, req IARequest) (IAResponse, error) {
	var lastErr error
	for attempt := range maxRetries + 1 {
		resp, err := groqChatRaw(ctx, httpClient, groqURL, groqKey, req)
		if err == nil {
			return resp, nil
		}

		var rateErr *RateLimitError
		if !errors.As(err, &rateErr) {
			return resp, err
		}

		lastErr = err

		if attempt < maxRetries {
			backoff := time.Duration(math.Min(
				float64(baseBackoff)*math.Pow(2, float64(attempt)),
				float64(maxBackoff),
			))
			select {
			case <-ctx.Done():
				return resp, ctx.Err()
			case <-time.After(backoff):
			}
		}
	}
	return IAResponse{}, fmt.Errorf("groq: rate limit excedido após %d tentativas: %w", maxRetries+1, lastErr)
}

// groqChatRaw faz a chamada HTTP real à API Groq, sem retry.
func groqChatRaw(ctx context.Context, httpClient *http.Client, groqURL, groqKey string, req IARequest) (IAResponse, error) {
	var zero IAResponse
	if groqURL == "" {
		return zero, fmt.Errorf("GROQ_URL não definido")
	}
	if groqKey == "" {
		return zero, fmt.Errorf("GROQ_API_KEY não definido")
	}

	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	body, err := json.Marshal(req)
	if err != nil {
		return zero, fmt.Errorf("groq: serializar request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, groqURL, bytes.NewBuffer(body))
	if err != nil {
		return zero, fmt.Errorf("groq: criar request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+groqKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return zero, fmt.Errorf("groq: http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusTooManyRequests {
			return zero, &RateLimitError{Model: req.Model, Reason: string(b)}
		}
		return zero, fmt.Errorf("groq: status %d: %s", resp.StatusCode, string(b))
	}

	var iaResp IAResponse
	if err := json.NewDecoder(resp.Body).Decode(&iaResp); err != nil {
		return zero, fmt.Errorf("groq: decodificar: %w", err)
	}
	if len(iaResp.Choices) == 0 {
		return zero, fmt.Errorf("groq: resposta sem choices")
	}
	return iaResp, nil
}
