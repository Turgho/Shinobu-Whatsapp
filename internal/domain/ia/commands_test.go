// Testes da whitelist de comandos do NLU (commands.go) — puros.
package ia

import (
	"strings"
	"testing"
)

func TestDispatchableCommand(t *testing.T) {
	valid := []string{"clima", "play", "sticker", "agenda", "piada", "aniversário"}
	for _, n := range valid {
		if !DispatchableCommand(n) {
			t.Errorf("DispatchableCommand(%q) deveria ser true", n)
		}
	}

	// variante sem acento aceita (o modelo às vezes produz sem acento)
	if !DispatchableCommand("aniversario") {
		t.Error("aniversario sem acento deveria ser aceito")
	}

	invalid := []string{"", " shutdown", "shutdown", "Sticker", "CLIMA", "reiniciar_api"}
	for _, n := range invalid {
		if DispatchableCommand(n) {
			t.Errorf("DispatchableCommand(%q) deveria ser false", n)
		}
	}
}

func TestBuildNLUPromptSection(t *testing.T) {
	section := buildNLUPromptSection()
	if !strings.HasPrefix(section, "COMANDOS:\n") {
		t.Fatalf("seção deveria começar com COMANDOS:, got %q", section[:20])
	}

	// toda a lista está presente, com descrição
	for _, name := range nluCommands {
		if !strings.Contains(section, "- "+name+":") {
			t.Errorf("comando %q ausente da seção", name)
		}
	}

	// constância de paridade: nluCommands × nluCommandDesc
	for _, name := range nluCommands {
		if desc, ok := nluCommandDesc[name]; !ok || desc == "" {
			t.Errorf("nluCommandDesc sem entrada para %q", name)
		}
	}
	for name := range nluCommandDesc {
		if !DispatchableCommand(name) {
			t.Errorf("nluCommandDesc tem %q fora de nluCommands", name)
		}
	}
}
