package ffmpeg

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ConvertWebPToPNG converte um arquivo WebP estático para PNG.
// O caller é responsável por remover o arquivo com os.Remove.
func ConvertWebPToPNG(ctx context.Context, inPath string) (string, error) {
	outPath := filepath.Join(os.TempDir(), fmt.Sprintf("unsticker_%d.png", time.Now().UnixNano()))
	cmd := FfmpegCmd(ctx, "-i", inPath, outPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("ffmpeg webp->png: %w: %s", err, string(output))
	}
	return outPath, nil
}

// ConvertWebPToMP4 converte um WebP animado para MP4 (H.264), formato que
// o WhatsApp exige para mensagens de vídeo/GIF. WhatsApp não aceita GIF
// bruto como vídeo — internamente todo "GIF" do WhatsApp é um MP4 com a
// flag gifPlayback ativada no envio.
func ConvertWebPToMP4(ctx context.Context, inPath string) (string, error) {
	outPath := filepath.Join(os.TempDir(), fmt.Sprintf("unsticker_%d.mp4", time.Now().UnixNano()))
	cmd := FfmpegCmd(ctx, "-i", inPath,
		"-vf", "fps=15,scale=512:-2:flags=lanczos", // -2 mantém aspect ratio par, exigido pelo yuv420p
		"-c:v", "libx264",
		"-pix_fmt", "yuv420p", // obrigatório para compatibilidade com o player do WhatsApp
		"-movflags", "+faststart",
		outPath,
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("ffmpeg webp->mp4: %w: %s", err, string(output))
	}
	return outPath, nil
}
