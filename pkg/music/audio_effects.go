package music

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Turgho/YuukoWhatsapp/pkg/ffmpeg"
)

// SlowedReverb aplica slowed (85% velocidade) + reverb nos bytes de áudio recebidos
// e retorna os bytes do arquivo resultante em mp3.
//
// ext deve incluir o ponto, ex: ".m4a", ".mp3", ".ogg".
func SlowedReverb(ctx context.Context, data []byte, ext string) ([]byte, error) {
	tmpIn, err := os.CreateTemp("", "audio-in-*"+ext)
	if err != nil {
		return nil, fmt.Errorf("audioeffects: erro ao criar arquivo de entrada: %w", err)
	}
	defer os.Remove(tmpIn.Name())

	if _, err = tmpIn.Write(data); err != nil {
		tmpIn.Close()
		return nil, fmt.Errorf("audioeffects: erro ao gravar áudio de entrada: %w", err)
	}
	tmpIn.Close()

	tmpOut := filepath.Join(os.TempDir(), fmt.Sprintf("audio-out-%d.mp3", os.Getpid()))
	defer os.Remove(tmpOut)

	if err := runFFmpeg(ctx, tmpIn.Name(), tmpOut); err != nil {
		return nil, err
	}

	out, err := os.ReadFile(tmpOut)
	if err != nil {
		return nil, fmt.Errorf("audioeffects: erro ao ler mp3 gerado: %w", err)
	}

	return out, nil
}

// runFFmpeg aplica os filtros de slowed + reverb via ffmpeg.
func runFFmpeg(ctx context.Context, input, output string) error {
	// atempo=0.85 → 85% da velocidade original
	// aecho      → simula reverb com 3 reflexões (delays em ms e seus decaimentos)
	audioFilter := "atempo=0.85,aecho=0.8:0.9:1000|1800|2500:0.4|0.3|0.2"

	args := []string{
		"-i", input,
		"-af", audioFilter,
		"-c:a", "libmp3lame", // codec mp3
		"-q:a", "2", // qualidade VBR (~190 kbps) — bom equilíbrio tamanho/qualidade
		"-threads", "2", // não sobrecarregar o servidor
		"-y",
		output,
	}

	var stderr bytes.Buffer
	cmd := ffmpeg.FfmpegCmd(ctx, args...)
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("audioeffects: ffmpeg falhou: %w\n%s", err, stderr.String())
	}

	return nil
}
