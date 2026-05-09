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

	args := []string{
		input,
		"--cookies", "cookies.txt",

		"--default-search", "ytsearch1", // Permite buscar músicas sem URL direta (YouTube search)

		"--ignore-config", // Ignora configs globais do yt-dlp (evita conflitos no host)
		"--no-playlist",   // Evita baixar playlists inteiras sem querer

		// web,android: o cliente android serve como fallback quando o web
		// retorna streams com restrição de formato (erro "format not available").
		"--extractor-args", "youtube:player_client=web,android",

		"--user-agent", "Mozilla/5.0", // Evita bloqueios simples de bot detection
		"--socket-timeout", "15", // Timeout para evitar travamentos em rede lenta

		"--no-check-certificate", // Evita falhas TLS em ambientes restritos

		"--quiet",       // Reduz output desnecessário
		"--no-warnings", // Remove warnings que poluem logs

		"-x",         // Extrai apenas áudio (sem vídeo)
		"-f", "ba/b", // ba = best audio; b = best geral como fallback universal

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

	// Após conversão o yt-dlp sempre gera download.mp3,
	// mas buscamos no diretório para não depender de extensão fixa.
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

	// Lê o arquivo final gerado.
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
