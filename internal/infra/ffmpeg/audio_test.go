package ffmpeg

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestParseDurationSeconds(t *testing.T) {
	tests := []struct {
		name   string
		stderr string
		want   int
	}{
		{
			name:   "duração fracionada arredonda para cima",
			stderr: "[mov,mp4 @ 0x55] Duration: 00:00:03.20, start: 0.000000, bitrate: 128 kb/s",
			want:   4,
		},
		{
			name:   "exatamente o teto",
			stderr: "Duration: 00:00:07.00, start: 0.000000, bitrate: 99 kb/s",
			want:   7,
		},
		{
			name:   "duração maior que o limite é truncada",
			stderr: "Duration: 00:01:30.00, start: 0.000000, bitrate: 99 kb/s",
			want:   7,
		},
		{
			name:   "duração desconhecida assume o teto",
			stderr: "Duration: N/A, bitrate: N/A",
			want:   7,
		},
		{
			name:   "duração mínima nunca zera",
			stderr: "Duration: 00:00:00.10, start: 0.000000",
			want:   1,
		},
		{
			name:   "sem informação assume o teto",
			stderr: "sem nada aqui",
			want:   7,
		},
		{
			name: "progresso do output vence o Duration do input",
			stderr: "Duration: 00:01:30.00, start: 0.000000, bitrate: 99 kb/s\n" +
				"frame=  60 fps=0.0 q=-0.0 size=      36kB time=00:00:03.10 bitrate=  94.5kbits/s speed=10x\n",
			want: 4, // 3.10s arredondado para cima
		},
		{
			name: "última linha de progresso é a final",
			stderr: "time=00:00:01.00 bitrate= 94.5kbits/s\n" +
				"size=      72kB time=00:00:06.50 bitrate= 94.5kbits/s\n",
			want: 7, // 6.5s → 7
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseDurationSeconds(tt.stderr); got != tt.want {
				t.Errorf("parseDurationSeconds() = %d, esperado %d", got, tt.want)
			}
		})
	}
}

// binAvailable aponta para a raiz do repositório (os binários são chamados
// como ./bin/ffmpeg) e pula o teste quando não estão presentes.
func binAvailable(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Clean(filepath.Join(wd, "../../.."))
	for _, bin := range []string{"bin/ffmpeg"} {
		if _, err := os.Stat(filepath.Join(root, bin)); err != nil {
			t.Skipf("%s não disponível", bin)
		}
	}
	return root
}

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

func geraMidia(t *testing.T, args ...string) string {
	t.Helper()
	outFile, err := os.CreateTemp("", "ffmpeg-test-*.mp4")
	if err != nil {
		t.Fatal(err)
	}
	outPath := outFile.Name()
	outFile.Close()
	os.Remove(outPath)
	t.Cleanup(func() { os.Remove(outPath) })

	cmd := FfmpegCmd(context.Background(), append([]string{"-hide_banner", "-y"}, append(args, outPath)...)...)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("ffmpeg: %v\n%s", err, output)
	}
	return outPath
}

// Cobre o caminho feliz da extração: mp4 com trilha → OGG Opus válido.
func TestExtractOpus(t *testing.T) {
	root := binAvailable(t)
	chdirRoot(t, root)

	in := geraMidia(t,
		"-f", "lavfi", "-i", "sine=frequency=440:duration=3",
		"-f", "lavfi", "-i", "testsrc=s=320x240:duration=3",
		"-shortest", "-c:v", "libx264", "-pix_fmt", "yuv420p",
		"-c:a", "aac",
	)

	out, seconds, err := ExtractOpus(context.Background(), in)
	if err != nil {
		t.Fatalf("ExtractOpus falhou: %v", err)
	}
	defer os.Remove(out)

	if seconds < 1 || seconds > 7 {
		t.Errorf("seconds fora do intervalo [1,7]: %d", seconds)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(data, []byte("OggS")) {
		t.Error("saída não é OGG")
	}
	if !bytes.Contains(data[:min(len(data), 128)], []byte("OpusHead")) {
		t.Error("saída não tem cabeçalho Opus")
	}
}

// Mídia sem trilha de áudio é um fallback (figurinha comum), não um erro.
func TestExtractOpusSemAudio(t *testing.T) {
	root := binAvailable(t)
	chdirRoot(t, root)

	in := geraMidia(t,
		"-f", "lavfi", "-i", "testsrc=s=320x240:duration=1",
		"-c:v", "libx264", "-pix_fmt", "yuv420p",
	)

	if _, _, err := ExtractOpus(context.Background(), in); !errors.Is(err, ErrNoAudioTrack) {
		t.Errorf("esperava ErrNoAudioTrack, veio: %v", err)
	}
}
