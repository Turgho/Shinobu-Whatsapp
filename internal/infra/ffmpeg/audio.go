package ffmpeg

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// ErrNoAudioTrack indica que a mídia de entrada não tem trilha de áudio.
var ErrNoAudioTrack = errors.New("ffmpeg: mídia sem trilha de áudio")

// maxAudioSeconds é o limite do WhatsApp para o áudio de um audio sticker
// (validado entre 1 e 10 segundos).
const maxAudioSeconds = 10

// reDuration captura "Duration: 00:00:03.20, ..." do stderr do ffmpeg.
var reDuration = regexp.MustCompile(`Duration:\s*(\d+):(\d+):(\d+(?:\.\d+)?)`)

// reProgress captura "time=00:00:03.10" da linha de progresso — informa a
// duração real do ARQUIVO GERADO (o Duration do input pode ser N/A).
var reProgress = regexp.MustCompile(`time=(\d+):(\d+):(\d+(?:\.\d+)?)`)

// ExtractOpus extrai a trilha de áudio de inPath para OGG Opus, cortada em
// maxAudioSeconds (limite do WhatsApp para figurinha com som). Devolve o
// caminho do arquivo e a duração real em segundos (mínimo 1).
//
// ErrNoAudioTrack é retornado quando a entrada não tem áudio — o caller
// decide o fallback.
// O caller é responsável por remover o arquivo retornado com os.Remove.
func ExtractOpus(ctx context.Context, inPath string) (string, int, error) {
	outFile, err := os.CreateTemp("", "soundsticker-*.ogg")
	if err != nil {
		return "", 0, fmt.Errorf("ffmpeg extrair áudio: temporário: %w", err)
	}
	outPath := outFile.Name()
	outFile.Close()
	os.Remove(outPath) // o ffmpeg cria o arquivo

	args := []string{
		"-t", strconv.Itoa(maxAudioSeconds), // limita na leitura do input
		"-i", inPath,
		"-vn",
		"-c:a", "libopus",
		"-b:a", "64k",
		"-ar", "48000",
		"-ac", "1",
		"-y", outPath,
	}

	var stderr bytes.Buffer
	cmd := FfmpegCmd(ctx, args...)
	cmd.Stderr = &stderr

	if err := RunLowPriority(cmd); err != nil {
		output := stderr.String()
		os.Remove(outPath)
		if isNoAudioOutput(output) {
			return "", 0, ErrNoAudioTrack
		}
		return "", 0, fmt.Errorf("ffmpeg extrair áudio: %w: %s", err, output)
	}

	return outPath, parseDurationSeconds(stderr.String()), nil
}

// isNoAudioOutput reconhece o erro do ffmpeg quando a entrada não tem trilha
// de áudio. Duas frases conforme a versão/etapa em que falha.
func isNoAudioOutput(output string) bool {
	return strings.Contains(output, "does not contain any stream") ||
		strings.Contains(output, "matches no streams")
}

// parseDurationSeconds devolve a duração real do áudio gerado, limitada ao
// intervalo [1, 10] do WhatsApp.
//
// Prioriza a última linha de progresso ("time=00:00:03.10"), que mede o
// arquivo de saída; o "Duration:" do input serve de fallback (em alguns
// formatos, como lavfi, vem N/A). Sem informação, assume o teto.
func parseDurationSeconds(stderr string) int {
	if m := lastMatch(reProgress, stderr); m != nil {
		if seconds, ok := durationToSeconds(m); ok {
			return clampAudioSeconds(seconds)
		}
	}
	if m := reDuration.FindStringSubmatch(stderr); m != nil {
		if seconds, ok := durationToSeconds(m); ok {
			return clampAudioSeconds(seconds)
		}
	}
	return maxAudioSeconds
}

// lastMatch devolve a ÚLTIMA ocorrência da regex (a linha final de progresso).
func lastMatch(re *regexp.Regexp, s string) []string {
	var last []string
	for _, m := range re.FindAllStringSubmatch(s, -1) {
		last = m
	}
	return last
}

// durationToSeconds converte o match (h, m, s) em segundos fracionários.
func durationToSeconds(m []string) (float64, bool) {
	if len(m) < 4 {
		return 0, false
	}
	h, _ := strconv.Atoi(m[1])
	min, _ := strconv.Atoi(m[2])
	sec, err := strconv.ParseFloat(m[3], 64)
	if err != nil {
		return 0, false
	}
	return float64(h*3600+min*60) + sec, true
}

// clampAudioSeconds aplica o intervalo [1, 10] com arredondamento para cima.
func clampAudioSeconds(total float64) int {
	seconds := int(total)
	if float64(seconds) < total {
		seconds++
	}
	if seconds < 1 {
		seconds = 1
	}
	if seconds > maxAudioSeconds {
		seconds = maxAudioSeconds
	}
	return seconds
}
