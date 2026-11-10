// Testes do parser de tempo do !agenda — lógica pura, sem rede.
// Regras relativas a "agora" são computadas no teste com a mesma regra
// do parser (tempo é monótono o suficiente para comparações de data).
package public

import (
	"testing"
	"time"
)

var testLoc = time.UTC

func TestNormalizePtMonths(t *testing.T) {
	if got := normalizePtMonths("5 de dezembro de 2026 09:00"); got != "5 de December de 2026 09:00" && got != "5 de december de 2026 09:00" {
		t.Errorf("normalizePtMonths = %q", got)
	}
	if got := normalizePtMonths("28 DE JUNHO"); got != "28 de june" {
		t.Errorf("uppercase: %q", got)
	}
}

func TestParseAgendaTimeAbsolute(t *testing.T) {
	// ISO com hora: parsa direto
	got, err := parseAgendaTime("2026-12-25T09:30", testLoc)
	if err != nil {
		t.Fatalf("ISO sem offset: %v", err)
	}
	if got.Year() != 2026 || got.Month() != time.December || got.Hour() != 9 || got.Minute() != 30 {
		t.Fatalf("ISO: %s", got)
	}
}

func TestParseAgendaTimeNoYearRollsForward(t *testing.T) {
	now := time.Now().In(testLoc)
	jan1 := time.Date(now.Year(), time.January, 1, 5, 0, 0, 0, testLoc)
	if jan1.Before(now) {
		jan1 = jan1.AddDate(1, 0, 0) // mesma regra do parser: data passada → próximo ano
	}

	got, err := parseAgendaTime("01/01 05:00", testLoc)
	if err != nil {
		t.Fatalf("01/01 05:00: %v", err)
	}
	if !got.Equal(jan1) {
		t.Fatalf("expected %s, got %s", jan1, got)
	}
}

func TestParseAgendaTimeNoTimeUses8h(t *testing.T) {
	now := time.Now().In(testLoc)
	target := time.Date(now.Year(), time.December, 25, 8, 0, 0, 0, testLoc)
	if target.Before(now) {
		target = target.AddDate(1, 0, 0)
	}

	got, err := parseAgendaTime("25/12", testLoc)
	if err != nil {
		t.Fatalf("25/12: %v", err)
	}
	if !got.Equal(target) {
		t.Fatalf("expected %s, got %s — sem hora deveria assumir 08:00", target, got)
	}
}

func TestParseAgendaTimePtMonths(t *testing.T) {
	now := time.Now().In(testLoc)
	target := time.Date(now.Year(), time.December, 25, 9, 0, 0, 0, testLoc)
	if target.Before(now) {
		target = target.AddDate(1, 0, 0)
	}

	got, err := parseAgendaTime("25 de dezembro 09:00", testLoc)
	if err != nil {
		t.Fatalf("25 de dezembro 09:00: %v", err)
	}
	if !got.Equal(target) {
		t.Fatalf("expected %s, got %s", target, got)
	}
}

func TestParseAgendaTimeRelativeMinutes(t *testing.T) {
	got, ok := parseRelativeDuration("daqui 5 minutos", testLoc)
	if !ok {
		t.Fatal("daqui 5 minutos deveria parar")
	}
	diff := time.Until(got)
	if diff < 4*time.Minute || diff > 6*time.Minute {
		t.Fatalf("delta esperado ~5min, got %s", diff)
	}
}

func TestParseAgendaTimeRelativeHoursAndDays(t *testing.T) {
	if _, ok := parseRelativeDuration("em 2 horas", testLoc); !ok {
		t.Fatal("em 2 horas deveria parsear")
	}
	if _, ok := parseRelativeDuration("3 dias", testLoc); !ok {
		t.Fatal("3 dias deveria parsear")
	}
	// unidade desconhecida não parseia
	if _, ok := parseRelativeDuration("7 semanas", testLoc); ok {
		t.Fatal("7 semanas deveria falhar (unidade não suportada)")
	}
}

func TestParseRelativeDateHojeAmanha(t *testing.T) {
	now := time.Now().In(testLoc)

	today, ok := parseRelativeDate("hoje 14:30", testLoc)
	if !ok {
		t.Fatal("hoje 14:30 deveria parsear")
	}
	if today.Hour() != 14 || today.Minute() != 30 || today.Day() != now.Day() {
		t.Fatalf("hoje: %s", today)
	}

	tomorrow, ok := parseRelativeDate("amanhã 08:00", testLoc)
	if !ok {
		t.Fatal("amanhã 08:00 deveria parsear")
	}
	if want := now.AddDate(0, 0, 1); tomorrow.Day() != want.Day() || tomorrow.Month() != want.Month() {
		t.Fatalf("amanhã: %s, want %s", tomorrow, want)
	}

	// "amanha" sem acento também aceita (digitado rápido no celular)
	if _, ok := parseRelativeDate("amanha 22:00", testLoc); !ok {
		t.Fatal("amanha sem acento deveria parsear")
	}
}

func TestParseAgendaTimeInvalid(t *testing.T) {
	if _, err := parseAgendaTime("não-é-data", testLoc); err == nil {
		t.Fatal("entrada inválida deveria retornar erro")
	}
}
