package cotacao

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

// CotacaoResult representa a cotação de uma moeda contra o real.
type CotacaoResult struct {
	Code      string  // "USD" ou "EUR"
	Name      string  // "Dólar Americano"
	Bid       float64 // preço de compra
	Ask       float64 // preço de venda
	High      float64 // máxima do dia
	Low       float64 // mínima do dia
	PctChange float64 // variação percentual
	UpdatedAt string  // data/hora da cotação
}

// apiResponse mapeia a resposta da AwesomeAPI.
type apiResponse map[string]struct {
	Code      string `json:"code"`
	Codein    string `json:"codein"`
	Name      string `json:"name"`
	High      string `json:"high"`
	Low       string `json:"low"`
	VarBid    string `json:"varBid"`
	PctChange string `json:"pctChange"`
	Bid       string `json:"bid"`
	Ask       string `json:"ask"`
	Timestamp string `json:"timestamp"`
	CreateDate string `json:"create_date"`
}

// CotacaoClient busca cotações via AwesomeAPI (pública, sem chave).
type CotacaoClient struct {
	APIURL     string
	httpClient *http.Client
	logger     *zap.Logger
}

// NewCotacaoClient cria um cliente com timeout de 8s.
func NewCotacaoClient(logger *zap.Logger) *CotacaoClient {
	return &CotacaoClient{
		// Suporta até 30 pares por requisição.
		APIURL: "https://economia.awesomeapi.com.br/json/last/USD-BRL,EUR-BRL",
		httpClient: &http.Client{
			Timeout: 8 * time.Second,
		},
		logger: logger,
	}
}

// Fetch busca as cotações de USD e EUR contra BRL em uma única chamada.
func (c *CotacaoClient) Fetch(ctx context.Context) ([]CotacaoResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.APIURL, nil)
	if err != nil {
		return nil, fmt.Errorf("cotacao: criar requisição: %w", err)
	}
	// Necessary to avoid 403 on some environments.
	req.Header.Set("User-Agent", "Shinobu-Whatsapp/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cotacao: request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cotacao: status %d", resp.StatusCode)
	}

	var raw apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("cotacao: decode JSON: %w", err)
	}

	currencies := []string{"USDBRL", "EURBRL"}
	result := make([]CotacaoResult, 0, 2)

	for _, key := range currencies {
		item, ok := raw[key]
		if !ok {
			c.logger.Warn("cotacao: moeda ausente na resposta", zap.String("key", key))
			continue
		}

		r := CotacaoResult{
			Code:      item.Code,
			Name:      parseName(item.Name),
			UpdatedAt: item.CreateDate,
			Bid:       parseFloat(c.logger, item.Bid),
			Ask:       parseFloat(c.logger, item.Ask),
			High:      parseFloat(c.logger, item.High),
			Low:       parseFloat(c.logger, item.Low),
			PctChange: parseFloat(c.logger, item.PctChange),
		}
		result = append(result, r)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("cotacao: nenhuma moeda retornada")
	}

	return result, nil
}

// parseFloat converte string para float64; loga warn e retorna 0 em erro.
func parseFloat(logger *zap.Logger, s string) float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		logger.Warn("cotacao: parse float", zap.String("value", s), zap.Error(err))
	}
	return v
}

// parseName extrai apenas a primeira parte do nome ("Dólar Americano/Real Brasileiro" → "Dólar Americano").
func parseName(name string) string {
	before, _, _ := strings.Cut(name, "/")
	return before
}
