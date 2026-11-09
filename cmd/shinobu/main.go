// Painel de controle do Shinobu no terminal.
//
// Uso:
//
//	go run ./cmd/shinobu              → menu interativo
//	go run ./cmd/shinobu status       → subcomando direto
//	shinobu release minor "feat: x"   → wizard de release
//
// Subcomandos: status, test, integ, build, release, run, clean-cache.
// Com `make cli` o binário fica em bin/shinobu (gitignored, uso local).
package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	// sem args → menu interativo; com args → execução direta
	if len(os.Args) <= 1 {
		menu()
		return
	}

	name := strings.ToLower(os.Args[1])
	rest := os.Args[2:]

	var c *command
	for i := range commands {
		if commands[i].name == name {
			c = &commands[i]
		}
	}
	if c == nil {
		die("subcomando inválido: %q — use status, test, integ, build, release, run, clean-cache", name)
	}

	if err := c.run(rest); err != nil {
		die("%v", err)
	}
}

// command é um subcomando da CLI.
type command struct {
	name string
	desc string
	run  func(args []string) error
}

// commands também alimenta o menu interativo (main_menu.go).
var commands = []command{
	{name: "status", desc: "Versão, branch, mudanças pendentes e binário", run: cmdStatus},
	{name: "test", desc: "Suite de testes unitários", run: func([]string) error { return doTest(false) }},
	{name: "integ", desc: "Testes de integração contra APIs reais (INTEG=1)", run: func([]string) error { return doTest(true) }},
	{name: "build", desc: "Build otimizado do binário Shinobu", run: func([]string) error { return doBuild() }},
	{name: "release", desc: "Wizard de release: bump + msg + confirmação + push", run: cmdRelease},
	{name: "run", desc: "Sobe o bot em primeiro plano (go run)", run: cmdRun},
	{name: "clean-cache", desc: "Limpa cache de música do yt-dlp", run: cmdCleanCache},
}

// cmdRun sobe o bot no primeiro plano (Ctrl+C para parar).
func cmdRun([]string) error {
	return runCmd("go", "run", "./cmd/bot/main.go")
}

func die(format string, args ...any) {
	fmt.Printf(colorRed+"\n✗ "+format+colorReset+"\n", args...)
	os.Exit(1)
}
