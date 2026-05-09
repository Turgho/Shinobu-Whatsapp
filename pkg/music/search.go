package music

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// deezerSearchResult representa a resposta da API pública do Deezer.
type deezerSearchResult struct {
	Data []struct {
		ID int64 `json:"id"`
	} `json:"data"`
}

// searchDeezer busca uma faixa no Deezer por nome e retorna a URL da primeira encontrada.
// Usa a API pública gratuita — sem autenticação, sem bloqueio de IP.
func searchDeezer(query string) (string, error) {
	apiURL := "https://api.deezer.com/search?q=" + url.QueryEscape(query)

	resp, err := http.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("deezer: erro na requisição: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("deezer: erro ao ler resposta: %w", err)
	}

	var result deezerSearchResult
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("deezer: erro ao parsear JSON: %w", err)
	}

	if len(result.Data) == 0 {
		return "", fmt.Errorf("deezer: nenhuma faixa encontrada para %q", query)
	}

	trackURL := fmt.Sprintf("https://www.deezer.com/track/%d", result.Data[0].ID)
	return trackURL, nil
}

// DownloadAudio baixa áudio temporário a partir de nome ou URL.
//
// Fluxo para texto (nome da música):
//  1. Busca no Deezer via API pública (sem bloqueio de IP em servidores)
//  2. Obtém a URL da faixa (ex: https://www.deezer.com/track/123456)
//  3. Passa a URL para o yt-dlp baixar
//
// NOTA: ytsearch1 (YouTube) foi removido pois o YouTube bloqueia IPs de
// datacenter (como Square Cloud) exigindo PO Token e JS runtime, tornando
// o download inviável em produção. O Deezer resolve isso sem restrições.
//
// Fluxo para URL direta:
//   - Suporta URLs do Deezer (https://www.deezer.com/track/...)
//   - URLs do YouTube ainda falharão por bloqueio de IP no servidor
//
// Retorna os bytes do arquivo final e a extensão gerada (ex: "mp3").
func DownloadAudio(ctx context.Context, query string) ([]byte, string, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, "", fmt.Errorf("music/download: query vazia")
	}

	// Se não for URL, busca no Deezer para obter a URL da faixa.
	input := query
	if !strings.HasPrefix(query, "http://") &&
		!strings.HasPrefix(query, "https://") {
		trackURL, err := searchDeezer(query)
		if err != nil {
			return nil, "", fmt.Errorf("music/download: %w", err)
		}
		input = trackURL
		fmt.Println("[music] faixa encontrada no Deezer:", input)
	}

	// Cria uma pasta temporária única para evitar conflito entre execuções.
	tmpDir, err := os.MkdirTemp("", "music-*")
	if err != nil {
		return nil, "", fmt.Errorf("music/download: erro ao criar pasta temporária: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	outTemplate := filepath.Join(tmpDir, "download.%(ext)s")

	args := []string{
		input,

		"--ignore-config", // Ignora configs globais do yt-dlp (evita conflitos no host)
		"--no-playlist",   // Evita baixar playlists inteiras sem querer

		"--socket-timeout", "15", // Timeout para evitar travamentos em rede lenta
		"--no-check-certificate", // Evita falhas TLS em ambientes restritos

		"--quiet",       // Reduz output desnecessário
		"--no-warnings", // Remove warnings que poluem logs

		"-x",                   // Extrai apenas áudio (sem vídeo)
		"-f", "bestaudio/best", // Melhor áudio disponível

		"--audio-format", "mp3", // Converte sempre para mp3
		"--audio-quality", "5", // Qualidade balanceada (rápido e leve)

		"--match-filter", "duration < 900", // Bloqueia faixas maiores que 15min

		"-o", outTemplate, // Template de saída temporária
	}

	fmt.Println("[music] executando yt-dlp com args:")
	fmt.Println(strings.Join(args, " "))
	fmt.Println("[music] query original:", query)
	fmt.Println("[music] input final:", input)
	fmt.Println("[music] temp dir:", tmpDir)
	fmt.Println("[music] output template:", outTemplate)

	var out bytes.Buffer
	cmd := ytdlpCmd(ctx, args...)
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		fmt.Println("[music] yt-dlp ERRO:")
		fmt.Println(out.String())
		return nil, "", fmt.Errorf("music/download: yt-dlp falhou: %w", err)
	}

	// Busca o arquivo gerado dinamicamente — não assume extensão fixa.
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return nil, "", fmt.Errorf("music/download: erro ao listar pasta temporária: %w", err)
	}

	var outFile, ext string
	for _, e := range entries {
		if !e.IsDir() {
			outFile = filepath.Join(tmpDir, e.Name())
			ext = strings.TrimPrefix(filepath.Ext(e.Name()), ".")
			break
		}
	}

	if outFile == "" {
		return nil, "", fmt.Errorf("music/download: nenhum arquivo gerado pelo yt-dlp em %s", tmpDir)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		return nil, "", fmt.Errorf("music/download: erro ao ler arquivo gerado: %w", err)
	}

	fmt.Println("[music] yt-dlp executou com sucesso")
	fmt.Println("[music] arquivo gerado:", outFile)
	fmt.Println("[music] output:")
	fmt.Println(out.String())

	return data, ext, nil
}
