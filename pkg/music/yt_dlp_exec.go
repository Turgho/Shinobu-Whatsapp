package music

import (
	"context"
	"os/exec"
)

// ytdlpCmd cria o exec.Cmd para o yt-dlp
var ytdlpCmd = func(ctx context.Context, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, "./bin/yt-dlp", args...)
}
