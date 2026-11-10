// Implementação dos comandos do painel — tudo delega pro Makefile ou go test,
// sem duplicar lógica de fluxo.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// execCmd cria um os.Cmd (helper primário para ui.go).
func execCmd(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}

// execCmdOutput executa e captura apenas stdout.
func execCmdOutput(name string, args ...string) ([]byte, error) {
	return execCmd(name, args...).Output()
}

// gitInfo carrega o estado atual do repositório.
type gitInfo struct {
	branch  string
	lastTag string
	dirtyN  int
	lastMsg string
}

func loadGitInfo() gitInfo {
	g := gitInfo{"?", "—", 0, ""}

	if b, err := outCmd("git", "rev-parse", "--abbrev-ref", "HEAD"); err == nil {
		g.branch = b
	}
	if t, err := outCmd("git", "describe", "--tags", "--abbrev=0"); err == nil {
		g.lastTag = t
	}
	if m, err := outCmd("git", "log", "-1", "--pretty=%s"); err == nil {
		g.lastMsg = m
	}
	if s, err := outCmd("git", "status", "--porcelain"); err == nil && s != "" {
		g.dirtyN = len(strings.Split(s, "\n"))
	}
	return g
}

// binarySize tamanho legível do binário Shinobu ("" se não existe).
func binarySize() string {
	st, err := os.Stat("Shinobu")
	if err != nil || st.IsDir() {
		return ""
	}
	return fmt.Sprintf("%.1f MB", float64(st.Size())/1024/1024)
}

// cmdStatus painel de status: git + binário.
func cmdStatus([]string) error {
	g := loadGitInfo()

	head("Status do repo")
	fmt.Printf("  branch:      %s\n", colorBold+g.branch+colorReset)
	fmt.Printf("  última tag:  %s\n", colorBold+g.lastTag+colorReset)
	dim("  último commit: " + g.lastMsg)
	if g.dirtyN > 0 {
		fmt.Printf("  pendentes:   %s%d arquivo(s)%s\n", colorYellow, g.dirtyN, colorReset)
	} else {
		fmt.Printf("  pendentes:   limpo\n")
	}

	if size := binarySize(); size != "" {
		fmt.Printf("  binário:     Shinobu (%s)\n", size)
	} else {
		dim("  binário:     não compilado — rode o alvo build")
	}
	return nil
}

// doTest roda os testes; quando integ=true, ativa a tag integration + INTEG=1.
func doTest(integ bool) error {
	if integ {
		warn("integração usa APIs reais — pode demorar (yt-dlp baixa vídeo)")
		c := execCmd("go", "test", "-tags", "integration", "-count=1", "./...")
		c.Env = append(os.Environ(), "INTEG=1")
		c.Stdout, c.Stderr, c.Stdin = os.Stdout, os.Stderr, os.Stdin
		return c.Run()
	}
	return runCmd("go", "test", "./...", "-count=1")
}

// doBuild delega pro Makefile (mesmos ldflags e bump de versão).
func doBuild() error {
	return runCmd("make", "build")
}

// makeRelease chama o alvo release do Makefile (fonte única do fluxo).
func makeRelease(bump, msg string, dry bool) error {
	argv := []string{"release"}
	if dry {
		argv = append(argv, "DRY=1")
	}
	argv = append(argv, "BUMP="+bump)
	if msg != "" {
		argv = append(argv, "MSG="+msg)
	}
	return runCmd("make", argv...)
}

// cmdRelease wizard: bump → msg → confirmação → release real via Makefile.
//
// Sem args os passos v interativos; com args (ex: release minor "feat: x")
// só pede confirmação.
func cmdRelease(args []string) error {
	head("Release wizard")

	bump := ""
	if len(args) > 0 {
		bump = strings.ToLower(args[0])
	}
	switch bump {
	case "patch", "minor", "major":
		// ok, veio por argumento
	case "":
		fmt.Println("Tipo de bump:")
		fmt.Printf("  %s1%s) patch — fix pequeno    (+0.0.1)\n", colorCyan, colorReset)
		fmt.Printf("  %s2%s) minor — feature nova   (+0.1.0)\n", colorCyan, colorReset)
		fmt.Printf("  %s3%s) major — breaking change(+1.0.0)\n", colorCyan, colorReset)
		fmt.Printf("%s? escolha [default 1]: %s", colorYellow, colorReset)
		switch readLineTrimmed() {
		case "", "1":
			bump = "patch"
		case "2":
			bump = "minor"
		case "3":
			bump = "major"
		default:
			return fmt.Errorf("escolha inválida — use 1, 2 ou 3")
		}
	default:
		return fmt.Errorf("bump inválido %q — use patch, minor ou major", bump)
	}

	msg := ""
	if len(args) > 1 {
		msg = strings.Join(args[1:], " ")
	} else {
		fmt.Printf("%s? mensagem do release (vazio usa a tag): %s", colorYellow, colorReset)
		msg = readLineTrimmed()
	}
	dim(fmt.Sprintf("msg: %q", msg))

	fmt.Printf("%sVai rodar: testes → build → commit (incl. binário) → bump %s → push%s\n",
		colorBold, bump, colorReset)
	if !confirm("Confirmar release?") {
		dim("cancelado — nada foi commitado, taggeado ou pushado")
		return nil
	}
	return makeRelease(bump, msg, false)
}

// cmdCleanCache apaga storage/music-cache (bot recria com MkdirAll quando precisar).
func cmdCleanCache([]string) error {
	const dir = "storage/music-cache"
	if _, err := os.Stat(dir); err != nil {
		warn("cache %s não existe — nada a limpar", dir)
		return nil
	}
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("limpando cache: %w", err)
	}
	ok("cache de música limpo (%s removido)", dir)
	return nil
}
