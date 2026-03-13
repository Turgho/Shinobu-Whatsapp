package geocoding

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"go.uber.org/zap"
)

// GeoCoding representa o client de geocoding
type GeoCoding struct {
	APIURL string
	Logger *zap.Logger
}

// GeoResult guarda a coordenada de um lugar
type GeoResult struct {
	Latitude    float64
	Longitude   float64
	DisplayName string
}

// NewGeoCoding é o construtor com logger
func NewGeoCoding(apiURL string, logger *zap.Logger) *GeoCoding {
	return &GeoCoding{
		APIURL: apiURL,
		Logger: logger,
	}
}

// Lookup faz a busca por uma query, retorna coordenadas
func (g *GeoCoding) Lookup(query string, limit int) ([]GeoResult, error) {
	q := url.QueryEscape(query)
	fullURL := fmt.Sprintf("%s?q=%s&format=json&limit=%d", g.APIURL, q, limit)

	g.Logger.Info("Fazendo request de geocoding", zap.String("url", fullURL))

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		g.Logger.Error("Erro ao criar request HTTP", zap.Error(err))
		return nil, err
	}

	req.Header.Set("User-Agent", "YuukoBot/1.0 (victor.hugo3692111@gmail.com)")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		g.Logger.Error("Erro ao fazer request HTTP", zap.Error(err))
		return nil, err
	}
	defer resp.Body.Close()

	var results []struct {
		Lat         string `json:"lat"`
		Lon         string `json:"lon"`
		DisplayName string `json:"display_name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		g.Logger.Error("Erro ao decodificar JSON", zap.Error(err))
		return nil, err
	}

	geoResults := make([]GeoResult, len(results))
	for i, r := range results {
		var lat, lon float64
		fmt.Sscanf(r.Lat, "%f", &lat)
		fmt.Sscanf(r.Lon, "%f", &lon)
		geoResults[i] = GeoResult{
			Latitude:    lat,
			Longitude:   lon,
			DisplayName: r.DisplayName,
		}
	}

	g.Logger.Info("Resultados de geocoding obtidos", zap.Int("count", len(geoResults)))
	return geoResults, nil
}
