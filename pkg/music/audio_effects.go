package music

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Turgho/YuukoWhatsapp/pkg/ffmpeg"
)

// Effect representa um efeito de áudio disponível.
type Effect struct {
	Name        string
	Description string
	Filter      string
}

// Effects contém todos os efeitos disponíveis pelo nome do arg.
var Effects = map[string]Effect{
	"reverb": {
		Name:        "reverb",
		Description: "reverb + slowed clássico",
		Filter:      "atempo=0.85,aecho=0.8:0.88:30|50|80:0.35|0.25|0.15",
	},
	"deep": {
		Name:        "deep",
		Description: "mais lento e grave",
		Filter:      "atempo=0.75,aecho=0.8:0.88:40|70|100:0.4|0.3|0.2",
	},
	"echo": {
		Name:        "echo",
		Description: "eco pronunciado",
		Filter:      "aecho=0.8:0.9:500|1000:0.5|0.3",
	},
	"nightcore": {
		Name:        "nightcore",
		Description: "mais rápido e agudo",
		Filter:      "atempo=1.25,asetrate=44100*1.25,aresample=44100",
	},
	"bass": {
		Name:        "bass",
		Description: "boost de grave",
		Filter:      "equalizer=f=60:width_type=o:width=2:g=8",
	},
	"lofi": {
		Name:        "lofi",
		Description: "lofi com ruído e reverb leve",
		Filter:      "atempo=0.9,aecho=0.6:0.7:20|40:0.2|0.1,highpass=f=200,lowpass=f=8000",
	},
}

// EffectList retorna uma string formatada com todos os efeitos disponíveis.
func EffectList() string {
	var b bytes.Buffer
	b.WriteString("🎛️ *Efeitos disponíveis:*\n\n")
	for key, e := range Effects {
		b.WriteString(fmt.Sprintf("▸ `%s` — %s\n", key, e.Description))
	}
	return b.String()
}

// Apply aplica o efeito pelo nome nos bytes de áudio fornecidos
// e retorna os bytes processados em mp3.
func Apply(ctx context.Context, data []byte, ext string, effectName string) ([]byte, error) {
	effect, ok := Effects[effectName]
	if !ok {
		return nil, fmt.Errorf("efeito %q não encontrado", effectName)
	}

	tmpIn, err := os.CreateTemp("", "audio-in-*"+ext)
	if err != nil {
		return nil, fmt.Errorf("audioeffects: erro ao criar arquivo de entrada: %w", err)
	}
	defer os.Remove(tmpIn.Name())

	if _, err = tmpIn.Write(data); err != nil {
		tmpIn.Close()
		return nil, fmt.Errorf("audioeffects: erro ao gravar áudio de entrada: %w", err)
	}
	tmpIn.Close()

	tmpOut := filepath.Join(os.TempDir(), fmt.Sprintf("audio-out-%d.mp3", os.Getpid()))
	defer os.Remove(tmpOut)

	if err := runFFmpeg(ctx, tmpIn.Name(), tmpOut, effect.Filter); err != nil {
		return nil, err
	}

	out, err := os.ReadFile(tmpOut)
	if err != nil {
		return nil, fmt.Errorf("audioeffects: erro ao ler mp3 gerado: %w", err)
	}
	return out, nil
}

// runFFmpeg executa o ffmpeg com o filtro especificado.
func runFFmpeg(ctx context.Context, input, output, filter string) error {
	args := []string{
		"-i", input,
		"-af", filter,
		"-c:a", "libmp3lame",
		"-q:a", "2",
		"-threads", "1",
		"-y",
		output,
	}

	var stderr bytes.Buffer
	cmd := ffmpeg.FfmpegCmd(ctx, args...)
	cmd.Stderr = &stderr
	cmd.SysProcAttr = ffmpeg.LowPriorityProc()

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("audioeffects: ffmpeg falhou: %w\n%s", err, stderr.String())
	}
	return nil
}
