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
		"--cookies", "cookies.txt",

		"--default-search", "ytsearch1", // Permite buscar músicas sem URL direta (YouTube search)

		"--ignore-config", // Ignora configs globais do yt-dlp (evita conflitos no host)
		"--no-playlist",   // Evita baixar playlists inteiras sem querer

		// Usa client web (mais estável que android no Square Cloud)
		"--extractor-args", "youtube:player_client=tv_simply",

		"--user-agent", "Mozilla/5.0", // Evita bloqueios simples de bot detection
		"--socket-timeout", "15", // Timeout para evitar travamentos em rede lenta

		"--no-check-certificate", // Evita falhas TLS em ambientes restritos

		"--quiet",       // Reduz output desnecessário
		"--no-warnings", // Remove warnings que poluem logs

		"-x",              // Extrai apenas áudio (sem vídeo)
		"-f", "bestaudio", // Formato mais compatível (evita erro "format not available")

		"--audio-format", "mp3", // Converte sempre para mp3
		"--audio-quality", "5", // Qualidade balanceada (rápido e leve)

		"--match-filter", "duration < 900", // Bloqueia vídeos maiores que 15min

		"-o", outTemplate, // Template de saída temporária
	}

	fmt.Println("[music] executando yt-dlp com args:")
	fmt.Println(strings.Join(args, " "))

	var out bytes.Buffer
	cmd := ytdlpCmd(ctx, args...)
	cmd.Stdout = &out
	cmd.Stderr = &out

	// DEBUG LOGS
	fmt.Println("[music] query original:", query)
	fmt.Println("[music] input final:", input)
	fmt.Println("[music] temp dir:", tmpDir)
	fmt.Println("[music] output template:", outTemplate)

	if err := cmd.Run(); err != nil {
		fmt.Println("[music] yt-dlp ERRO:")
		fmt.Println(out.String())
		return nil, "", fmt.Errorf("music/download: yt-dlp falhou: %w", err)
	}

	// Lê o arquivo final gerado.
	data, err := os.ReadFile(outFile)
	if err != nil {
		return nil, "", fmt.Errorf("music/download: erro ao ler arquivo gerado: %w", err)
	}

	fmt.Println("[music] yt-dlp executou com sucesso")
	fmt.Println("[music] output:")
	fmt.Println(out.String())

	return data, "mp3", nil
}
