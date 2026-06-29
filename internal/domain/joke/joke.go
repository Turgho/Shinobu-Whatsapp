package joke

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Joke representa uma piada retornada pela JokeAPI.
type Joke struct {
	Type     string // "single" ou "twopart"
	Setup    string // pergunta (twopart)
	Delivery string // resposta (twopart)
	Joke     string // piada completa (single)
}

// JokeClient busca piadas via JokeAPI (pública, sem chave).
type JokeClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

// NewJokeClient cria um cliente com timeout de 8s.
func NewJokeClient(baseURL string, logger *zap.Logger) *JokeClient {
	return &JokeClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 8 * time.Second,
		},
		logger: logger,
	}
}

// jokeResponse mapeia a resposta da JokeAPI.
type jokeResponse struct {
	Error    bool   `json:"error"`
	Type     string `json:"type"`
	Setup    string `json:"setup"`
	Delivery string `json:"delivery"`
	Joke     string `json:"joke"`
	Safe     bool   `json:"safe"`
}

// Ping verifica se a JokeAPI está disponível.
func (c *JokeClient) Ping(ctx context.Context) error {
	url := c.baseURL + "/ping"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("joke: criar requisição ping: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("joke: ping: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("joke: ping status %d", resp.StatusCode)
	}

	return nil
}

// Fetch busca uma piada aleatória em PT-BR.
func (c *JokeClient) Fetch(ctx context.Context) (*Joke, error) {
	url := c.baseURL + "/joke/Any?lang=pt"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("joke: criar requisição: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("joke: request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("joke: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("joke: ler body: %w", err)
	}

	var raw jokeResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("joke: decode JSON: %w", err)
	}

	if raw.Error || !raw.Safe {
		return nil, fmt.Errorf("joke: piada não segura ou erro da API")
	}

	j := &Joke{Type: raw.Type}
	switch raw.Type {
	case "twopart":
		j.Setup = raw.Setup
		j.Delivery = raw.Delivery
	case "single":
		j.Joke = raw.Joke
	default:
		return nil, fmt.Errorf("joke: tipo desconhecido %q", raw.Type)
	}

	return j, nil
}
