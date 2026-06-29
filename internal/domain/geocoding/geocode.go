package geocoding

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

type GeoCoding struct {
	NominatimURL string
	OpenMeteoURL string
	Logger       *zap.Logger
	httpClient   *http.Client
}

type GeoResult struct {
	Latitude    float64
	Longitude   float64
	DisplayName string
	Country     string
	Timezone    string
}

// NewGeoCoding cria client com timeout 10s. Nominatim é primário, Open-Meteo é fallback.
func NewGeoCoding(nominatimURL, openMeteoURL string, logger *zap.Logger) *GeoCoding {
	return &GeoCoding{
		NominatimURL: nominatimURL,
		OpenMeteoURL: openMeteoURL,
		Logger:       logger,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
	}
}

// Lookup busca coordenadas para uma query de lugar.
// Tenta Nominatim primeiro (mais preciso no Brasil); fallback para Open-Meteo.
func (g *GeoCoding) Lookup(ctx context.Context, query string, limit int) ([]GeoResult, error) {
	if results, err := g.lookupNominatim(ctx, query, limit); err == nil && len(results) > 0 {
		return results, nil
	}
	g.Logger.Warn("Nominatim falhou ou sem resultados, tentando Open-Meteo", zap.String("query", query))
	return g.lookupOpenMeteo(ctx, query, limit)
}

// lookupNominatim consulta Nominatim (OpenStreetMap).
// Monta DisplayName a partir de address.city/town/village + state — nunca usa display_name cru.
// User-Agent obrigatório: "Shinobu-Whatsapp/1.0".
func (g *GeoCoding) lookupNominatim(ctx context.Context, query string, limit int) ([]GeoResult, error) {
	params := url.Values{}
	params.Set("q", query)
	params.Set("format", "json")
	params.Set("limit", fmt.Sprintf("%d", limit))
	params.Set("addressdetails", "1")

	fullURL := fmt.Sprintf("%s?%s", g.NominatimURL, params.Encode())

	g.Logger.Info("Fazendo request de geocoding (Nominatim)", zap.String("query", query))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		g.Logger.Error("Erro ao criar request HTTP (Nominatim)", zap.Error(err))
		return nil, fmt.Errorf("geocoding: erro ao criar request: %w", err)
	}
	req.Header.Set("User-Agent", "Shinobu-Whatsapp/1.0")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		g.Logger.Error("Erro ao executar request HTTP (Nominatim)", zap.Error(err))
		return nil, fmt.Errorf("geocoding: erro na requisição: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		g.Logger.Error("Nominatim retornou status inesperado",
			zap.Int("status", resp.StatusCode),
			zap.String("body", string(body)),
		)
		return nil, fmt.Errorf("geocoding: status %d", resp.StatusCode)
	}

	var raw []struct {
		Lat         string `json:"lat"`
		Lon         string `json:"lon"`
		DisplayName string `json:"display_name"`
		Address     struct {
			Country string `json:"country"`
			State   string `json:"state"`
			City    string `json:"city"`
			Town    string `json:"town"`
			Village string `json:"village"`
		} `json:"address"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		g.Logger.Error("Erro ao decodificar JSON (Nominatim)", zap.Error(err))
		return nil, fmt.Errorf("geocoding: erro ao decodificar resposta: %w", err)
	}

	if len(raw) == 0 {
		g.Logger.Warn("Nenhum resultado encontrado (Nominatim)", zap.String("query", query))
		return nil, nil
	}

	results := make([]GeoResult, 0, len(raw))
	for _, r := range raw {
		lat, _ := strconv.ParseFloat(r.Lat, 64)
		lon, _ := strconv.ParseFloat(r.Lon, 64)

		country := r.Address.Country

		city := r.Address.City
		if city == "" {
			city = r.Address.Town
		}
		if city == "" {
			city = r.Address.Village
		}

		parts := []string{}
		if city != "" {
			parts = append(parts, city)
		}
		if r.Address.State != "" {
			parts = append(parts, r.Address.State)
		}
		displayName := strings.Join(parts, ", ")
		if displayName == "" {
			displayName = query
		}

		results = append(results, GeoResult{
			Latitude:    lat,
			Longitude:   lon,
			DisplayName: displayName,
			Country:     country,
		})
	}

	g.Logger.Info("Geocoding concluído (Nominatim)", zap.Int("resultados", len(results)))
	return results, nil
}

// lookupOpenMeteo fallback de geocoding via Open-Meteo.
// DisplayName montado como "Cidade, Estado" a partir de name + admin1.
func (g *GeoCoding) lookupOpenMeteo(ctx context.Context, query string, limit int) ([]GeoResult, error) {
	params := url.Values{}
	params.Set("name", query)
	params.Set("count", fmt.Sprintf("%d", limit))
	params.Set("language", "pt")

	fullURL := fmt.Sprintf("%s?%s", g.OpenMeteoURL, params.Encode())

	g.Logger.Info("Fazendo request de geocoding (Open-Meteo)", zap.String("query", query))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		g.Logger.Error("Erro ao criar request HTTP (Open-Meteo)", zap.Error(err))
		return nil, fmt.Errorf("geocoding: erro ao criar request: %w", err)
	}

	resp, err := g.httpClient.Do(req)
	if err != nil {
		g.Logger.Error("Erro ao executar request HTTP (Open-Meteo)", zap.Error(err))
		return nil, fmt.Errorf("geocoding: erro na requisição: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		g.Logger.Error("Open-Meteo retornou status inesperado",
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
			Admin1    string  `json:"admin1"`
			Timezone  string  `json:"timezone"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		g.Logger.Error("Erro ao decodificar JSON (Open-Meteo)", zap.Error(err))
		return nil, fmt.Errorf("geocoding: erro ao decodificar resposta: %w", err)
	}

	if len(raw.Results) == 0 {
		g.Logger.Warn("Nenhum resultado encontrado (Open-Meteo)", zap.String("query", query))
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

	g.Logger.Info("Geocoding concluído (Open-Meteo)", zap.Int("resultados", len(results)))
	return results, nil
}
