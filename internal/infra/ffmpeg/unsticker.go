package ffmpeg

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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

// ConvertPNGSequenceToMP4 monta um MP4 (H.264) a partir de uma sequência de
// PNGs (pattern no formato do ffmpeg, ex.: dir/frame-%03d.png).
//
// Não é usado para WebP animado direto: o ffmpeg empacotado (7.0.2) não
// decodifica webp animado. Quem separa os frames é o webpmux e quem compõe é
// domain/sticker/animated.go — aqui só entra a sequência já pronta.
//
// O caller é responsável por remover o arquivo retornado com os.Remove.
func ConvertPNGSequenceToMP4(ctx context.Context, pattern string, fps float64) (string, error) {
	outPath := filepath.Join(os.TempDir(), fmt.Sprintf("unsticker_%d.mp4", time.Now().UnixNano()))
	cmd := FfmpegCmd(ctx,
		"-framerate", strconv.FormatFloat(fps, 'f', 3, 64),
		"-i", pattern,
		"-vf", "scale=512:-2:flags=lanczos", // -2 mantém aspect ratio par, exigido pelo yuv420p
		"-c:v", "libx264",
		"-pix_fmt", "yuv420p", // obrigatório para compatibilidade com o player do WhatsApp
		"-movflags", "+faststart",
		outPath,
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("ffmpeg sequência->mp4: %w: %s", err, string(output))
	}
	return outPath, nil
}
