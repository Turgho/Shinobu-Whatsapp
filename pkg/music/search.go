package music

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DownloadAudio baixa áudio temporário a partir de nome ou URL.
// - Se query for URL, usa direto.
// - Se query for texto, pesquisa no YouTube com ytsearch1.
// - Retorna os bytes do arquivo final e a extensão gerada.
//
// Exemplo de retorno final: mp3
func DownloadAudio(ctx context.Context, query string) ([]byte, string, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, "", fmt.Errorf("music/download: query vazia")
	}

	// Se não for URL, transforma em busca no YouTube.
	input := query
	if !strings.HasPrefix(query, "http://") &&
		!strings.HasPrefix(query, "https://") {
		input = "ytsearch1:" + query
	}

	// Cria uma pasta temporária única para evitar conflito entre execuções.
	tmpDir, err := os.MkdirTemp("", "music-*")
	if err != nil {
		return nil, "", fmt.Errorf("music/download: erro ao criar pasta temporária: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// O yt-dlp vai gerar algo como /tmp/music-123456/download.mp3
	outTemplate := filepath.Join(tmpDir, "download.%(ext)s")
	outFile := filepath.Join(tmpDir, "download.mp3")

	args := []string{
		input,
		"--default-search", "ytsearch1",
		"--cookies", "cookies.txt", // Algumas música podem ser bloqueadas por restrição de idade ou por outros motivos
		"--ignore-config", // Evita configs externas influenciando o bot.
		"--no-playlist",   // Evita playlist inteira.
		"--quiet",         // Reduz ruído no terminal.
		"--no-warnings",
		"-x",                   // Extrai apenas o áudio.
		"-f", "bestaudio/best", // Melhor audio disponível
		"--audio-format", "mp3", // Converte para mp3.
		"--audio-quality", "5", // Qualidade razoável sem exagerar no tempo de processamento.
		"--match-filter", "duration < 1800", // Evita vídeos longos demais.
		"-o", outTemplate, // Nome de saída temporário.
	}

	var out bytes.Buffer
	cmd := ytdlpCmd(ctx, args...)
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return nil, "", fmt.Errorf("music/download: yt-dlp falhou: %w\n%s", err, out.String())
	}

	// Lê o arquivo final gerado.
	data, err := os.ReadFile(outFile)
	if err != nil {
		return nil, "", fmt.Errorf("music/download: erro ao ler arquivo gerado: %w", err)
	}

	return data, "mp3", nil
}
