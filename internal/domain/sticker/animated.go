package sticker

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"

	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/ffmpeg"
	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/webpmux"
	"golang.org/x/image/webp"
)

// maxCanvasDim limita o canvas aceito: 1024 cobre com folga os stickers do
// WhatsApp (512×512) e evita estourar RAM com webp não-sticker.
const maxCanvasDim = 1024

// ConvertAnimatedWebPToMP4 converte bytes de WebP animado em MP4 (H.264) e
// devolve os bytes do vídeo pronto.
//
// Pipeline: o webpmux separa os frames (o ffmpeg do projeto não decodifica
// webp animado), a composição acontece em Go respeitando offset, blend e
// dispose de cada frame, e o ffmpeg monta o vídeo a partir da sequência.
// WhatsApp não aceita GIF bruto como vídeo — internamente todo "GIF" é um
// MP4 com a flag gifPlayback.
func ConvertAnimatedWebPToMP4(ctx context.Context, data []byte) ([]byte, error) {
	tmpIn, err := os.CreateTemp("", "unsticker-in-*.webp")
	if err != nil {
		return nil, fmt.Errorf("sticker/animated: temporário de entrada: %w", err)
	}
	defer os.Remove(tmpIn.Name())

	if _, err := tmpIn.Write(data); err != nil {
		tmpIn.Close()
		return nil, fmt.Errorf("sticker/animated: gravar entrada: %w", err)
	}
	tmpIn.Close()

	info, err := webpmux.ReadInfo(ctx, tmpIn.Name())
	if err != nil {
		return nil, err
	}
	if info.CanvasWidth > maxCanvasDim || info.CanvasHeight > maxCanvasDim {
		return nil, fmt.Errorf("sticker/animated: canvas grande demais (%dx%d)",
			info.CanvasWidth, info.CanvasHeight)
	}

	dir, err := os.MkdirTemp("", "unsticker-frames-*")
	if err != nil {
		return nil, fmt.Errorf("sticker/animated: diretório de frames: %w", err)
	}
	defer os.RemoveAll(dir)

	if err := composeFrames(ctx, info, tmpIn.Name(), dir); err != nil {
		return nil, err
	}

	mp4Path, err := ffmpeg.ConvertPNGSequenceToMP4(
		ctx, filepath.Join(dir, "frame-%03d.png"), info.AverageFPS())
	if err != nil {
		return nil, err
	}
	defer os.Remove(mp4Path)

	out, err := os.ReadFile(mp4Path)
	if err != nil {
		return nil, fmt.Errorf("sticker/animated: ler mp4: %w", err)
	}
	return out, nil
}

// composeFrames extrai frame a frame e pinta sobre um canvas comum, gravando
// o PNG de cada estado. Frame extraído é lido e descartado de cada vez — só
// um canvas + um frame em memória por vez (contêiner com 512MB de RAM).
func composeFrames(ctx context.Context, info *webpmux.Info, inPath, dir string) error {
	bg := info.Background()
	canvas := image.NewRGBA(image.Rect(0, 0, info.CanvasWidth, info.CanvasHeight))
	draw.Draw(canvas, canvas.Bounds(), image.NewUniform(bg), image.Point{}, draw.Src)

	srcPath := filepath.Join(dir, "src.webp")
	var prev webpmux.Frame

	for i, f := range info.Frames {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := webpmux.ExtractFrame(ctx, inPath, i+1, srcPath); err != nil {
			return err
		}

		img, err := decodeWebPFrame(srcPath)
		if err != nil {
			return fmt.Errorf("sticker/animated: decodificar frame %d: %w", i+1, err)
		}

		// dispose="background" no frame anterior limpa a área que ele
		// ocupou antes de pintar o próximo (regra do formato ANIM).
		if i > 0 && prev.Dispose == "background" {
			clearArea(canvas, prev, bg)
		}
		drawFrame(canvas, img, f)
		prev = f

		out := filepath.Join(dir, fmt.Sprintf("frame-%03d.png", i))
		if err := writePNG(out, canvas); err != nil {
			return err
		}
	}
	return nil
}

// decodeWebPFrame lê um webp extraído (fragmento, sempre estático).
func decodeWebPFrame(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return webp.Decode(f)
}

// drawFrame pinta um frame no canvas na posição declarada. blend="no" troca
// a área inteira (inclusive alfa); blend="yes" compõe por cima.
func drawFrame(dst *image.RGBA, src image.Image, f webpmux.Frame) {
	b := src.Bounds()
	rect := image.Rect(f.X, f.Y, f.X+b.Dx(), f.Y+b.Dy())
	op := draw.Over
	if f.Blend == "no" {
		op = draw.Src
	}
	draw.Draw(dst, rect, src, b.Min, op)
}

// clearArea apaga a área ocupada por um frame (dispose="background").
func clearArea(dst *image.RGBA, f webpmux.Frame, c color.Color) {
	draw.Draw(dst,
		image.Rect(f.X, f.Y, f.X+f.Width, f.Y+f.Height),
		image.NewUniform(c), image.Point{}, draw.Src)
}

func writePNG(path string, img image.Image) error {
	out, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("sticker/animated: criar png: %w", err)
	}
	if err := png.Encode(out, img); err != nil {
		out.Close()
		return fmt.Errorf("sticker/animated: gravar png: %w", err)
	}
	return out.Close()
}
