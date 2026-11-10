// Testes unitários do executor do yt-dlp (sem rede, sem serviço externo):
// usa um binário fake em shell no lugar do ./bin/yt-dlp.
package music

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// writeFakeYtdlp cria um script que imita o contrato esperado do yt-dlp:
// recebe args, grava o arquivo indicado após "-o" e loga invocação em logfile.
func writeFakeYtdlp(t testing.TB, body string) string {
	t.Helper()
	script := filepath.Join(t.TempDir(), "yt-dlp-fake")
	shell := "#!/bin/sh\n" + body
	if err := os.WriteFile(script, []byte(shell), 0o755); err != nil {
		t.Fatal(err)
	}
	return script
}

// fakeWriter: grava arquivo destino + registra args num arquivo.
const fakeWriter = `
logfile="$FAKE_LOG"
echo "$@" >> "$logfile"
target=""
prev=""
for a in "$@"; do
  if [ "$prev" = "-o" ]; then target="$a"; fi
  prev="$a"
done
head -c 64 /dev/urandom > "$target"
`

func newTestExecutor(t *testing.T, ytdlpPath string, cfg *YtdlpExecConfig) *Executor {
	t.Helper()
	cfg = cfg.defaults()
	cfg.YtdlpPath = ytdlpPath
	cfg.FfmpegPath = filepath.Join(t.TempDir(), "ffmpeg") // nao usado pelo fake
	e := NewExecutor(cfg, nil)
	e.log = nil
	return e
}

func TestDownloadSuccess(t *testing.T) {
	logfile := filepath.Join(t.TempDir(), "calls.log")
	os.WriteFile(logfile, nil, 0o644)
	os.Setenv("FAKE_LOG", logfile)
	defer os.Unsetenv("FAKE_LOG")

	execDir := t.TempDir()
	bin := writeFakeYtdlp(t, fakeWriter)
	cfg := &YtdlpExecConfig{TmpDir: execDir, CacheDir: "", MaxFileMB: 1}
	e := newTestExecutor(t, bin, cfg)

	dest := filepath.Join(t.TempDir(), "dest.m4a")
	if err := e.Download(context.Background(), "mc hariel parceiro", dest); err != nil {
		t.Fatalf("esperava sucesso, got: %v", err)
	}
	if fi, err := os.Stat(dest); err != nil || fi.Size() == 0 {
		t.Fatalf("destino inválido: %v", err)
	}

	// query não-URL recebe prefixo de busca; argv é seguro (query como 1 elemento)
	calls, _ := os.ReadFile(logfile)
	if !strings.Contains(strings.TrimSuffix(string(calls), "\n"), "\nytsearch1:mc hariel parceiro\n") &&
		!strings.Contains(strings.TrimSuffix(string(calls), "\n"), "ytsearch1:mc hariel parceiro") {
		t.Fatalf("argv inesperado: %s", calls)
	}

	// tmpdir da tarefa foi removido (só pode rester o dir base, vazio)
	entries, _ := os.ReadDir(execDir)
	for _, en := range entries {
		if inner, err := os.ReadDir(filepath.Join(execDir, en.Name())); err == nil && len(inner) > 0 {
			t.Fatalf("tmpdir %s não foi limpo (%d itens)", en.Name(), len(inner))
		}
	}
}

func TestDownloadURLNoSearchPrefix(t *testing.T) {
	logfile := filepath.Join(t.TempDir(), "calls.log")
	os.WriteFile(logfile, nil, 0o644)
	os.Setenv("FAKE_LOG", logfile)
	defer os.Unsetenv("FAKE_LOG")

	bin := writeFakeYtdlp(t, fakeWriter)
	e := newTestExecutor(t, bin, &YtdlpExecConfig{TmpDir: t.TempDir()})

	dest := filepath.Join(t.TempDir(), "dest.m4a")
	err := e.Download(context.Background(), "https://www.youtube.com/watch?v=abc", dest)
	if err != nil {
		t.Fatalf("fallo inesperado: %v", err)
	}
	if calls, _ := os.ReadFile(logfile); strings.Contains(string(calls), "ytsearch1:") {
		t.Fatalf("URL não deveria ganhar prefixo de busca: %s", calls)
	}
}

func TestDownloadTimeout(t *testing.T) {
	bin := writeFakeYtdlp(t, "sleep 10\n")
	e := newTestExecutor(t, bin, &YtdlpExecConfig{TmpDir: t.TempDir(), TimeoutSec: 1})
	dest := filepath.Join(t.TempDir(), "dest.m4a")
	start := time.Now()
	err := e.Download(context.Background(), "x", dest)
	if err == nil {
		t.Fatal("esperava timeout")
	}
	ed, ok := AsErrDownload(err)
	if !ok || ed.Category != ErrTimeout {
		t.Fatalf("categoria errada: %v", err)
	}
	if time.Since(start) > 3*time.Second {
		t.Fatalf("timeout não matou processo cedo: %v", time.Since(start))
	}
}

func TestDownloadCancelBeforeStart(t *testing.T) {
	bin := writeFakeYtdlp(t, fakeWriter)
	e := newTestExecutor(t, bin, &YtdlpExecConfig{TmpDir: t.TempDir()})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := e.Download(ctx, "q", filepath.Join(t.TempDir(), "d.m4a"))
	if ed, ok := AsErrDownload(err); !ok || ed.Category != ErrTimeout {
		t.Fatalf("esperava ErrTimeout, got %v", err)
	}
}

