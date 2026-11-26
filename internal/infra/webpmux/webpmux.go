// Package webpmux executa o binário webpmux (libwebp) para inspecionar WebP
// animado e extrair seus frames individualmente.
//
// Por quê: o ffmpeg estático empacotado (7.0.2, johnvansickle) não decodifica
// WebP animado — ele pula os chunks ANIM/ANMF e falha com "image data not
// found". O demuxer animado do FFmpeg (libavformat/webp_anim_dec.c) só entrou
// em 2026-05 e não existe nas branches de release 7.x/8.x. O caminho
// viável, portanto, é separar os frames com webpmux e compor/converter por fora.
package webpmux

import (
	"context"
	"fmt"
	"image/color"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"syscall"
)

// binPath aponta para o binário (mesmo diretório ./bin do ffmpeg).
const binPath = "./bin/webpmux"

// Frame descreve um frame de um WebP animado, conforme `webpmux -info`.
type Frame struct {
	Index      int    // posição no arquivo, base 1
	Width      int    // largura do fragmento
	Height     int    // altura do fragmento
	X          int    // offset horizontal no canvas
	Y          int    // offset vertical no canvas
	DurationMS int    // duração exibida, em milissegundos
	Dispose    string // "none" ou "background"
	Blend      string // "yes" (compor com alfa) ou "no" (substituir a área)
}

// Info é o resultado de `webpmux -info`.
type Info struct {
	CanvasWidth  int
	CanvasHeight int
	Frames       []Frame
	bgARGB       uint32 // cor de fundo no formato 0xAARRGGBB
}

// AverageFPS devolve a taxa de frames média da animação (fallback: 10fps).
func (i *Info) AverageFPS() float64 {
	var total int
	for _, f := range i.Frames {
		total += f.DurationMS
	}
	if len(i.Frames) == 0 || total <= 0 {
		return 10
	}
	fps := 1000.0 * float64(len(i.Frames)) / float64(total)
	// Faixa razoável: evita 0.1fps (frame de 10s) ou 1000fps (duração 1ms).
	if fps < 1 {
		return 1
	}
	if fps > 30 {
		return 30
	}
	return fps
}

var (
	reCanvas  = regexp.MustCompile(`Canvas size:\s*(\d+)\s*x\s*(\d+)`)
	reFrames  = regexp.MustCompile(`Number of frames:\s*(\d+)`)
	reBGColor = regexp.MustCompile(`Background color\s*:\s*0x([0-9A-Fa-f]+)`)
	reLine    = regexp.MustCompile(
		`^\s*(\d+):\s+(\d+)\s+(\d+)\s+(\w+)\s+(\d+)\s+(\d+)\s+(\d+)\s+(\w+)\s+(\w+)\s+`)
)

// ReadInfo lê os metadados do WebP animado (canvas, frames e durações).
func ReadInfo(ctx context.Context, path string) (*Info, error) {
	out, err := run(ctx, "-info", path)
	if err != nil {
		return nil, fmt.Errorf("webpmux -info: %w: %s", err, string(out))
	}
	return parseInfo(string(out))
}

// ExtractFrame extrai o frame de índice index (base 1) para outPath.
func ExtractFrame(ctx context.Context, inPath string, index int, outPath string) error {
	out, err := run(ctx, "-get", "frame", strconv.Itoa(index), inPath, "-o", outPath)
	if err != nil {
		return fmt.Errorf("webpmux frame %d: %w: %s", index, err, string(out))
	}
	return nil
}

// run executa o webpmux e devolve a saída combinada.
func run(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, binPath, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Pdeathsig: syscall.SIGTERM,
	}
	return cmd.CombinedOutput()
}

// parseInfo interpreta a saída de `webpmux -info`.
func parseInfo(out string) (*Info, error) {
	info := &Info{}

	if m := reCanvas.FindStringSubmatch(out); m != nil {
		info.CanvasWidth, _ = strconv.Atoi(m[1])
		info.CanvasHeight, _ = strconv.Atoi(m[2])
	}
	if m := reBGColor.FindStringSubmatch(out); m != nil {
		v, err := strconv.ParseUint(m[1], 16, 32)
		if err == nil {
			info.bgARGB = uint32(v)
		}
	}
	if m := reFrames.FindStringSubmatch(out); m != nil {
		count, _ := strconv.Atoi(m[1])
		// Limite de segurança: um sticker animado típico tem <100 frames.
		// Acima disso (webp corrompido ou não-sticker) paramos aqui em vez
		// de estourar disco/RAM com N processos.
		if count > 300 {
			count = 300
		}
		info.Frames = make([]Frame, 0, count)
	}

	for _, line := range strings.Split(out, "\n") {
		m := reLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		idx, _ := strconv.Atoi(m[1])
		if idx > 300 {
			break
		}
		width, _ := strconv.Atoi(m[2])
		height, _ := strconv.Atoi(m[3])
		x, _ := strconv.Atoi(m[5])
		y, _ := strconv.Atoi(m[6])
		dur, _ := strconv.Atoi(m[7])

		info.Frames = append(info.Frames, Frame{
			Index:      idx,
			Width:      width,
			Height:     height,
			X:          x,
			Y:          y,
			DurationMS: dur,
			Dispose:    strings.ToLower(m[8]),
			Blend:      strings.ToLower(m[9]),
		})
	}

	if info.CanvasWidth == 0 || len(info.Frames) == 0 {
		return nil, fmt.Errorf("webpmux: saída inesperada em -info")
	}
	return info, nil
}

// Background devolve a cor de fundo do WebP. Fundo com alfa baixo vira preto
// opaco: o MP4 de saída não tem canal alfa e pixels transparentes ficariam
// com lixo visual.
func (i *Info) Background() color.RGBA {
	if i.bgARGB == 0 {
		return color.RGBA{A: 0xFF}
	}
	a := uint8(i.bgARGB >> 24)
	if a < 0x80 {
		return color.RGBA{A: 0xFF}
	}
	return color.RGBA{
		R: uint8(i.bgARGB >> 16),
		G: uint8(i.bgARGB >> 8),
		B: uint8(i.bgARGB),
		A: a,
	}
}
