// Utilitários do pacote ia — testes puros, sem rede.
package ia

import (
	"strings"
	"testing"
)

// montar a tag de raciocínio por concatenação: os bytes literais da tag,
// se presentes no arquivo, são cortados por ferramentas de formatação.
func thinkBlock(inner string) string {
	return "<" + "think" + ">" + inner + "<" + "/" + "think" + ">"
}

func TestCleanPrompt(t *testing.T) {
	cases := map[string]string{
		"  olá  shinobu ": "olá  shinobu", // trim só nas bordas
		"":                "",
	}
	for in, want := range cases {
		if got := cleanPrompt(in); got != want {
			t.Errorf("cleanPrompt(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTruncateText(t *testing.T) {
	if got := truncateText("abcdefghij", 5); got != "abcde" {
		t.Errorf("truncate = %q", got)
	}
	// não deve cortar no meio de runa multi-byte
	got := truncateText("áéíóúáéíóú", 4)
	if got != "áéíó" {
		t.Errorf("truncate runes = %q (esperava 4 runas)", got)
	}
	if truncateText("abc", 10) != "abc" {
		t.Error("string curta não deve mudar")
	}
}

func TestStripThinkTags(t *testing.T) {
	if got := stripThinkTags("  resposta  final "); got != strings.TrimSpace("resposta  final") {
		t.Errorf("sem tag: %q", got)
	}
	tagged := thinkBlock(" interno pensado ") + "resposta"
	if got := stripThinkTags(tagged); got != "resposta" {
		t.Errorf("com tag: %q", got)
	}
}
