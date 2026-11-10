package sticker

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/ffmpeg"
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

	// Nome único por requisição: um nome derivado do PID colidia entre
	// "!sticker" simultâneos (mesmo arquivo sobrescrito pelo ffmpeg -y).
	tmpOutFile, err := os.CreateTemp(os.TempDir(), "sticker-out-*.webp")
	if err != nil {
		return nil, fmt.Errorf("sticker/convert: erro ao criar arquivo temporário de saída: %w", err)
	}
	tmpOut := tmpOutFile.Name()
	tmpOutFile.Close()
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
			"-vf", scaleFilter + ",fps=10",
			"-vcodec", "libwebp",
			"-loop", "0",
			"-compression_level", "6",
			"-q:v", "35",
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

	if err := ffmpeg.RunLowPriority(cmd); err != nil {
		return fmt.Errorf("sticker/convert: ffmpeg falhou: %w\n%s", err, stderr.String())
	}

	if err := injectStickerMeta(ctx, output, "𝕯𝖗𝖆𝖒𝖆𝖙𝖍𝖚𝖗𝖌𝖔 14991791379", "𝑺𝒉𝒊𝒏𝒐𝒃𝒖 🦇"); err != nil {
		return err
	}

	return nil
}

// injectStickerMeta insere metadados EXIF no webp via webpmux.
// O EXIF segue formato TIFF little-endian (0x49,0x49) com tag "Application Notes" (0x41,0x57 = "AW")
// contendo um JSON com pack-name, publisher e emoji. O tamanho do payload é colocado
// nos bytes 14-17 do cabeçalho EXIF (little-endian uint32).
func injectStickerMeta(ctx context.Context, webpPath, author, pack string) error {
	jsonBytes := []byte(fmt.Sprintf(
		`{"sticker-pack-id":"yuuko.shinobu","sticker-pack-name":"%s","sticker-pack-publisher":"%s","emojis":["🦇"]}`,
		pack, author,
	))

	exifAttr := []byte{
		0x49, 0x49, 0x2A, 0x00, // TIFF little-endian header + magic
		0x08, 0x00, 0x00, 0x00, // offset do IFD
		0x01, 0x00, // número de tags (1)
		0x41, 0x57, // tag ID — "AW" (Application Notes)
		0x07, 0x00, // tipo de dado (undefined)
		0x00, 0x00, 0x00, 0x00, // placeholder: tamanho do payload
		0x16, 0x00, 0x00, 0x00, // offset do payload
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
	// CommandContext: se o pedido for cancelado (timeout do handler), o
	// webpmux morre junto — antes ele ficava órfão rodando.
	cmd := exec.CommandContext(ctx, "./bin/webpmux",
		"-set", "exif", tmpExif.Name(),
		webpPath, "-o", tmpOut,
	)
	cmd.SysProcAttr = ffmpeg.LowPriorityProc() // webpmux também com prioridade baixa

	if out, err := cmd.CombinedOutput(); err != nil {
		os.Remove(tmpOut)
		return fmt.Errorf("webpmux: %w\n%s", err, string(out))
	}

	return os.Rename(tmpOut, webpPath)
}
