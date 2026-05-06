package sticker

import (
	"context"
	"os/exec"
)

// ffmpegCmd cria o exec.Cmd para o ffmpeg.
// Separado para facilitar substituição em testes.
var ffmpegCmd = func(ctx context.Context, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, "ffmpeg", args...)
}
