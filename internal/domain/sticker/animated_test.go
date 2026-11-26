package sticker

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/ffmpeg"
)

// binAvailable checa se os binários do projeto estão presentes na raiz.
func binAvailable(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Clean(filepath.Join(wd, "../../.."))
	for _, bin := range []string{"bin/webpmux", "bin/ffmpeg"} {
		if _, err := os.Stat(filepath.Join(root, bin)); err != nil {
			t.Skipf("%s não disponível", bin)
		}
	}
	return root
}

// chdirRoot aponta o cwd para a raiz do repositório — os binários são
// chamados como ./bin/ffmpeg e ./bin/webpmux. Restaura ao final.
func chdirRoot(t *testing.T, root string) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
}

// geraWebPAnimado cria uma amostra de webp animado com o ffmpeg do projeto.
func geraWebPAnimado(t *testing.T) []byte {
	t.Helper()

	outFile, err := os.CreateTemp("", "sample-*.webp")
	if err != nil {
		t.Fatal(err)
	}
	outPath := outFile.Name()
	outFile.Close()
	os.Remove(outPath)
	t.Cleanup(func() { os.Remove(outPath) })

	cmd := ffmpeg.FfmpegCmd(context.Background(),
		"-hide_banner", "-y",
		"-f", "lavfi", "-i", "testsrc=s=512x512:d=2:rate=10",
		"-vf", "fps=10",
		"-vcodec", "libwebp", "-loop", "0",
		"-compression_level", "6", "-q:v", "35", "-threads", "2",
		outPath,
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("gerar webp animado: %v\n%s", err, output)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// O ffmpeg empacotado não decodifica webp animado, então a conversão passa
// por webpmux (frames) + composição em Go. Este teste valida o pipeline
// inteiro de ponta a ponta.
func TestConvertAnimatedWebPToMP4(t *testing.T) {
	root := binAvailable(t)
	chdirRoot(t, root)

	data := geraWebPAnimado(t)
	if !bytes.HasPrefix(data, []byte("RIFF")) {
		t.Fatalf("amostra não é webp: %q", data[:12])
	}

	out, err := ConvertAnimatedWebPToMP4(context.Background(), data)
	if err != nil {
		t.Fatalf("ConvertAnimatedWebPToMP4 falhou: %v", err)
	}

	// MP4 começa com a box "ftyp" na posição 4.
	if len(out) < 12 || string(out[4:8]) != "ftyp" {
		t.Fatalf("saída não é mp4 (primeiros bytes: %q)", out[:min(len(out), 16)])
	}
	if len(out) < 1024 {
		t.Errorf("mp4 pequeno demais: %d bytes", len(out))
	}
}

// TestConvertWebPStaticIgnoredAsAnimated garante que um webp estático não é
// tratado como animação (quem decide é o caller via dl.Animated).
func TestConvertAnimatedWebPRejectsGarbage(t *testing.T) {
	root := binAvailable(t)
	chdirRoot(t, root)

	if _, err := ConvertAnimatedWebPToMP4(context.Background(), []byte("não é webp")); err == nil {
		t.Error("esperava erro para conteúdo inválido")
	}
}

// TestConvertAnimatedWebPComposicao valida a composição de frames com offset
// e blend — o caso que o ffmpeg sozinho não cobre (webp animado não decodifica).
//
// Animação montada: canvas 300x300 azul; frame 1 = vermelho 200x200 em
// (50,60) com blend; frame 2 = amarelo 300x300 substituindo tudo.
func TestConvertAnimatedWebPComposicao(t *testing.T) {
	root := binAvailable(t)
	chdirRoot(t, root)

	dir := t.TempDir()
	base := geraCor(t, dir, "blue", "300x300", 80)
	f1 := geraCor(t, dir, "red", "200x200", 64)
	f2 := geraCor(t, dir, "yellow", "300x300", 64)

	sample := filepath.Join(dir, "offset.webp")
	cmd := webpmuxCmd(t,
		"-frame", f1, "+100+50+60+0+b",
		"-frame", f2, "+100+0+0+0-b",
		"-loop", "0", "-bgcolor", "255,0,0,255",
		"-o", sample, base,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("montar amostra: %v\n%s", err, out)
	}

	data, err := os.ReadFile(sample)
	if err != nil {
		t.Fatal(err)
	}

	mp4, err := ConvertAnimatedWebPToMP4(context.Background(), data)
	if err != nil {
		t.Fatalf("conversão falhou: %v", err)
	}
	mp4Path := filepath.Join(dir, "out.mp4")
	if err := os.WriteFile(mp4Path, mp4, 0644); err != nil {
		t.Fatal(err)
	}

	// Volta o primeiro frame para PNG e conferir a composição.
	frame0 := filepath.Join(dir, "frame0.png")
	cmd = ffmpeg.FfmpegCmd(context.Background(),
		"-hide_banner", "-y", "-i", mp4Path, "-frames:v", "1", "-update", "1", frame0)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("extrair frame: %v\n%s", err, out)
	}

	img := lePNG(t, frame0)
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	// Fator de escala do encode (300x300 → 512x512) para mapear os pixels.
	sx := w / 300
	sy := h / 300
	if sx == 0 {
		sx = 1
	}
	if sy == 0 {
		sy = 1
	}

	// Canto (10,10): fundo azul, fora do retângulo vermelho.
	verificaCor(t, img, 10*sx, 10*sy, "fundo", 0, 0, 255)
	// Centro (150,160): dentro do frame 1 em (50,60)-(250,260) → vermelho.
	verificaCor(t, img, 150*sx, 160*sy, "frame com offset", 255, 0, 0)
}

// geraCor cria um webp de cor sólida no tamanho pedido.
func geraCor(t *testing.T, dir, name, size string, quality int) string {
	t.Helper()
	out := filepath.Join(dir, name+".webp")
	cmd := ffmpeg.FfmpegCmd(context.Background(),
		"-hide_banner", "-loglevel", "error", "-y",
		"-f", "lavfi", "-i", "color=c="+name+":s="+size,
		"-frames:v", "1", "-vcodec", "libwebp", "-q:v", strconv.Itoa(quality), out)
	if o, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("gerar %s: %v\n%s", name, err, o)
	}
	return out
}

// webpmuxCmd monta a chamada ao binário (cwd na raiz).
func webpmuxCmd(t *testing.T, args ...string) *exec.Cmd {
	t.Helper()
	return exec.Command("./bin/webpmux", args...)
}

func lePNG(t *testing.T, path string) image.Image {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	return img
}

// verificaCor confere o pixel com tolerância (o encode yuv420p desloca um pouco).
func verificaCor(t *testing.T, img image.Image, x, y int, label string, r, g, b uint8) {
	t.Helper()
	cr, cg, cb, _ := img.At(x, y).RGBA()
	got := [3]uint8{uint8(cr >> 8), uint8(cg >> 8), uint8(cb >> 8)}
	// Tolerância de 60 por canal: suficiente para distinguir as cores e
	// absorver a conversão RGB→YUV.
	if dist(got[0], r) > 60 || dist(got[1], g) > 60 || dist(got[2], b) > 60 {
		t.Errorf("%s em (%d,%d): esperado (%d,%d,%d), veio (%d,%d,%d)",
			label, x, y, r, g, b, got[0], got[1], got[2])
	}
}

func dist(a, b uint8) int {
	if a > b {
		return int(a - b)
	}
	return int(b - a)
}
