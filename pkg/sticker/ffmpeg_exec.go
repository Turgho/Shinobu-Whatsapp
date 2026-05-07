package sticker

import (
	"context"
	"os/exec"
)

// ffmpegCmd cria o exec.Cmd para o ffmpeg.
var ffmpegCmd = func(ctx context.Context, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, "ffmpeg", args...)
}
