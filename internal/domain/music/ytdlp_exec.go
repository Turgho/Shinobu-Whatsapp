// Executor do yt-dlp como processo filho do Go.
//
// Ciclo de vida de cada tarefa:
//  1. semáforo de concorrência (MaxDownloads simultâneos)
//  2. diretório temporário próprio
//  3. yt-dlp via exec.CommandContext (sem shell, argv seguro)
//  4. timeout por tarefa (config.TimeoutSec) + cancelamento pelo chamador
//  5. classificação do erro (stderr → ErrCategory)
//  6. limpeza do tmpdir mesmo em erro/timeout/panico do chamador
//
// O executor grava o resultado no destPath do chamador (cache ou temp)
// sem carregar o arquivo na memória; o fluxo do !play só lê os bytes
// uma vez, para o upload do WhatsApp.
package music

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/failure"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"go.uber.org/zap"
)

// FailureCategory conecta a taxonomia da música à taxonomia compartilhada
// (internal/domain/failure) — o router usa isso para responder ao usuário.
func (e *ErrDownload) FailureCategory() failure.Category {
	switch e.Category {
	case ErrTimeout:
		return failure.Unavailable
	case ErrForbidden, ErrRateLimited, ErrSignIn:
		return failure.Forbidden
	case ErrTooBig, ErrExtraction:
		return failure.InvalidInput
	default:
		return failure.Unknown
	}
}

// ErrCategory classifica a falha de um download para log e mensagem ao usuário.
type ErrCategory int

const (
	ErrUnknown     ErrCategory = iota
	ErrTimeout                 // timeout da tarefa ou cancelamento
	ErrForbidden               // HTTP 403 — origem recusou o acesso
	ErrRateLimited             // HTTP 429 — origem pediu desaceleração
	ErrSignIn                  // origem exige autenticação (idade, região, PO token)
	ErrJsRuntime               // runtime JS (Deno) ausente ou incompatível
	ErrTooBig                  // arquivo maior que o limite configurado
	ErrExtraction              // falha de extração/formato do yt-dlp
)

// ErrDownload carrega a categoria do erro para o handler transformar em mensagem.
type ErrDownload struct {
	Category ErrCategory
	Detail   string // stderr truncado e saneado; nunca contém a query
}

func (e *ErrDownload) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("music/exec: categoria %d: %s", e.Category, e.Detail)
	}
	return fmt.Sprintf("music/exec: categoria %d", e.Category)
}

// CategoryText traduz a categoria para mensagem amigável ao usuário.
func (c ErrCategory) CategoryText() string {
	switch c {
	case ErrTimeout:
		return "a música demorou demais pra processar e foi cancelada."
	case ErrForbidden:
		return "o YouTube recusou esse download agora. Tenta com outro link."
	case ErrRateLimited:
		return "o YouTube pediu pra desacelerar. Tenta de novo em instantes."
	case ErrSignIn:
		return "esse conteúdo pede autenticação que o bot não tem (restrito ou por idade)."
	case ErrJsRuntime:
		return "componente do processador de mídia ausente no servidor. Avisa o dono."
	case ErrTooBig:
		return "essa música ficou grande demais até pra enviar. Tenta outra."
	default:
		return "Não achei essa música. Tenta com outro nome."
	}
}

// Executor gerencia execuções do yt-dlp com concorrência limitada.
type Executor struct {
	cfg *YtdlpExecConfig
	log *zap.Logger

	sem  chan struct{}
	busy atomic.Int64
}

// NewExecutor cria o executor e aplica defaults na config.
// Não valida existência de binários — use Probe() no startup.
func NewExecutor(cfg *YtdlpExecConfig, logger *zap.Logger) *Executor {
	c := cfg.defaults()
	return &Executor{
		cfg: c,
		log: logger,
		sem: make(chan struct{}, c.MaxDownloads),
	}
}

// Download baixa a query e grava o resultado final em destPath.
// destPath deve ficar FORA do tmpdir da tarefa (cache ou temp do chamador):
// se um download concorrente de mesma query cair no mesmo destino, o rename
// atômico (moveFile) garante um único vencedor sem arquivo corrompido.
func (e *Executor) Download(ctx context.Context, query, destPath string) error {
	// 1. Slot de concorrência.
	select {
	case e.sem <- struct{}{}:
		e.busy.Add(1)
		defer func() { <-e.sem; e.busy.Add(-1) }()
	case <-ctx.Done():
		return &ErrDownload{Category: ErrTimeout, Detail: "cancelado antes de iniciar"}
	}

	// 2. Diretório temporário da tarefa.
	dir := filepath.Join(e.cfg.TmpDir, "ytdlp-"+taskID())
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("music/exec: tmpdir: %w", err)
	}
	defer os.RemoveAll(dir) // 6. limpeza garantida

	// 3-4. execução com timeout.
	target := filepath.Join(dir, "audio.m4a")
	runCtx, cancel := context.WithTimeout(ctx, e.cfg.LocalTimeout())
	defer cancel()

	start := time.Now()
	args := buildArgs(e.cfg, query, target)
	// Kill de grupo de processos: o yt-dlp pode abrir filhos
	// (ffmpeg/deno) que não podem sobreviver ao cancelamento.
	cmd := exec.CommandContext(runCtx, e.cfg.YtdlpPath, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid:   true,            // grupo de processos próprio
		Pdeathsig: syscall.SIGTERM, // morre com o bot (evita órfão no container)
	}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) // mata o grupo
	}
	cmd.WaitDelay = 5 * time.Second
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if e.log != nil {
		e.log.Debug("music/exec: iniciando download",
			zap.String("task", filepath.Base(dir)), zap.Int("max_mb", e.cfg.MaxFileMB))
	}

	err := cmd.Run()
	dur := time.Since(start)

	if err != nil {
		detail := sanitizeErr(stderr.String())
		cat := classifyErr(detail, runCtx.Err())
		if e.log != nil {
			e.log.Warn("music/exec: falhou",
				zap.String("task", filepath.Base(dir)),
				zap.Duration("dur", dur),
				zap.String("category", cat.CategoryText()),
				zap.String("detail", detail),
				zap.String("ctx_err", fmt.Sprint(runCtx.Err())))
		}
		return &ErrDownload{Category: cat, Detail: detail}
	}

	if e.log != nil {
		e.log.Debug("music/exec: sucesso",
			zap.String("task", filepath.Base(dir)), zap.Duration("dur", dur))
	}

	// 5. validação do arquivo.
	info, err := os.Stat(target)
	if err != nil || info.Size() == 0 {
		detail := sanitizeErr(stderr.String())
		if e.log != nil {
			e.log.Warn("music/exec: arquivo final ausente",
				zap.String("task", filepath.Base(dir)), zap.String("detail", detail))
		}
		return &ErrDownload{Category: ErrExtraction, Detail: "arquivo final ausente"}
	}
	if info.Size() > int64(e.cfg.MaxFileMB)<<20 {
		return &ErrDownload{Category: ErrTooBig}
	}

	if err := moveFile(target, destPath); err != nil {
		return &ErrDownload{Category: ErrUnknown,
			Detail: fmt.Sprintf("mover resultado: %v", err)}
	}
	return nil
}

