// Package console sincroniza o estado dos comandos com o Shinobu Console.
//
// O bot roda em um app **sem domínio público** (SquareCloud só expõe
// websites), então ele não consegue ser chamado de fora. O caminho
// inverso funciona: o Console tem HTTPS público, então o bot **consulta**
// o Console para saber o estado desejado e **reporta** o que está
// aplicando.
//
// Sem CONSOLE_URL/CONSOLE_TOKEN configurados, o bot funciona sozinho
// (só perde o controle remoto pelo Console).
package console

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

// Estado é o conjunto de comandos desabilitados desejado (no Console).
type State struct {
	Disabled  []string `json:"disabled"`
	UpdatedAt string   `json:"updatedAt"`
}

// CatalogCommand descreve um comando para o Console exibir.
type CatalogCommand struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Private     bool   `json:"private"`
}

// Report é o que o bot manda ao Console: o catálogo de comandos e o
// estado que está efetivamente aplicado.
type Report struct {
	Catalog   []CatalogCommand `json:"catalog"`
	Disabled  []string         `json:"disabled"`
	AppliedAt string           `json:"appliedAt"`
}

// Client fala com o Console.
type Client struct {
	baseURL string
	token   string
	http    *http.Client
	log     *zap.Logger
}

// NewClient cria o cliente. Sem URL ou token, Enabled() é false.
func NewClient(baseURL, token string, log *zap.Logger) *Client {
	if log == nil {
		log = zap.NewNop()
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		http:    &http.Client{Timeout: 15 * time.Second},
		log:     log.Named("CONSOLE"),
	}
}

// Enabled informa se a sincronização está configurada.
func (c *Client) Enabled() bool {
	return c != nil && c.baseURL != "" && c.token != ""
}

// FetchState busca o estado desejado no Console.
func (c *Client) FetchState(ctx context.Context) (*State, error) {
	if !c.Enabled() {
		return nil, fmt.Errorf("console: sincronização não configurada")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v1/commands/state", nil)
	if err != nil {
		return nil, fmt.Errorf("console: criar requisição: %w", err)
	}
	req.Header.Set("X-Bot-Token", c.token)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("console: buscar estado: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("console: token inválido")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("console: status %d", resp.StatusCode)
	}

	var state State
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&state); err != nil {
		return nil, fmt.Errorf("console: decodificar estado: %w", err)
	}
	return &state, nil
}

// Report envia o catálogo e o estado aplicado para o Console.
func (c *Client) Report(ctx context.Context, report Report) error {
	if !c.Enabled() {
		return fmt.Errorf("console: sincronização não configurada")
	}

	body, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("console: codificar report: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/commands/state", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("console: criar requisição: %w", err)
	}
	req.Header.Set("X-Bot-Token", c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("console: reportar estado: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("console: token inválido")
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("console: status %d", resp.StatusCode)
	}
	return nil
}
