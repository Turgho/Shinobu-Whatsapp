package ffmpeg

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// ConvertWebPToPNG converte um arquivo WebP para PNG.
// Retorna o caminho do arquivo PNG gerado.
// O caller é responsável por remover o arquivo com os.Remove.
func ConvertWebPToPNG(ctx context.Context, inPath string) (string, error) {
	outPath := filepath.Join(os.TempDir(), fmt.Sprintf("unsticker_png_%d.png", os.Getpid()))

	cmd := FfmpegCmd(ctx, "-i", inPath, outPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("ffmpeg webp->png: %w: %s", err, string(output))
	}

	return outPath, nil
}

// ConvertWebPToGIF converte um WebP animado para GIF.
// Retorna o caminho do arquivo GIF gerado.
// O caller é responsável por remover o arquivo com os.Remove.
func ConvertWebPToGIF(ctx context.Context, inPath string) (string, error) {
	outPath := filepath.Join(os.TempDir(), fmt.Sprintf("unsticker_gif_%d.gif", os.Getpid()))

	cmd := FfmpegCmd(ctx, "-i", inPath, "-vf", "fps=15,scale=512:-1", outPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("ffmpeg webp->gif: %w: %s", err, string(output))
	}

	return outPath, nil
}
