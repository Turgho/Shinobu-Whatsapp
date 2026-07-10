package feriado

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"go.uber.org/zap"
)

// Feriado representa um feriado retornado pela Feriados API.
type Feriado struct {
	Data     string `json:"data"` // "DD/MM/YYYY"
	Nome     string `json:"nome"`
	Tipo     string `json:"tipo"` // NACIONAL, ESTADUAL, MUNICIPAL, FACULTATIVO
	Bancario bool   `json:"bancario"`
}

// apiResponse mapeia o wrapper retornado pela Feriados API.
type apiResponse struct {
	Ano      string    `json:"ano"`
	Feriados []apiItem `json:"feriados"`
	Meta     struct {
		Total int `json:"total"`
	} `json:"meta"`
}

type apiItem struct {
	Data     string `json:"data"`
	Nome     string `json:"nome"`
	Tipo     string `json:"tipo"`
	Bancario bool   `json:"bancario"`
}

// FeriadosClient busca feriados via Feriados API (requer Bearer token).
type FeriadosClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	logger     *zap.Logger
}

// NewFeriadosClient cria um cliente para a Feriados API.
// A base URL deve ser "https://feriadosapi.com".
func NewFeriadosClient(baseURL, apiKey string, logger *zap.Logger) *FeriadosClient {
	return &FeriadosClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 8 * time.Second,
		},
		logger: logger,
	}
}

// Upcoming retorna os próximos N feriados a partir de hoje.
// Se uf for fornecido (ex: "SP"), busca feriados estaduais + nacionais daquele estado.
// Se uf for vazio, busca apenas feriados nacionais.
func (c *FeriadosClient) Upcoming(ctx context.Context, n int, uf string) ([]Feriado, error) {
	now := time.Now()

	items, err := c.fetchYear(ctx, now.Year(), uf)
	if err != nil {
		return nil, err
	}

	// Filtra apenas feriados futuros.
	today := now.Format("02/01/2006")
	items = filterFuture(items, today)

	// Se não tiver N, busca ano seguinte.
	if len(items) < n {
		next, err := c.fetchYear(ctx, now.Year()+1, uf)
		if err != nil {
			c.logger.Warn("feriado: erro ao buscar ano seguinte", zap.Error(err))
		} else {
			items = append(items, next...)
		}
	}

	if len(items) > n {
		items = items[:n]
	}

	if len(items) == 0 {
		return []Feriado{}, nil
	}

	return items, nil
}

func (c *FeriadosClient) fetchYear(ctx context.Context, year int, uf string) ([]Feriado, error) {
	var url string
	if uf != "" {
		url = fmt.Sprintf("%s/api/v1/feriados/estado/%s?ano=%d", c.baseURL, uf, year)
	} else {
		url = fmt.Sprintf("%s/api/v1/feriados/nacionais?ano=%d", c.baseURL, year)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("feriado: criar requisição: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("User-Agent", "Shinobu-Whatsapp/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("feriado: request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("feriado: status %d", resp.StatusCode)
	}

	var apiResp apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("feriado: decode JSON: %w", err)
	}

	result := make([]Feriado, 0, len(apiResp.Feriados))
	for _, item := range apiResp.Feriados {
		result = append(result, Feriado{
			Data:     item.Data,
			Nome:     item.Nome,
			Tipo:     item.Tipo,
			Bancario: item.Bancario,
		})
	}

	return result, nil
}

// filterFuture filtra e ordena feriados com data >= today.
// A data vem no formato "DD/MM/YYYY" da API.
func filterFuture(items []Feriado, today string) []Feriado {
	var future []Feriado
	for _, f := range items {
		if f.Data >= today {
			future = append(future, f)
		}
	}
	sort.Slice(future, func(i, j int) bool {
		return future[i].Data < future[j].Data
	})
	return future
}
