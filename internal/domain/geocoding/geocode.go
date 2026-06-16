package geocoding

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"go.uber.org/zap"
)

// GeoCoding é o client de geocoding via Open-Meteo Geocoding API.
type GeoCoding struct {
	APIURL string
	Logger *zap.Logger
}

// GeoResult guarda as coordenadas e nome de um lugar.
type GeoResult struct {
	Latitude    float64
	Longitude   float64
	DisplayName string
	Country     string
	Timezone    string
}

// NewGeoCoding cria um novo client de geocoding.
// apiURL padrão: "https://geocoding-api.open-meteo.com/v1/search"
func NewGeoCoding(apiURL string, logger *zap.Logger) *GeoCoding {
	return &GeoCoding{APIURL: apiURL, Logger: logger}
}

// Lookup busca coordenadas para uma query de texto.
// limit define o número máximo de resultados (parâmetro "count" na API).
func (g *GeoCoding) Lookup(ctx context.Context, query string, limit int) ([]GeoResult, error) {
	params := url.Values{}
	params.Set("name", query)
	params.Set("count", fmt.Sprintf("%d", limit))
	params.Set("language", "pt")
	// params.Set("countrycode", "BR") // busca global

	fullURL := fmt.Sprintf("%s?%s", g.APIURL, params.Encode())

	g.Logger.Info("Fazendo request de geocoding", zap.String("query", query))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		g.Logger.Error("Erro ao criar request HTTP", zap.Error(err))
		return nil, fmt.Errorf("geocoding: erro ao criar request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		g.Logger.Error("Erro ao executar request HTTP", zap.Error(err))
		return nil, fmt.Errorf("geocoding: erro na requisição: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		g.Logger.Error("Geocoding retornou status inesperado",
			zap.Int("status", resp.StatusCode),
			zap.String("body", string(body)),
		)
		return nil, fmt.Errorf("geocoding: status %d", resp.StatusCode)
	}

	var raw struct {
		Results []struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
			Name      string  `json:"name"`
			Country   string  `json:"country"`
			Admin1    string  `json:"admin1"` // estado
			Timezone  string  `json:"timezone"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		g.Logger.Error("Erro ao decodificar JSON", zap.Error(err))
		return nil, fmt.Errorf("geocoding: erro ao decodificar resposta: %w", err)
	}

	if len(raw.Results) == 0 {
		g.Logger.Warn("Nenhum resultado encontrado", zap.String("query", query))
		return nil, nil
	}

	results := make([]GeoResult, 0, len(raw.Results))
	for _, r := range raw.Results {
		results = append(results, GeoResult{
			Latitude:    r.Latitude,
			Longitude:   r.Longitude,
			DisplayName: fmt.Sprintf("%s, %s", r.Name, r.Admin1),
			Country:     r.Country,
			Timezone:    r.Timezone,
		})
	}

	g.Logger.Info("Geocoding concluído", zap.Int("resultados", len(results)))
	return results, nil
}
