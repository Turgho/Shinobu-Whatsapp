package ffmpeg

import (
	"context"
	"os/exec"
)

// ffmpegCmd cria o exec.Cmd para o ffmpeg.
var FfmpegCmd = func(ctx context.Context, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, "./bin/ffmpeg", args...)
}
