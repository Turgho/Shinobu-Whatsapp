//go:build integration

// Teste de integração REAL do Executor: yt-dlp novo + Deno + YouTube.
// Conteúdo CC-BY (Blender Foundation) baixado com autorização explícita.
// Roda apenas com: INTEG=1 go test -tags integration ./internal/domain/music/
package music

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIntegrationYtdlpReal(t *testing.T) {
	if os.Getenv("INTEG") != "1" {
		t.Skip("só roda com INTEG=1")
	}
	// CWD do go test é o dir do pacote; resolve binários a partir da raiz do repo.
	repo, _ := filepath.Abs(filepath.Join("..", "..", ".."))
	cfg := &YtdlpExecConfig{
		YtdlpPath:  filepath.Join(repo, "bin", "yt-dlp"),
		FfmpegPath: filepath.Join(repo, "bin", "ffmpeg"),
		DenoPath:   filepath.Join(repo, "bin", "deno"),
		TmpDir:     t.TempDir(),
		MaxFileMB:  40,
		TimeoutSec: 120,
	}
	e := NewExecutor(cfg, nil)
	dest := filepath.Join(t.TempDir(), "integ.m4a")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	start := time.Now()
	if err := e.Download(ctx, "https://www.youtube.com/watch?v=YE7VzlLtp-4", dest); err != nil {
		t.Fatalf("download real falhou: %v", err)
	}
	info, err := os.Stat(dest)
	if err != nil || info.Size() == 0 {
		t.Fatalf("arquivo inválido: %v", err)
	}
	t.Logf("sucesso: %d bytes em %s", info.Size(), time.Since(start))
}

func TestIntegrationServiceWithCache(t *testing.T) {
	if os.Getenv("INTEG") != "1" {
		t.Skip("só roda com INTEG=1")
	}
	repo, _ := filepath.Abs(filepath.Join("..", "..", ".."))
	cfg := &YtdlpExecConfig{
		YtdlpPath:  filepath.Join(repo, "bin", "yt-dlp"),
		FfmpegPath: filepath.Join(repo, "bin", "ffmpeg"),
		DenoPath:   filepath.Join(repo, "bin", "deno"),
		TmpDir:     t.TempDir(),
		CacheDir:   filepath.Join(t.TempDir(), "cache"),
		MaxFileMB:  40,
		TimeoutSec: 120,
	}
	svc := NewService(NewExecutor(cfg, nil), NewCache(cfg.CacheDir, nil), "", "")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	start1 := time.Now()
	d1, ext, err := svc.DownloadAudio(ctx, "https://www.youtube.com/watch?v=YE7VzlLtp-4")
	if err != nil || len(d1) == 0 {
		t.Fatalf("1ª chamada falhou: %v", err)
	}
	n, mb := svc.cache.Stats()
	t.Logf("1ª: %d bytes ext=%s cache=%d/%.1fMB dur=%s", len(d1), ext, n, mb, time.Since(start1))

	start2 := time.Now()
	d2, _, err := svc.DownloadAudio(ctx, "https://www.youtube.com/watch?v=YE7VzlLtp-4")
	if err != nil || len(d2) == 0 {
		t.Fatalf("2ª chamada (cache) falhou: %v", err)
	}
	dur2 := time.Since(start2)
	t.Logf("2ª (cache hit): %d bytes dur=%s", len(d2), dur2)
	if dur2 > 2*time.Second {
		t.Fatalf("cache hit deveria ser quase instantâneo: %s", dur2)
	}
}
