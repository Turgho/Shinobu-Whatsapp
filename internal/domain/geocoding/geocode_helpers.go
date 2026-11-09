// Helpers compartilhados pelos lookups de geocoding (Nominatim / Open-Meteo).
// Extraídos para permitir teste unitário sem rede.
package geocoding

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

// resolveGeoJSON valida o status HTTP e decodifica o JSON da resposta.
// Remove a duplicação de tratamento de erro entre os dois lookups.
func (g *GeoCoding) resolveGeoJSON(resp *http.Response, target any, source string) error {
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		g.Logger.Error("geocoding: status inesperado",
			zap.String("source", source),
			zap.Int("status", resp.StatusCode),
			zap.ByteString("body", body),
		)
		return fmt.Errorf("geocoding: status %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		g.Logger.Error("geocoding: erro ao decodificar resposta", zap.String("source", source), zap.Error(err))
		return fmt.Errorf("geocoding: erro ao decodificar resposta: %w", err)
	}
	return nil
}

// combineDisplayName monta "Cidade, Estado" ignorando partes vazias
// (evita "Cidade," com vírgula pendente quando o estado não veio).
// Fallback = valor cru passado pelo chamador quando não há cidade/estado.
func combineDisplayName(city, state, fallback string) string {
	parts := make([]string, 0, 2)
	if c := strings.TrimSpace(city); c != "" {
		parts = append(parts, c)
	}
	if s := strings.TrimSpace(state); s != "" {
		parts = append(parts, s)
	}
	if len(parts) == 0 {
		return strings.TrimSpace(fallback)
	}
	return strings.Join(parts, ", ")
}
