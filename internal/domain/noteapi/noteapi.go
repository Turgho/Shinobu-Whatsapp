// Package noteapi expõe o cliente HTTP para os endpoints do servidor de
// música que roda no notebook (mesma base e token do download: env
// MUSIC_SERVER_URL e MUSIC_API_TOKEN).
package noteapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

// Client conversa com os endpoints /stats e /admin/* do servidor de música.
type Client struct {
	serverURL string
	apiToken  string
	http      *http.Client
	logger    *zap.Logger
}

// NewClient constrói o cliente http e injeta o logger.
func NewClient(serverURL, apiToken string, logger *zap.Logger) *Client {
	return &Client{
		serverURL: strings.TrimRight(serverURL, "/"),
		apiToken:  apiToken,
		logger:    logger.Named("NOTEAPI"),
		http:      &http.Client{Timeout: 30 * time.Second},
	}
}

// StatusRef é o subconjunto de /stats usado nos comandos do bot.
type StatusRef struct {
	Uptime       string  `json:"uptime"`
	CPUUsage     float64 `json:"cpu_usage"`
	CPUCores     int     `json:"cpu_cores"`
	CPUTemp      string  `json:"cpu_temp"`
	RAMUsedMB    float64 `json:"ram_used_mb"`
	RAMTotalMB   float64 `json:"ram_total_mb"`
	DiskFreeGB   float64 `json:"disk_free_gb"`
	DiskTotalGB  float64 `json:"disk_total_gb"`
	WifiState    string  `json:"wifi_state"`
	WifiSignal   string  `json:"wifi_signal"`
	CacheEntries int     `json:"cache_entries"`
	CacheSizeMB  float64 `json:"cache_size_mb"`
}

// Status consulta GET /stats e decodifica o JSON de resposta.
func (c *Client) Status(ctx context.Context) (*StatusRef, error) {
	body, code, err := c.do(ctx, http.MethodGet, "/stats", 35*time.Second)
	if err != nil {
		return nil, err
	}
	if code != http.StatusOK {
		return nil, fmt.Errorf("noteapi: /stats retornou %d: %s", code, body)
	}
	var st StatusRef
	if err := json.Unmarshal([]byte(body), &st); err != nil {
		return nil, fmt.Errorf("noteapi: falha ao decodificar /stats: %w", err)
	}
	return &st, nil
}

// Action executa POST em um endpoint admin (/admin/...) e retorna o
// código HTTP e o corpo. Timeouts curtos: comandos de reboot/tunnel podem
// derrubar a conexão antes da resposta — nesse caso consideramos enviado.
func (c *Client) Action(ctx context.Context, path string, timeout time.Duration) (string, int, error) {
	body, code, err := c.do(ctx, http.MethodPost, path, timeout)
	if err != nil {
		// Conexão morta em operação destrutiva (reboot/tunnel) = comando
		// provavelmente executado; o chamador decide.
		return body, 0, err
	}
	return body, code, nil
}

// ClearCache executa DELETE /cache.
func (c *Client) ClearCache(ctx context.Context) (string, error) {
	if c.serverURL == "" || c.apiToken == "" {
		return "", fmt.Errorf("noteapi: MUSIC_SERVER_URL/MUSIC_API_TOKEN ausentes no .env do bot")
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.serverURL+"/cache/reset", nil)
	if err != nil {
		return "", fmt.Errorf("noteapi: erro ao criar requisição: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiToken)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("noteapi: falha ao limpar cache: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
	if code := resp.StatusCode; code != http.StatusOK {
		return string(body), fmt.Errorf("noteapi: /cache retornou %d: %s", code, body)
	}
	return string(body), nil
}

// do executa uma requisição autenticada e retorna corpo + código HTTP.
func (c *Client) do(ctx context.Context, method, path string, timeout time.Duration) (string, int, error) {
	if c.serverURL == "" || c.apiToken == "" {
		return "", 0, fmt.Errorf("noteapi: MUSIC_SERVER_URL/MUSIC_API_TOKEN ausentes no .env do bot")
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, c.serverURL+path, nil)
	if err != nil {
		return "", 0, fmt.Errorf("noteapi: erro ao criar requisição: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiToken)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("noteapi: requisição %s falhou: %w", method+path, err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
	return string(body), resp.StatusCode, nil
}
