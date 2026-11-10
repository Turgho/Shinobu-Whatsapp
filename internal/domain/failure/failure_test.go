package failure

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

type fakeStatus struct{ code int }

func (f fakeStatus) Error() string   { return "status" }
func (f fakeStatus) StatusCode() int { return f.code }

type fakeCategorizer struct{ c Category }

func (f fakeCategorizer) Error() string             { return "erro" }
func (f fakeCategorizer) FailureCategory() Category { return f.c }

func TestClassify(t *testing.T) {
	casos := []struct {
		nome string
		err  error
		want Category
	}{
		{"nil", nil, Unknown},
		{"timeout de contexto", context.DeadlineExceeded, Unavailable},
		{"cancelado", context.Canceled, Unavailable},
		{"403", fakeStatus{403}, Forbidden},
		{"429", fakeStatus{429}, Forbidden},
		{"404", fakeStatus{404}, NotFound},
		{"500", fakeStatus{500}, Unavailable},
		{"erro comum", errors.New("boom"), Unknown},
		{"com categoria própria", fakeCategorizer{InvalidInput}, InvalidInput},
		{"embrulhado", fmt.Errorf("camada: %w", fakeStatus{503}), Unavailable},
	}

	for _, c := range casos {
		if got := Classify(c.err); got != c.want {
			t.Errorf("%s: Classify = %v, esperava %v", c.nome, got, c.want)
		}
	}
}

func TestUserMessageNaoVazaDetalhe(t *testing.T) {
	for cat := Unknown; cat <= NotFound; cat++ {
		msg := cat.UserMessage()
		if msg == "" {
			t.Errorf("categoria %v sem mensagem", cat)
		}
		for _, prohibido := range []string{"error", "http", "status", "panic", "goroutine"} {
			if contains(msg, prohibido) {
				t.Errorf("mensagem da categoria %v vaza detalhe técnico: %q", cat, msg)
			}
		}
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
