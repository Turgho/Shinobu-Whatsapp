// Cores ANSI, helpers de leitura e saída do painel — stdlib puro.
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const (
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
	colorReset  = "\033[0m"
)

// stdinReader compartilhado: um único bufio.Reader sobre stdin evita
// perder bytes já bufferizados ao encadear leituras (menu → wizard → confirm).
var stdinReader = bufio.NewReader(os.Stdin)

// readLine lê uma linha (com \n final).
func readLine() string {
	line, _ := stdinReader.ReadString('\n')
	return line
}

// readLineTrimmed lê uma linha sem espaços nas bordas.
func readLineTrimmed() string {
	return strings.TrimSpace(readLine())
}

// ok imprime um "✓ msg" verde.
func ok(format string, args ...any) {
	fmt.Print(colorGreen + "✓ " + fmt.Sprintf(format, args...) + colorReset + "\n")
}

// warn imprime um "! msg" amarelo.
func warn(format string, args ...any) {
	fmt.Print(colorYellow + "! " + fmt.Sprintf(format, args...) + colorReset + "\n")
}

// dim imprime texto atenuado (detalhe não crítico).
func dim(s string) { fmt.Print(colorDim + s + colorReset + "\n") }

// head imprime um título de seção em bold+cyan.
func head(format string, args ...any) {
	fmt.Print(colorBold + colorCyan + "\n== " + fmt.Sprintf(format, args...) + colorReset + "\n")
}

// runCmd executa um comando no primeiro plano, herdando stdin/stdout/stderr
// (saída em tempo real, sem captura — delegado pro Makefile).
func runCmd(name string, args ...string) error {
	c := execCmd(name, args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	return c.Run()
}

// outCmd executa e retorna stdout trimado; erro vira mensagem legível.
func outCmd(name string, args ...string) (string, error) {
	out, err := execCmdOutput(name, args...)
	if err != nil {
		return "", fmt.Errorf("%s falhou: %w", name, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// confirm pergunta s/N ao usuário.
func confirm(question string) bool {
	fmt.Print(colorYellow + "? " + question + " [s/N]: " + colorReset)
	s := strings.ToLower(readLineTrimmed())
	return s == "s" || s == "sim" || s == "y" || s == "yes"
}
