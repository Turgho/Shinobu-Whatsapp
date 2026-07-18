package music

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

type Config struct {
	ServerURL string
	APIToken  string
}

// DownloadAudio baixa áudio de uma query (música/URL).
// Usa servidor remoto (MUSIC_SERVER_URL) se configurado, senão usa binário ytdlp local.
func DownloadAudio(ctx context.Context, cfg *Config, query string) ([]byte, string, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, "", fmt.Errorf("music/download: query vazia")
	}

	if cfg.ServerURL != "" && cfg.APIToken != "" {
		return downloadViaTunnel(ctx, cfg.ServerURL, cfg.APIToken, query)
	}

	if _, err := os.Stat("./bin/ytdlp"); err == nil {
		return downloadViaBinary(ctx, query)
	}

	return nil, "", fmt.Errorf("music/download: nenhum método disponível — defina MUSIC_SERVER_URL ou coloque o binário em ./bin/ytdlp")
}

func downloadViaTunnel(ctx context.Context, serverURL, apiToken, query string) ([]byte, string, error) {
	endpoint := strings.TrimRight(serverURL, "/") + "/play"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBufferString(query))
	if err != nil {
		return nil, "", fmt.Errorf("music/tunnel: erro ao criar requisição: %w", err)
	}
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Authorization", "Bearer "+apiToken)

	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("music/tunnel: erro ao chamar servidor: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("music/tunnel: servidor retornou %d: %s", resp.StatusCode, string(body))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("music/tunnel: erro ao ler resposta: %w", err)
	}

	return data, "mp3", nil
}

func downloadViaBinary(ctx context.Context, query string) ([]byte, string, error) {
	if !strings.HasPrefix(query, "http://") && !strings.HasPrefix(query, "https://") {
		query = "ytsearch1:" + query
	}

	tmpFile, err := os.CreateTemp("", "ytdlp-*.m4a")
	if err != nil {
		return nil, "", fmt.Errorf("music/binary: erro ao criar arquivo temporário: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	cmd := exec.CommandContext(ctx, "./bin/ytdlp",
		"-f", "bestaudio[ext=m4a]/bestaudio[ext=webm]/bestaudio",
		"--concurrent-fragments", "2",
		"--buffer-size", "16K",
		"--retries", "5",
		"--fragment-retries", "5",
		"--retry-sleep", "3",
		"--no-write-thumbnail",
		"--no-embed-metadata",
		"--no-mtime",
		"--no-post-overwrites",
		"--cookies-from-browser", "chrome",
		"-o", tmpPath,
		query,
	)
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, "", fmt.Errorf("music/binary: yt-dlp falhou: %w", err)
	}

	data, err := os.ReadFile(tmpPath)
	if err != nil {
		return nil, "", fmt.Errorf("music/binary: erro ao ler arquivo: %w", err)
	}

	return data, "m4a", nil
}
