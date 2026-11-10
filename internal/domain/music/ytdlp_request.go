// Fallback remoto de download (notebook via túnel — LEGADO).
//
// Mantido só enquanto a migração local estiver em observação: o fluxo de
// !play pode cair aqui quando o executor local estiver desativado. Os
// comandos administrativos (!note/*) usam o MESMO par de envs
// (MUSIC_SERVER_URL / API_AUTH_TOKEN) via noteapi.
//
// Não use esse caminho como solução permanente: o notebook depende de
// Wi-Fi e do túnel Cloudflare, exatamente o que queremos eliminar.
package music

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// downloadViaTunnel envia a query por POST e devolve o MP3 inteiro em RAM.
// Timeout de 90s é o limite prático da resposta buffering do túnel.
func (s *Service) downloadViaTunnel(ctx context.Context, query string) ([]byte, string, error) {
	endpoint := strings.TrimRight(s.serverURL, "/") + "/play"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBufferString(query))
	if err != nil {
		return nil, "", &ErrDownload{Category: ErrUnknown,
			Detail: fmt.Sprintf("tunel: criar requisição: %v", err)}
	}
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Authorization", "Bearer "+s.apiToken)

	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", &ErrDownload{Category: ErrUnknown,
			Detail: fmt.Sprintf("tunel: servidor inacessível: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		return nil, "", &ErrDownload{Category: ErrUnknown,
			Detail: fmt.Sprintf("tunel: servidor retornou %d: %s", resp.StatusCode, body)}
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return nil, "", &ErrDownload{Category: ErrUnknown,
			Detail: fmt.Sprintf("tunel: ler resposta: %v", err)}
	}
	return data, "mp3", nil
}
