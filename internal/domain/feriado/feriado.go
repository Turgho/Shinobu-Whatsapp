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

// Feriado representa um feriado nacional brasileiro.
type Feriado struct {
	Date string // "2026-01-01"
	Name string
	Type string // "national", "religious", etc.
}

// apiItem mapeia o item retornado pela BrasilAPI.
type apiItem struct {
	Date string `json:"date"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// FeriadoClient busca feriados via BrasilAPI (pública, sem chave).
type FeriadoClient struct {
	APIURL     string
	httpClient *http.Client
	logger     *zap.Logger
}

func NewFeriadoClient(logger *zap.Logger) *FeriadoClient {
	return &FeriadoClient{
		APIURL: "https://brasilapi.com.br/api/feriados/v1",
		httpClient: &http.Client{
			Timeout: 8 * time.Second,
		},
		logger: logger,
	}
}

// Upcoming retorna os próximos N feriados a partir de hoje.
// Busca o ano atual e, se necessário, o próximo ano para completar N itens.
func (c *FeriadoClient) Upcoming(ctx context.Context, n int) ([]Feriado, error) {
	now := time.Now()
	today := now.Format("2006-01-02")

	items, err := c.fetchYear(ctx, now.Year())
	if err != nil {
		return nil, err
	}

	// Filtra apenas feriados futuros (date >= today) e ordena.
	items = filterFuture(items, today)

	// Se não tiver N, busca ano seguinte.
	if len(items) < n {
		next, err := c.fetchYear(ctx, now.Year()+1)
		if err != nil {
			// Se o próximo ano falhar, retorna o que temos.
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

func (c *FeriadoClient) fetchYear(ctx context.Context, year int) ([]Feriado, error) {
	url := fmt.Sprintf("%s/%d", c.APIURL, year)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("feriado: criar requisição: %w", err)
	}
	req.Header.Set("User-Agent", "Shinobu-Whatsapp/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("feriado: request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("feriado: status %d", resp.StatusCode)
	}

	var items []apiItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("feriado: decode JSON: %w", err)
	}

	result := make([]Feriado, 0, len(items))
	for _, item := range items {
		result = append(result, Feriado{
			Date: item.Date,
			Name: item.Name,
			Type: item.Type,
		})
	}

	return result, nil
}

// filterFuture filtra e ordena feriados com date >= today.
func filterFuture(items []Feriado, today string) []Feriado {
	var future []Feriado
	for _, f := range items {
		if f.Date >= today {
			future = append(future, f)
		}
	}
	sort.Slice(future, func(i, j int) bool {
		return future[i].Date < future[j].Date
	})
	return future
}
