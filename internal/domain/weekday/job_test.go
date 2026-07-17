package weekday

import (
	"testing"
	"time"
)

func TestParseDay(t *testing.T) {
	tests := []struct {
		input string
		want  time.Weekday
		ok    bool
	}{
		{"friday", time.Friday, true},
		{"Friday", time.Friday, true},
		{"FRIDAY", time.Friday, true},
		{"monday", time.Monday, true},
		{"sunday", time.Sunday, true},
		{"saturday", time.Saturday, true},
		{"fri", 0, false},
		{"", 0, false},
		{"sexta", 0, false},
	}
	for _, tt := range tests {
		got, ok := parseDay(tt.input)
		if ok != tt.ok || (ok && got != tt.want) {
			t.Errorf("parseDay(%q) = %v, %v; want %v, %v", tt.input, got, ok, tt.want, tt.ok)
		}
	}
}

func TestWeekdayJobNext_SameWeek(t *testing.T) {
	bogotá, _ := time.LoadLocation("America/Bogota")
	// 2026-07-15 é terça-feira, 09:00 BRT
	now := time.Date(2026, 7, 15, 9, 0, 0, 0, bogotá)
	j := &WeekdayJob{
		day:      time.Friday,
		hour:     10,
		minute:   0,
		location: bogotá,
	}

	got := j.Next(now)
	want := time.Date(2026, 7, 17, 10, 0, 0, 0, bogotá)
	if !got.Equal(want) {
		t.Errorf("Next(%v) = %v; want %v", now, got, want)
	}
}

func TestWeekdayJobNext_SameDayBeforeTime(t *testing.T) {
	bogotá, _ := time.LoadLocation("America/Bogota")
	// 2026-07-17 é sexta-feira, 09:59 BRT — antes do horário
	now := time.Date(2026, 7, 17, 9, 59, 0, 0, bogotá)
	j := &WeekdayJob{
		day:      time.Friday,
		hour:     10,
		minute:   0,
		location: bogotá,
	}

	got := j.Next(now)
	want := time.Date(2026, 7, 17, 10, 0, 0, 0, bogotá)
	if !got.Equal(want) {
		t.Errorf("Next(%v) = %v; want %v", now, got, want)
	}
}

func TestWeekdayJobNext_SameDayExactTime(t *testing.T) {
	bogotá, _ := time.LoadLocation("America/Bogota")
	// Exatamente no horário agendado
	now := time.Date(2026, 7, 17, 10, 0, 0, 0, bogotá)
	j := &WeekdayJob{
		day:      time.Friday,
		hour:     10,
		minute:   0,
		location: bogotá,
	}

	got := j.Next(now)
	want := time.Date(2026, 7, 17, 10, 0, 0, 0, bogotá)
	if !got.Equal(want) {
		t.Errorf("Next(%v) = %v; want %v", now, got, want)
	}
}

func TestWeekdayJobNext_SameDayAfterTime(t *testing.T) {
	bogotá, _ := time.LoadLocation("America/Bogota")
	// 2026-07-17 é sexta-feira, 10:01 BRT — 1 minuto depois do horário
	now := time.Date(2026, 7, 17, 10, 1, 0, 0, bogotá)
	j := &WeekdayJob{
		day:      time.Friday,
		hour:     10,
		minute:   0,
		location: bogotá,
	}

	got := j.Next(now)
	// Deve pular para a próxima sexta (24/07) porque o horário de hoje já passou
	want := time.Date(2026, 7, 24, 10, 0, 0, 0, bogotá)
	if !got.Equal(want) {
		t.Errorf("Next(%v) = %v; want %v", now, got, want)
	}
}

func TestWeekdayJobNext_NextWeek(t *testing.T) {
	bogotá, _ := time.LoadLocation("America/Bogota")
	// Quarta-feira, o próximo dia desejado (sexta) é em 2 dias
	now := time.Date(2026, 7, 15, 14, 30, 0, 0, bogotá)
	j := &WeekdayJob{
		day:      time.Friday,
		hour:     10,
		minute:   0,
		location: bogotá,
	}

	got := j.Next(now)
	want := time.Date(2026, 7, 17, 10, 0, 0, 0, bogotá)
	if !got.Equal(want) {
		t.Errorf("Next(%v) = %v; want %v", now, got, want)
	}
}

func TestWeekdayJobNext_SkipFullWeek(t *testing.T) {
	bogotá, _ := time.LoadLocation("America/Bogota")
	// Sexta-feira 10:01 — o horário de hoje já passou, pula 7 dias
	now := time.Date(2026, 7, 17, 10, 1, 0, 0, bogotá)
	j := &WeekdayJob{
		day:      time.Friday,
		hour:     10,
		minute:   0,
		location: bogotá,
	}

	got := j.Next(now)
	want := time.Date(2026, 7, 24, 10, 0, 0, 0, bogotá)
	if !got.Equal(want) {
		t.Errorf("Next(%v) = %v; want %v", now, got, want)
	}
}

func TestWeekdayJobNext_MinuteOffset(t *testing.T) {
	bogotá, _ := time.LoadLocation("America/Bogota")
	// Quinta-feira 14:30, job é sexta 10:30
	now := time.Date(2026, 7, 16, 14, 30, 0, 0, bogotá)
	j := &WeekdayJob{
		day:      time.Friday,
		hour:     10,
		minute:   30,
		location: bogotá,
	}

	got := j.Next(now)
	want := time.Date(2026, 7, 17, 10, 30, 0, 0, bogotá)
	if !got.Equal(want) {
		t.Errorf("Next(%v) = %v; want %v", now, got, want)
	}
}
