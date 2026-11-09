// Testes puros do pacote weather (helpers de parsing/horas — sem rede).
package weather

import "testing"

func TestParseFloat(t *testing.T) {
	if got := parseFloat("23.5"); got != 23.5 {
		t.Errorf("parseFloat(23.5) = %v", got)
	}
	if got := parseFloat("-7.25"); got != -7.25 {
		t.Errorf("parseFloat(-7.25) = %v", got)
	}
	// entrada inválida → 0 (contrato usado pelos handlers)
	if got := parseFloat("abc"); got != 0 {
		t.Errorf("parseFloat(abc) = %v", got)
	}
}

func TestFindHourIndexExact(t *testing.T) {
	times := []string{
		"2026-11-09T10:00", "2026-11-09T11:00", "2026-11-09T12:00",
	}
	if i := findHourIndex(times, "2026-11-09T11:00"); i != 1 {
		t.Errorf("índice exato = %d, want 1", i)
	}
}

func TestFindHourIndexFallback(t *testing.T) {
	times := []string{"2026-11-09T10:00", "2026-11-09T11:00", "2026-11-09T12:00"}
	// hora sem dado exato: último índice anterior a currentTime
	if i := findHourIndex(times, "2026-11-09T11:30"); i != 1 {
		t.Errorf("índice fallback = %d, want 1", i)
	}
	// currentTime posterior a todos os dados
	if i := findHourIndex(times, "2026-11-09T23:00"); i != len(times)-1 {
		t.Errorf("índie final = %d, want %d", i, len(times)-1)
	}
}

func TestFindHourIndexEdge(t *testing.T) {
	if i := findHourIndex(nil, "2026-11-09T11:00"); i != -1 {
		t.Errorf("times vazio = %d, want -1", i)
	}
	// currentTime curto demais (menos de 13 chars) → -1
	if i := findHourIndex([]string{"2026-11-09T10:00"}, "11:00"); i != -1 {
		t.Errorf("currentTime curto = %d, want -1", i)
	}
}
