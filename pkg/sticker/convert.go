package sticker

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Turgho/YuukoWhatsapp/pkg/ffmpeg"
)

// ConvertToWebP converte os bytes de entrada (imagem ou vídeo) em WebP
// usando ffmpeg e retorna os bytes do arquivo resultante.
//
// Para imagens estáticas: redimensiona e corta centralizado para 512×512.
// Para vídeos/gifs animados: idem, limitado a 6 segundos e 15fps
// (limites do WhatsApp para stickers animados).
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
	// Redimensiona cobrindo 512×512 inteiro e corta o excesso centralizado.
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
			"-vcodec", "libwebp",
			"-frames:v", "1",
			"-compression_level", "6",
			"-q:v", "80",
			"-threads", "1",
			"-y",
			output,
		}
	}

	var stderr bytes.Buffer
	cmd := ffmpeg.FfmpegCmd(ctx, args...)
	cmd.Stderr = &stderr
	cmd.SysProcAttr = ffmpeg.LowPriorityProc() // prioridade baixa no Linux

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("sticker/convert: ffmpeg falhou: %w\n%s", err, stderr.String())
	}

	injectStickerMeta(output, "𝑺𝒉𝒊𝒏𝒐𝒃𝒖", "𝕯𝖗𝖆𝖒𝖆𝖙𝖍𝖚𝖗𝖌𝖔")

	return nil
}

func injectStickerMeta(webpPath, author, pack string) error {
	jsonBytes := []byte(fmt.Sprintf(
		`{"sticker-pack-id":"yuuko.shinobu","sticker-pack-name":"%s","sticker-pack-publisher":"%s","emojis":["🦇"]}`,
		pack, author,
	))

	exifAttr := []byte{
		0x49, 0x49, 0x2A, 0x00,
		0x08, 0x00, 0x00, 0x00,
		0x01, 0x00,
		0x41, 0x57,
		0x07, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x16, 0x00, 0x00, 0x00,
	}

	l := len(jsonBytes)
	exifAttr[14] = byte(l)
	exifAttr[15] = byte(l >> 8)
	exifAttr[16] = byte(l >> 16)
	exifAttr[17] = byte(l >> 24)

	exif := append(exifAttr, jsonBytes...)

	tmpExif, err := os.CreateTemp("", "*.exif")
	if err != nil {
		return err
	}
	defer os.Remove(tmpExif.Name())
	tmpExif.Write(exif)
	tmpExif.Close()

	tmpOut := webpPath + ".tmp.webp"
	cmd := exec.Command("./bin/webpmux",
		"-set", "exif", tmpExif.Name(),
		webpPath, "-o", tmpOut,
	)
	cmd.SysProcAttr = ffmpeg.LowPriorityProc() // webpmux também com prioridade baixa

	if out, err := cmd.CombinedOutput(); err != nil {
		os.Remove(tmpOut)
		return fmt.Errorf("webpmux: %v\n%s", err, string(out))
	}

	return os.Rename(tmpOut, webpPath)
}
