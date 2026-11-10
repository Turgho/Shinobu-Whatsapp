package commands

import "testing"

// As heurísticas do NLU decidem o que o bot faz com uma mensagem —
// deserving teste de tabela (o comportamento mudou várias vezes).
func TestIsBareMention(t *testing.T) {
	const botJID = "5511999999999@lid"

	casos := []struct {
		nome string
		msg  string
		want bool
	}{
		{"só o nome", "shinobu", true},
		{"só o nome maiúsculo", "Shinobu", true},
		{"nome com pontuação", "shinobu?", true},
		{"só o JID do bot", botJID, true},
		{"nome + espaço", "shinobu ", true},
		{"pergunta de verdade", "shinobu, que horas são?", false},
		{"outro nome", "kafka", false},
	}

	for _, c := range casos {
		if got := isBareMention(c.msg, botJID); got != c.want {
			t.Errorf("%s: isBareMention(%q) = %v, esperava %v", c.nome, c.msg, got, c.want)
		}
	}
}

// isMentioned normaliza a caixa: "Shinobu" e "shinobu" contam como
// menção, mas "shinobuzinho" não (palavra isolada).
func TestIsMentioned(t *testing.T) {
	const botJID = "5511999999999@lid"

	casos := []struct {
		nome string
		msg  string
		want bool
	}{
		{"palavra exata", "shinobu", true},
		{"maiúscula", "Shinobu", true},
		{"prefixo não casa", "shinobuzinho", false},
		{"sufixo não casa", "oshinobu", false},
		{"no meio da frase", "oi shinobu tudo bem", true},
		{"palavra diferente", "kafka", false},
		{"menção por JID", "falta com " + botJID, true},
	}

	for _, c := range casos {
		if got := isMentioned(c.msg, botJID); got != c.want {
			t.Errorf("%s: isMentioned(%q) = %v, esperava %v", c.nome, c.msg, got, c.want)
		}
	}
}

func TestTruncateRunes(t *testing.T) {
	// cortar por rune (nada de partir um caractere UTF-8)
	msg := "🌸🌸🌸"
	if got := truncateRunes(msg, 2); got != "🌸🌸…" {
		t.Errorf("truncateRunes = %q", got)
	}
	if got := truncateRunes("ab", 5); got != "ab" {
		t.Errorf("truncateRunes não deveria alterar texto curto: %q", got)
	}
}
