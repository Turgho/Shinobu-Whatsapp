package geocoding

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"go.uber.org/zap"
)

// GeoCoding é o client de geocoding via Nominatim (OpenStreetMap).
type GeoCoding struct {
	APIURL string
	Logger *zap.Logger
}

// GeoResult guarda as coordenadas e nome de um lugar.
type GeoResult struct {
	Latitude    float64
	Longitude   float64
	DisplayName string
}

// NewGeoCoding cria um novo client de geocoding.
func NewGeoCoding(apiURL string, logger *zap.Logger) *GeoCoding {
	return &GeoCoding{APIURL: apiURL, Logger: logger}
}

// Lookup busca coordenadas para uma query de texto.
// limit define o número máximo de resultados.
func (g *GeoCoding) Lookup(query string, limit int) ([]GeoResult, error) {
	fullURL := fmt.Sprintf("%s?q=%s&format=json&limit=%d", g.APIURL, url.QueryEscape(query), limit)

	g.Logger.Info("Fazendo request de geocoding", zap.String("query", query))

	req, err := http.NewRequest(http.MethodGet, fullURL, nil)
	if err != nil {
		g.Logger.Error("Erro ao criar request HTTP", zap.Error(err))
		return nil, fmt.Errorf("geocoding: erro ao criar request: %w", err)
	}

	// Nominatim exige User-Agent identificado
	req.Header.Set("User-Agent", "YuukoBot/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		g.Logger.Error("Erro ao executar request HTTP", zap.Error(err))
		return nil, fmt.Errorf("geocoding: erro na requisição: %w", err)
	}
	defer resp.Body.Close()

	var raw []struct {
		Lat         string `json:"lat"`
		Lon         string `json:"lon"`
		DisplayName string `json:"display_name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		g.Logger.Error("Erro ao decodificar JSON", zap.Error(err))
		return nil, fmt.Errorf("geocoding: erro ao decodificar resposta: %w", err)
	}

	results := make([]GeoResult, 0, len(raw))
	for _, r := range raw {
		// strconv.ParseFloat reporta erros de forma confiável
		lat, err := strconv.ParseFloat(r.Lat, 64)
		if err != nil {
			g.Logger.Warn("Latitude inválida, ignorando resultado",
				zap.String("lat", r.Lat), zap.Error(err))
			continue
		}
		lon, err := strconv.ParseFloat(r.Lon, 64)
		if err != nil {
			g.Logger.Warn("Longitude inválida, ignorando resultado",
				zap.String("lon", r.Lon), zap.Error(err))
			continue
		}
		results = append(results, GeoResult{
			Latitude:    lat,
			Longitude:   lon,
			DisplayName: r.DisplayName,
		})
	}

	g.Logger.Info("Geocoding concluído", zap.Int("resultados", len(results)))
	return results, nil
}
