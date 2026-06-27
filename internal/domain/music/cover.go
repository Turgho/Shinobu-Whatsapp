package music

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/ffmpeg"
)

// EmbedMetadata insere capa e título no arquivo de áudio via ffmpeg.
// Retorna o áudio modificado ou o original se falhar (metadata é opcional).
func EmbedMetadata(audioData []byte, ext, title string) ([]byte, error) {
	inputPath, err := writeTempAudio(audioData, ext)
	if err != nil {
		return audioData, nil
	}
	defer os.Remove(inputPath)

	coverArt := extractCoverArt(inputPath)
	outputPath := inputPath + ".out." + ext
	defer os.Remove(outputPath)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	args := []string{"-i", inputPath}

	if coverArt != nil {
		coverPath, err := writeCoverTemp(coverArt)
		if err == nil {
			defer os.Remove(coverPath)
			args = append(args, "-i", coverPath)
		}
	}

	args = append(args, "-map", "0:a")
	if coverArt != nil {
		args = append(args, "-map", "1")
	}
	args = append(args, "-c:a", "copy")

	// embed cover as attached picture if present
	if coverArt != nil {
		args = append(args, "-disposition:v", "attached_pic")
	}

	artist := extractArtist(title)
	cleanTitle := cleanTrackTitle(title)

	if artist != "" {
		args = append(args, "-metadata", fmt.Sprintf("artist=%s", artist))
	}
	if cleanTitle != "" {
		args = append(args, "-metadata", fmt.Sprintf("title=%s", cleanTitle))
	}

	args = append(args, "-y", outputPath)

	var stderr bytes.Buffer
	cmd := ffmpeg.FfmpegCmd(ctx, args...)
	cmd.Stderr = &stderr

	if err := ffmpeg.RunLowPriority(cmd); err != nil {
		return audioData, nil
	}

	out, err := os.ReadFile(outputPath)
	if err != nil {
		return audioData, nil
	}

	return out, nil
}

// extractCoverArt extrai thumbnail JPEG do áudio. Retorna nil se falhar.
func extractCoverArt(audioPath string) []byte {
	tmpFile, err := os.CreateTemp("", "cover-*.jpg")
	if err != nil {
		return nil
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := ffmpeg.FfmpegCmd(ctx,
		"-i", audioPath,
		"-an",
		"-vcodec", "mjpeg",
		"-vframes", "1",
		"-y",
		tmpPath,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := ffmpeg.RunLowPriority(cmd); err != nil {
		return nil
	}

	data, err := os.ReadFile(tmpPath)
	if err != nil {
		return nil
	}

	return data
}

// writeTempAudio salva dados em arquivo temporário.
func writeTempAudio(data []byte, ext string) (string, error) {
	f, err := os.CreateTemp("", "audio-*."+ext)
	if err != nil {
		return "", err
	}
	path := f.Name()
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(path)
		return "", err
	}
	f.Close()
	return path, nil
}

func writeCoverTemp(data []byte) (string, error) {
	f, err := os.CreateTemp("", "cover-*.jpg")
	if err != nil {
		return "", err
	}
	path := f.Name()
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(path)
		return "", err
	}
	f.Close()
	return path, nil
}

// extractArtist tenta separar "artista - música" de uma query.
func extractArtist(query string) string {
	parts := strings.SplitN(query, " - ", 2)
	if len(parts) == 2 {
		return strings.TrimSpace(parts[0])
	}
	parts = strings.SplitN(query, " – ", 2) // en dash
	if len(parts) == 2 {
		return strings.TrimSpace(parts[0])
	}
	return ""
}

// cleanTrackTitle retorna só o título da música, sem o artista.
func cleanTrackTitle(query string) string {
	for _, sep := range []string{" - ", " – "} {
		parts := strings.SplitN(query, sep, 2)
		if len(parts) == 2 {
			return strings.TrimSpace(parts[1])
		}
	}
	return strings.TrimSpace(query)
}