// BusySlots retorna o nº de downloads em execução (observabilidade admin).
func (e *Executor) BusySlots() int {
	return int(e.busy.Load())
}

// buildArgs monta os argumentos do yt-dlp. Query sempre como último
// elemento único do argv — sem shell, sem interpolação.
func buildArgs(cfg *YtdlpExecConfig, query, target string) []string {
	if !isHTTP(query) {
		query = "ytsearch1:" + query
	}
	args := []string{
		"--no-warnings",
		"--no-playlist",
		"-f", "bestaudio[ext=m4a]/bestaudio[ext=opus]/bestaudio/best",
		"--ffmpeg-location", cfg.FfmpegPath,
		"--concurrent-fragments", "2",
		"--buffer-size", "16K",
		"--retries", "3",
		"--fragment-retries", "3",
		"--retry-sleep", "3",
		"--max-filesize", fmt.Sprintf("%dM", cfg.MaxFileMB),
		"--no-write-thumbnail",
		"--no-embed-metadata", // capa/título são embutidos depois (cover.go)
		"--no-mtime",
		"-o", target,
	}
	if cfg.DenoPath != "" {
		args = append(args, "--js-runtimes", "deno:"+cfg.DenoPath)
	}
	return append(args, query)
}

// isHTTP checagem barata de URL antes do prefixo de busca.
func isHTTP(query string) bool {
	return strings.HasPrefix(query, "http://") || strings.HasPrefix(query, "https://")
}

// classifyErr mapeia padrões de stderr do yt-dlp para categorias.
func classifyErr(stderr string, ctxErr error) ErrCategory {
	switch {
	case ctxErr != nil && errors.Is(ctxErr, context.DeadlineExceeded):
		return ErrTimeout
	case ctxErr != nil && errors.Is(ctxErr, context.Canceled):
		return ErrTimeout
	case strings.Contains(stderr, "HTTP Error 403"):
		return ErrForbidden
	case strings.Contains(stderr, "HTTP Error 429") ||
		strings.Contains(stderr, "too many requests"):
		return ErrRateLimited
	case strings.Contains(stderr, "Sign in to confirm") ||
		strings.Contains(stderr, "Sign in to verify") ||
		strings.Contains(stderr, "age-restricted") ||
		strings.Contains(stderr, "bot protection"):
		return ErrSignIn
	case strings.Contains(stderr, "deno") ||
		strings.Contains(stderr, "js runtime") ||
		strings.Contains(stderr, "JavaScript runtime"):
		return ErrJsRuntime
	case strings.Contains(stderr, "max-filesize") ||
		strings.Contains(stderr, "File is larger"):
		return ErrTooBig
	case strings.Contains(stderr, "is not a valid URL") ||
		strings.Contains(stderr, "Unsupported URL") ||
		strings.Contains(stderr, "No suitable format"):
		return ErrExtraction
	default:
		return ErrUnknown
	}
}

// sanitizeErr limpa e limita o stderr p/ log: 512 bytes, sem caminho de tmpdir.
func sanitizeErr(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 512 {
		s = s[:512]
	}
	// Remove caminhos da tarefa (não expõe layout de disco; e é a única
	// informação pseudo-identificadora no stderr do yt-dlp).
	if i := strings.Index(s, "ytdlp-"); i >= 0 && i > 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}

// taskID gera id único de tarefa p/ logs e tmpdir (ms + contador atômico
// p/ evitar colisão de duas tarefas criadas no mesmo milissegundo).
var taskSeq atomic.Uint64

func taskID() string {
	return fmt.Sprintf("%s.%d", time.Now().Format("150405.000"), taskSeq.Add(1))
}

// moveFile move com rename e fallback de cópia (tmpdir e cache podem
// estar em filesystems distintos). Não sobrescreve destino já pronto de
// outra concorrência: se o destino já existe, o download que chegou
// primeiro é o vencedor e o arquivo existente é servido.
func moveFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o644)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			// Destino já pronto (outro download ganhou): serve o existente.
			return nil
		}
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
