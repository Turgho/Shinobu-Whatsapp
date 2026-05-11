package geocoding

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
func (g *GeoCoding) Lookup(ctx context.Context, query string, limit int) ([]GeoResult, error) {
	fullURL := fmt.Sprintf("%s?q=%s&format=json&limit=%d", g.APIURL, url.QueryEscape(query), limit)

	g.Logger.Info("Fazendo request de geocoding", zap.String("query", query))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
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

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		g.Logger.Error("Geocoding retornou status inesperado",
			zap.Int("status", resp.StatusCode), zap.String("body", string(body)))
		return nil, fmt.Errorf("geocoding: status %d", resp.StatusCode)
	}

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
		lat, lon, ok := parseLatLon(r.Lat, r.Lon, g.Logger)
		if !ok {
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

func parseLatLon(latStr, lonStr string, log *zap.Logger) (lat, lon float64, ok bool) {
	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		log.Warn("Latitude inválida, ignorando resultado",
			zap.String("lat", latStr), zap.Error(err))
		return 0, 0, false
	}
	lon, err = strconv.ParseFloat(lonStr, 64)
	if err != nil {
		log.Warn("Longitude inválida, ignorando resultado",
			zap.String("lon", lonStr), zap.Error(err))
		return 0, 0, false
	}
	return lat, lon, true
}
