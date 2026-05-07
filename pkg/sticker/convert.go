package sticker

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// ConvertToWebP converte os bytes de entrada (imagem ou vídeo) em WebP
// usando ffmpeg e retorna os bytes do arquivo resultante.
//
// Para imagens estáticas: redimensiona para 512×512 mantendo proporção,
// com padding transparente.
//
// Para vídeos/gifs animados: idem, limitado a 6 segundos e 15fps,
// que são os limites do WhatsApp para stickers animados.
func ConvertToWebP(ctx context.Context, data []byte, ext string, animated bool) ([]byte, error) {
	tmpIn, err := os.CreateTemp("", "sticker-in-*"+ext)
	if err != nil {
		return nil, fmt.Errorf("sticker/convert: erro ao criar arquivo temporário de entrada: %w", err)
	}
	defer os.Remove(tmpIn.Name())

	if _, err = tmpIn.Write(data); err != nil {
		tmpIn.Close()
		return nil, fmt.Errorf("sticker/convert: erro ao gravar mídia de entrada: %w", err)
	}
	tmpIn.Close()

	tmpOut := filepath.Join(os.TempDir(), fmt.Sprintf("sticker-out-%d.webp", os.Getpid()))
	defer os.Remove(tmpOut)

	if err := runFFmpeg(ctx, tmpIn.Name(), tmpOut, animated); err != nil {
		return nil, err
	}

	out, err := os.ReadFile(tmpOut)
	if err != nil {
		return nil, fmt.Errorf("sticker/convert: erro ao ler webp gerado: %w", err)
	}

	return out, nil
}

// runFFmpeg executa o ffmpeg com os argumentos corretos para cada tipo de mídia.
func runFFmpeg(ctx context.Context, input, output string, animated bool) error {
	scaleFilter := "scale=512:512:force_original_aspect_ratio=increase," +
		"crop=512:512"

	var args []string

	if animated {
		args = []string{
			"-ss", "0",
			"-t", "6",
			"-i", input,
			"-vf", scaleFilter + ",fps=15",
			"-vcodec", "libwebp",
			"-loop", "0",
			"-compression_level", "6",
			"-q:v", "60",
			"-threads", "2",
			"-an",
			"-y",
			output,
		}
	} else {
		args = []string{
			"-i", input,
			"-vf", scaleFilter,
			"-frames:v", "1",
			"-y",
			output,
		}
	}

	var stderr bytes.Buffer
	cmd := ffmpegCmd(ctx, args...)
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("sticker/convert: ffmpeg falhou: %w\n%s", err, stderr.String())
	}

	return nil
}
