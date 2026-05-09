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
// - Se query for texto, pesquisa no SoundCloud com scsearch1.
// - Retorna os bytes do arquivo final e a extensão gerada.
//
// Exemplo de retorno final: mp3
func DownloadAudio(ctx context.Context, query string) ([]byte, string, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, "", fmt.Errorf("music/download: query vazia")
	}

	// Se não for URL, transforma em busca no SoundCloud.
	input := query
	if !strings.HasPrefix(query, "http://") &&
		!strings.HasPrefix(query, "https://") {
		input = "scsearch1:" + query
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

		"--default-search", "scsearch1", // Busca no SoundCloud por padrão
		"--ignore-config", // Ignora configs globais do yt-dlp
		"--no-playlist",   // Evita baixar playlists inteiras

		"--socket-timeout", "15", // Timeout para evitar travamentos
		"--no-check-certificate", // Evita falhas TLS em ambientes restritos

		"--quiet",       // Reduz output desnecessário
		"--no-warnings", // Remove warnings que poluem logs

		"-x",                   // Extrai apenas áudio (sem vídeo)
		"-f", "bestaudio/best", // Melhor áudio disponível

		"--audio-format", "mp3", // Converte sempre para mp3
		"--audio-quality", "5", // Qualidade balanceada

		"--match-filter", "duration < 900", // Bloqueia faixas maiores que 15min

		"-o", outTemplate,
	}

	fmt.Println("[music] executando yt-dlp com args:")
	fmt.Println(strings.Join(args, " "))

	var out bytes.Buffer
	cmd := ytdlpCmd(ctx, args...)
	cmd.Stdout = &out
	cmd.Stderr = &out

	fmt.Println("[music] query original:", query)
	fmt.Println("[music] input final:", input)
	fmt.Println("[music] temp dir:", tmpDir)
	fmt.Println("[music] output template:", outTemplate)

	if err := cmd.Run(); err != nil {
		fmt.Println("[music] yt-dlp ERRO:")
		fmt.Println(out.String())
		return nil, "", fmt.Errorf("music/download: yt-dlp falhou: %w", err)
	}

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
