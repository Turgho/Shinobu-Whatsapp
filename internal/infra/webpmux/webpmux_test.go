package webpmux

import (
	"testing"
)

// Saída real de `webpmux -info` em um webp animado (300x225) com frames de
// tamanho parcial — o caso que obriga a composição com offset.
const sampleInfo = `Canvas size: 300 x 225
Features present: animation transparency
Background color : 0xFF000000  Loop Count : 0
Number of frames: 4
No.: width height alpha x_offset y_offset duration   dispose blend image_size  compression
  1:   300   225    no        0        0      100       none    no       4242       lossy
  2:   204   205   yes       56       12      100       none   yes       4454       lossy
  3:   206   202   yes       54       14      100    background yes      4384       lossy
  4:   300   225    no        0        0      200       none   yes       3882       lossy
`

func TestParseInfo(t *testing.T) {
	info, err := parseInfo(sampleInfo)
	if err != nil {
		t.Fatalf("parseInfo falhou: %v", err)
	}

	if info.CanvasWidth != 300 || info.CanvasHeight != 225 {
		t.Errorf("canvas errado: %dx%d", info.CanvasWidth, info.CanvasHeight)
	}
	if len(info.Frames) != 4 {
		t.Fatalf("esperava 4 frames, veio %d", len(info.Frames))
	}

	f2 := info.Frames[1]
	if f2.Width != 204 || f2.Height != 205 {
		t.Errorf("frame 2 dimensões erradas: %dx%d", f2.Width, f2.Height)
	}
	if f2.X != 56 || f2.Y != 12 {
		t.Errorf("frame 2 offsets errados: (%d,%d)", f2.X, f2.Y)
	}
	if f2.DurationMS != 100 {
		t.Errorf("frame 2 duração errada: %d", f2.DurationMS)
	}
	if f2.Blend != "yes" {
		t.Errorf("frame 2 blend errado: %q", f2.Blend)
	}

	f3 := info.Frames[2]
	if f3.Dispose != "background" {
		t.Errorf("frame 3 dispose errado: %q", f3.Dispose)
	}
}

func TestParseInfoFullSize(t *testing.T) {
	const full = `Canvas size: 512 x 512
Features present: animation transparency
Background color : 0xFFFFFFFF  Loop Count : 0
Number of frames: 2
No.: width height alpha x_offset y_offset duration   dispose blend image_size  compression
  1:   512   512    no        0        0      100       none   yes       4396       lossy
  2:   512   512    no        0        0      100       none   yes       4388       lossy
`
	info, err := parseInfo(full)
	if err != nil {
		t.Fatalf("parseInfo falhou: %v", err)
	}
	if len(info.Frames) != 2 {
		t.Errorf("esperava 2 frames, veio %d", len(info.Frames))
	}
	if bg := info.Background(); bg.R != 0xFF || bg.G != 0xFF || bg.B != 0xFF || bg.A != 0xFF {
		t.Errorf("background errado: %+v", bg)
	}
	// 2 frames de 100ms → 10fps
	if fps := info.AverageFPS(); fps < 9.9 || fps > 10.1 {
		t.Errorf("AverageFPS errado: %v", fps)
	}
}

func TestParseInfoInvalid(t *testing.T) {
	if _, err := parseInfo("garbage"); err == nil {
		t.Error("esperava erro para saída inválida")
	}
}
