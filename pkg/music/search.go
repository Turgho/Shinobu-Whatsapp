package music

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// DownloadAudio baixa áudio a partir de nome ou URL.
//
// O download é feito por um servidor local (music_server.go) rodando no PC
// via Cloudflare Tunnel — usando IP residencial para evitar o bloqueio do
// YouTube em servidores de datacenter (como Square Cloud).
//
// Fluxo:
//  1. Bot envia POST para o tunnel com a query (nome ou URL)
//  2. Servidor local executa o yt-dlp com ytsearch1 ou URL direta
//  3. Retorna os bytes do mp3
//
// Configure a variável de ambiente MUSIC_SERVER_URL com a URL do tunnel.
// Exemplo: https://meu-tunnel.trycloudflare.com
func DownloadAudio(ctx context.Context, query string) ([]byte, string, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, "", fmt.Errorf("music/download: query vazia")
	}

	// URL do servidor local exposto via Cloudflare Tunnel.
	serverURL := os.Getenv("MUSIC_SERVER_URL")
	if serverURL == "" {
		return nil, "", fmt.Errorf("music/download: MUSIC_SERVER_URL não configurada")
	}

	endpoint := strings.TrimRight(serverURL, "/") + "/play"

	fmt.Println("[music] query:", query)
	fmt.Println("[music] enviando para:", endpoint)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBufferString(query))
	if err != nil {
		return nil, "", fmt.Errorf("music/download: erro ao criar requisição: %w", err)
	}
	req.Header.Set("Content-Type", "text/plain")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("music/download: erro ao chamar servidor: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("music/download: servidor retornou %d: %s", resp.StatusCode, string(body))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("music/download: erro ao ler resposta: %w", err)
	}

	fmt.Println("[music] recebido:", len(data), "bytes")

	return data, "mp3", nil
}
