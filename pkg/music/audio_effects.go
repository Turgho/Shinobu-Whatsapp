package music

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Turgho/YuukoWhatsapp/pkg/ffmpeg"
)

// Intensity representa a intensidade do efeito.
type Intensity int

const (
	IntensityLight  Intensity = iota // leve
	IntensityMedium                  // medio (padrão)
	IntensityHeavy                   // forte
)

// ParseIntensity converte uma string em Intensity.
// Retorna IntensityMedium se não reconhecer.
func ParseIntensity(s string) Intensity {
	switch s {
	case "leve", "light", "low":
		return IntensityLight
	case "forte", "heavy", "high":
		return IntensityHeavy
	default:
		return IntensityMedium
	}
}

// Effect define os parâmetros de um efeito em três intensidades.
type Effect struct {
	Name        string
	Description string
	Filters     [3]string // [leve, medio, forte]
}

// Effects contém todos os efeitos disponíveis.
var Effects = map[string]Effect{
	"reverb": {
		Name:        "reverb",
		Description: "reverb + slowed",
		Filters: [3]string{
			"atempo=0.90,aecho=0.7:0.75:20|40:0.2|0.1",           // leve
			"atempo=0.85,aecho=0.8:0.88:30|50|80:0.35|0.25|0.15", // medio
			"atempo=0.78,aecho=0.9:0.95:40|70|110:0.5|0.4|0.3",   // forte
		},
	},
	"deep": {
		Name:        "deep",
		Description: "mais lento e grave",
		Filters: [3]string{
			"atempo=0.88,aecho=0.7:0.80:30|60:0.25|0.15",         // leve
			"atempo=0.80,aecho=0.8:0.88:40|70:0.35|0.25",         // medio
			"atempo=0.70,aecho=0.85:0.92:50|90|130:0.45|0.3|0.2", // forte
		},
	},
	"echo": {
		Name:        "echo",
		Description: "eco",
		Filters: [3]string{
			"aecho=0.7:0.8:200|400:0.25|0.15",            // leve
			"aecho=0.8:0.9:400|700:0.4|0.25",             // medio
			"aecho=0.85:0.95:600|1000|1500:0.5|0.35|0.2", // forte
		},
	},
	"nightcore": {
		Name:        "nightcore",
		Description: "mais rápido e agudo",
		Filters: [3]string{
			"atempo=1.10,asetrate=44100*1.10,aresample=44100", // leve
			"atempo=1.25,asetrate=44100*1.25,aresample=44100", // medio
			"atempo=1.40,asetrate=44100*1.40,aresample=44100", // forte
		},
	},
	"bass": {
		Name:        "bass",
		Description: "boost de grave",
		Filters: [3]string{
			"equalizer=f=60:width_type=o:width=2:g=4",  // leve
			"equalizer=f=60:width_type=o:width=2:g=8",  // medio
			"equalizer=f=60:width_type=o:width=2:g=14", // forte
		},
	},
	"lofi": {
		Name:        "lofi",
		Description: "lofi com reverb leve",
		Filters: [3]string{
			"atempo=0.95,aecho=0.5:0.6:15|30:0.15|0.08,highpass=f=300,lowpass=f=10000", // leve
			"atempo=0.90,aecho=0.6:0.7:20|40:0.2|0.1,highpass=f=200,lowpass=f=8000",    // medio
			"atempo=0.82,aecho=0.7:0.8:30|60:0.3|0.2,highpass=f=150,lowpass=f=6000",    // forte
		},
	},
}

// EffectList retorna uma string formatada com todos os efeitos disponíveis.
func EffectList() string {
	var b bytes.Buffer
	b.WriteString("🎛️ *Efeitos disponíveis:*\n\n")
	for key, e := range Effects {
		b.WriteString(fmt.Sprintf("▸ `%s` — %s\n", key, e.Description))
	}
	b.WriteString("\n*Intensidades:* `leve` · `medio` · `forte`")
	b.WriteString("\n*Exemplo:* `!efeito reverb forte`")
	return b.String()
}

// Apply aplica o efeito nos bytes de áudio e retorna mp3 processado.
func Apply(ctx context.Context, data []byte, ext string, effectName string, intensity Intensity) ([]byte, error) {
	effect, ok := Effects[effectName]
	if !ok {
		return nil, fmt.Errorf("efeito %q não encontrado", effectName)
	}

	filter := effect.Filters[intensity]

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

	if err := runFFmpeg(ctx, tmpIn.Name(), tmpOut, filter); err != nil {
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
