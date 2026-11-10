package whatsapp

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// O WhatsApp recusa texto acima de ~4096 caracteres. Sem truncamento, o
// bot ficava sem resposta nenhuma (sem erro visível).
func TestTruncateTextDentroDoLimite(t *testing.T) {
	msg := strings.Repeat("a", 100)
	if got := truncateText(msg); got != msg {
		t.Errorf("texto curto não deveria ser alterado (%d runes)", utf8.RuneCountInString(got))
	}
}

func TestTruncateTextAcimaDoLimite(t *testing.T) {
	msg := strings.Repeat("a", maxTextLength+500)
	got := truncateText(msg)

	if utf8.RuneCountInString(got) > maxTextLength {
		t.Errorf("resultado passou do limite: %d runes", utf8.RuneCountInString(got))
	}
	if !strings.HasSuffix(got, "(continua)") {
		t.Errorf("faltou avisar o corte: %q", got[len(got)-40:])
	}
}

// Multibyte: o corte não pode partir um caractere UTF-8 no meio.
func TestTruncateTextPreservaRunes(t *testing.T) {
	msg := strings.Repeat("🌸", maxTextLength+100)
	got := truncateText(msg)

	if !utf8.ValidString(got) {
		t.Fatal("resultado não é UTF-8 válido")
	}
	if utf8.RuneCountInString(got) > maxTextLength {
		t.Errorf("resultado passou do limite: %d runes", utf8.RuneCountInString(got))
	}
}
