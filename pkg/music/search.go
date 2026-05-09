package music

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// cobaltRequest é o body enviado para a API do Cobalt.
type cobaltRequest struct {
	URL          string `json:"url"`
	DownloadMode string `json:"downloadMode"` // "audio" = só áudio, sem vídeo
	AudioFormat  string `json:"audioFormat"`  // "mp3", "opus", "ogg", "wav", "best"
	AudioBitrate string `json:"audioBitrate"` // kbps
}

// cobaltResponse é a resposta da API do Cobalt.
type cobaltResponse struct {
	Status   string `json:"status"`   // "tunnel", "redirect", "error"
	URL      string `json:"url"`      // URL do áudio para download
	Filename string `json:"filename"` // nome do arquivo gerado pelo Cobalt
}

// DownloadAudio baixa áudio a partir de uma URL direta (YouTube, SoundCloud, etc).
//
// Fluxo:
//  1. Bot envia POST para o Cobalt rodando no PC via Cloudflare Tunnel
//  2. Cobalt resolve o stream usando IP residencial (sem bloqueio de datacenter)
//  3. Cobalt retorna uma URL de túnel ou redirect para o áudio
//  4. Bot baixa os bytes diretamente dessa URL
//
// NOTA: o Cobalt não suporta busca por nome — apenas URLs diretas.
// O usuário deve enviar a URL completa (ex: https://youtube.com/watch?v=...).
//
// Configure a variável de ambiente COBALT_URL com a URL do tunnel.
// Exemplo: https://meu-tunnel.trycloudflare.com
//
// Retorna os bytes do arquivo e a extensão (ex: "mp3").
func DownloadAudio(ctx context.Context, query string) ([]byte, string, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, "", fmt.Errorf("music/download: query vazia")
	}

	// Cobalt só aceita URLs diretas — não suporta busca por nome.
	if !strings.HasPrefix(query, "http://") && !strings.HasPrefix(query, "https://") {
		return nil, "", fmt.Errorf("music/download: envie uma URL direta (ex: https://youtube.com/watch?v=...)")
	}

	// URL do Cobalt exposto via Cloudflare Tunnel.
	cobaltURL := os.Getenv("COBALT_URL")
	if cobaltURL == "" {
		return nil, "", fmt.Errorf("music/download: COBALT_URL não configurada")
	}

	endpoint := strings.TrimRight(cobaltURL, "/") + "/"

	reqBody := cobaltRequest{
		URL:          query,
		DownloadMode: "audio", // só áudio, sem vídeo
		AudioFormat:  "mp3",   // formato de saída
		AudioBitrate: "128",   // 128kbps — balanceia qualidade e velocidade no hardware fraco
	}

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, "", fmt.Errorf("music/download: erro ao serializar request: %w", err)
	}

	fmt.Println("[music] enviando para Cobalt:", endpoint)
	fmt.Println("[music] url:", query)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(reqBytes))
	if err != nil {
		return nil, "", fmt.Errorf("music/download: erro ao criar request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("music/download: erro ao chamar Cobalt: %w", err)
	}
	defer resp.Body.Close()

	var cobaltResp cobaltResponse
	if err := json.NewDecoder(resp.Body).Decode(&cobaltResp); err != nil {
		return nil, "", fmt.Errorf("music/download: erro ao parsear resposta do Cobalt: %w", err)
	}

	fmt.Println("[music] Cobalt status:", cobaltResp.Status)
	fmt.Println("[music] Cobalt url:", cobaltResp.URL)

	if cobaltResp.Status == "error" || cobaltResp.URL == "" {
		return nil, "", fmt.Errorf("music/download: Cobalt retornou erro para %q", query)
	}

	// Cobalt retorna "tunnel" ou "redirect" — em ambos baixamos direto da URL.
	dlReq, err := http.NewRequestWithContext(ctx, http.MethodGet, cobaltResp.URL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("music/download: erro ao criar request de download: %w", err)
	}

	dlResp, err := http.DefaultClient.Do(dlReq)
	if err != nil {
		return nil, "", fmt.Errorf("music/download: erro ao baixar áudio: %w", err)
	}
	defer dlResp.Body.Close()

	data, err := io.ReadAll(dlResp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("music/download: erro ao ler áudio: %w", err)
	}

	// Detecta extensão pelo filename retornado pelo Cobalt.
	ext := "mp3"
	if cobaltResp.Filename != "" {
		if e := filepath.Ext(cobaltResp.Filename); e != "" {
			ext = strings.TrimPrefix(e, ".")
		}
	}

	fmt.Println("[music] recebido:", len(data), "bytes, ext:", ext)

	return data, ext, nil
}
