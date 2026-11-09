// Menu interativo do painel — chamado quando a CLI roda sem subcomandos.
package main

import (
	"fmt"
	"strconv"
)

// clearScreen limpa o terminal via ANSI.
func clearScreen() { fmt.Print("\033[H\033[2J") }

// pauseWaiting aguarda enter para voltar ao menu.
func pauseWaiting() {
	fmt.Printf(colorDim + "\n[enter para voltar]" + colorReset)
	readLine()
}

// menu imprime o loop principal. Cada opção executa o subcomando e volta
// ao menu; 0/q saí.
func menu() {
	for {
		clearScreen()
		g := loadGitInfo()

		fmt.Print(colorBold + colorCyan)
		fmt.Println("\n╔══════════════════════════════════╗")
		fmt.Println("║  SHINOBU  ·  control panel       ║")
		fmt.Println("╚══════════════════════════════════╝")
		fmt.Print(colorReset)

		fmt.Printf("  %s%s%s · tag %s · ", colorBold, g.branch, colorReset, g.lastTag)
		if g.dirtyN > 0 {
			fmt.Printf("%s%d mudança(s) pendente(s)%s\n", colorYellow, g.dirtyN, colorReset)
		} else {
			fmt.Print(colorDim + "repo limpo" + colorReset + "\n")
		}

		fmt.Println()
		for i, c := range commands {
			fmt.Printf("  %s%2d%s) %-12s %s%s%s\n",
				colorCyan, i+1, colorReset, c.name, colorDim, c.desc, colorReset)
		}
		fmt.Printf("  %s 0%s) sair\n", colorCyan, colorReset)

		fmt.Printf("%s? opção: %s", colorYellow, colorReset)
		choice := readLineTrimmed()
		if choice == "0" || choice == "q" || choice == "sair" {
			return
		}

		n, err := strconv.Atoi(choice)
		if err != nil || n < 1 || n > len(commands) {
			warn("opção inválida — digite o número de 1 a %d", len(commands))
			pauseWaiting()
			continue
		}

		c := commands[n-1]
		fmt.Println()
		if err := c.run(nil); err != nil {
			fmt.Printf(colorRed+"✗ %v"+colorReset+"\n", err)
		}
		pauseWaiting()
	}
}
