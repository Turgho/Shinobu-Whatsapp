package commands

import (
	"errors"
	"fmt"
	"testing"
)

// O sentinel ErrNotACommand precisa ser reconhecido no caminho com prefixo
// (antes disso, o usuário ficava sem nenhuma resposta).
func TestNotACommandErrorIs(t *testing.T) {
	err := NotACommand("Não entendi a moeda. Usa assim: !cotacao usd")
	if !errors.Is(err, ErrNotACommand) {
		t.Error("errors.Is(err, ErrNotACommand) deveria ser true")
	}
	if errors.Is(err, errors.New("outro")) {
		t.Error("não deve casar com erros aleatórios")
	}
}

func TestNotACommandMessage(t *testing.T) {
	// Mensagem específica do handler
	if got := notACommandMessage(NotACommand("Não entendi a moeda.")); got != "Não entendi a moeda." {
		t.Errorf("mensagem específica ignorada: %q", got)
	}

	// Erro genérico → fallback
	if got := notACommandMessage(errors.New("qualquer")); got == "" {
		t.Error("fallback não deveria ser vazio")
	}

	// Erro embrulhado com %w → a mensagem específica continua valendo
	wrapped := fmt.Errorf("cotacao: %w", NotACommand("moeda inválida"))
	if got := notACommandMessage(wrapped); got != "moeda inválida" {
		t.Errorf("mensagem embrulhada perdida: %q", got)
	}
}