func TestDownloadConcurrencyLimit(t *testing.T) {
	// Script marca início/fim de seção crítica com lock em mkdir.
	// Teste lê os timestamps e verifica max concorrente == MaxDownloads.
	const maxOverlap = 2
	script := `
prev=""
for a in "$@"; do
  if [ "$prev" = "-o" ]; then target="$a"; fi
  prev="$a"
done
	dir=$(dirname "$target")
lock="$dir.lock"
while ! mkdir "$lock" 2>/dev/null; do sleep 0.02; done
date +%s%N > "$lock.start"
sleep 0.4
date +%s%N > "$lock.end"
rmdir "$lock"
head -c 8 /dev/urandom > "$target"
`
	bin := writeFakeYtdlp(t, script)
	tmp := t.TempDir()
	// dests ficam no mesmo diretório base, então os marks .lock.* também
	e := newTestExecutor(t, bin, &YtdlpExecConfig{
		TmpDir: tmp, CacheDir: "", MaxDownloads: maxOverlap, TimeoutSec: 30,
	})

	n := 4
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		go func(i int) {
			dest := filepath.Join(tmp, fmt.Sprintf("dest%d.m4a", i))
			errs <- e.Download(context.Background(), fmt.Sprintf("q%d", i), dest)
		}(i)
	}
	for i := 0; i < n; i++ {
		if err := <-errs; err != nil {
			t.Fatalf("download %d falhou: %v", i, err)
		}
	}

	// coleta janelas de sessão: os marks vivem como
	// "<tmpdir>.lock.start" / "<tmpdir>.lock.end" (paralelos aos tmpdirs)
	entries, _ := os.ReadDir(tmp)
	type win struct{ s, en time.Time }
	wins := []win{}
	for _, en := range entries {
		if !strings.HasSuffix(en.Name(), ".lock.start") {
			continue
		}
		basePath := filepath.Join(tmp, strings.TrimSuffix(en.Name(), ".start"))
		start, _ := nsFile(basePath + ".start")
		end, _ := nsFile(basePath + ".end")
		wins = append(wins, win{start, end})
	}
	if len(wins) != n {
		t.Fatalf("janelas coletadas = %d, esperado %d", len(wins), n)
	}
	max := 0
	for _, w := range wins {
		c := 0
		for _, v := range wins {
			if v.s.Before(w.en) && v.en.After(w.s) {
				c++
			}
		}
		if c > max {
			max = c
		}
	}
	if max > maxOverlap {
		t.Fatalf("max concorrente %d > limite %d", max, maxOverlap)
	}
}

func nsFile(path string) (time.Time, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return time.Time{}, err
	}
	var ns int64
	fmt.Sscanf(string(b), "%d", &ns)
	return time.Unix(0, ns), nil
}

func TestClassifyErr(t *testing.T) {
	cases := []struct {
		stderr string
		want   ErrCategory
	}{
		{"ERROR: ... HTTP Error 403: Forbidden", ErrForbidden},
		{"HTTP Error 429: Too Many Requests", ErrRateLimited},
		{"Sign in to confirm you're not a bot", ErrSignIn},
		{"unable to communicate with deno (js runtime)", ErrJsRuntime},
		{"File is larger than max-filesize", ErrTooBig},
		{"Unsupported URL: https://x", ErrExtraction},
		{"qualquer outro erro", ErrUnknown},
	}
	for _, c := range cases {
		if got := classifyErr(c.stderr, nil); got != c.want {
			t.Errorf("classify(%q) = %v, want %v", c.stderr, got, c.want)
		}
	}
	if got := classifyErr("", context.DeadlineExceeded); got != ErrTimeout {
		t.Errorf("classify(ctx deadline) = %v, want ErrTimeout", got)
	}
}

func TestCategoryText(t *testing.T) {
	if ErrForbidden.CategoryText() == ErrUnknown.CategoryText() {
		t.Fatal("categorias devem ter textos distintos")
	}
}

func TestSanitizeErr(t *testing.T) {
	long := strings.Repeat("a", 600)
	if len(sanitizeErr(long)) != 512 {
		t.Fatal("esperava truncamento p/ 512")
	}
	if got := sanitizeErr("/tmp/opencode/ytdlp-abc/audio.m4a: ok"); strings.Contains(got, "ytdlp-") {
		t.Fatalf("caminho de tmpdir deveria ser cortado: %q", got)
	}
}

func TestBuildArgsFlags(t *testing.T) {
	cfg := (&YtdlpExecConfig{DenoPath: "/bin/deno", MaxFileMB: 20}).defaults()
	args := buildArgs(cfg, "groove coverage", "/t/audio.m4a")
	text := strings.Join(args, " ")
	for _, want := range []string{"--no-playlist", "--max-filesize 20M", "-f bestaudio", "/bin/deno"} {
		if !strings.Contains(text, want) {
			t.Fatalf("args faltam %q em: %s", want, text)
		}
	}
	if args[len(args)-1] != "ytsearch1:groove coverage" {
		t.Fatalf("query deve ser o último argv element: %s", text)
	}
}
