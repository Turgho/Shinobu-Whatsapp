package sticker

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// CheckFFmpeg verifica se o ffmpeg está instalado e acessível no PATH.
// Deve ser chamado durante a inicialização do bot para falhar cedo e com
// uma mensagem clara, em vez de só dar erro na hora que alguém usar !sticker.
func CheckFFmpeg() error {
	_, err := exec.LookPath("ffmpeg")
	if err != nil {
		return fmt.Errorf(
			"ffmpeg não encontrado no PATH\n" +
				"  → Linux/Ubuntu: sudo apt install ffmpeg\n" +
				"  → macOS:        brew install ffmpeg\n" +
				"  → Windows:      https://ffmpeg.org/download.html",
		)
	}
	return nil
}

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
	scaleFilter := "scale=512:512:force_original_aspect_ratio=decrease," +
		"pad=512:512:(ow-iw)/2:(oh-ih)/2:color=0x00000000"

	var args []string

	if animated {
		args = []string{
			"-i", input,
			"-vf", scaleFilter + ",fps=15",
			"-loop", "0",
			"-t", "6",
			"-preset", "default",
			"-compression_level", "6",
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
