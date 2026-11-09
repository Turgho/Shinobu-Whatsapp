// Configuração centralizada do subsistema de mídia (yt-dlp local).
//
// Não contextualizar aqui valores que dependem de .env/config.yaml — o
// `internal/app` monta esta struct a partir de configs.MusicConfig.
package music

import (
	"time"
)

// YtdlpExecConfig contém os parâmetros de execução do yt-dlp local.
// Cada campo tem default aplicado via defaults().
type YtdlpExecConfig struct {
	LocalEnabled bool   // processamento local ativado (default: true)
	YtdlpPath    string // caminho do binário yt-dlp (default: ./bin/yt-dlp)
	FfmpegPath   string // caminho do ffmpeg p/ yt-dlp (default: ./bin/ffmpeg)
	DenoPath     string // runtime JS (default: ./bin/deno; vazio = usar deno em PATH)
	TmpDir       string // diretório temporário das tarefas (default: os.TempDir())
	CacheDir     string // cache de áudio (default: storage/music-cache; vazio = desativado)
	MaxFileMB    int    // tamanho máximo do arquivo aceito (default: 24)
	MaxDownloads int    // downloads simultâneos (default: 1)
	TimeoutSec   int    // timeout por tarefa (default: 240)
}

// defaults aplica os valores padrão aos campos zerados.
func (c *YtdlpExecConfig) defaults() *YtdlpExecConfig {
	if c.YtdlpPath == "" {
		c.YtdlpPath = "./bin/yt-dlp"
	}
	if c.FfmpegPath == "" {
		c.FfmpegPath = "./bin/ffmpeg"
	}
	if c.MaxFileMB <= 0 {
		c.MaxFileMB = 24
	}
	if c.MaxDownloads <= 0 {
		c.MaxDownloads = 1
	}
	if c.TimeoutSec <= 0 {
		c.TimeoutSec = 240
	}
	return c
}

// LocalTimeout retorna o timeout por tarefa como time.Duration.
func (c *YtdlpExecConfig) LocalTimeout() time.Duration {
	return time.Duration(c.TimeoutSec) * time.Second
}
