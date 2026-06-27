package music

import (
	"bytes"
	"context"
	"os"
	"time"

	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/ffmpeg"
)

// extractCoverArt extrai o thumbnail JPEG embutido no arquivo de áudio via ffmpeg.
// Retorna nil se não houver capa ou se o ffmpeg falhar — o caller trata nil como "sem capa".
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

// writeTempAudio escreve dados de áudio em um arquivo temporário e retorna o caminho.
// O caller deve remover o arquivo com defer os.Remove.
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

// ExtractCoverArtFromBytes extrai capa de áudio em memória.
// Conveniência que escreve o áudio em temp file, extrai a capa e limpa.
func ExtractCoverArtFromBytes(audioData []byte, ext string) []byte {
	path, err := writeTempAudio(audioData, ext)
	if err != nil {
		return nil
	}
	defer os.Remove(path)
	return extractCoverArt(path)
}
