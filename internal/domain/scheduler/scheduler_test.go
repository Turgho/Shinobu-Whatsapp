package scheduler

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"
)

// mockJob implementa Job para testes.
type mockJob struct {
	name     string
	next     time.Time
	nextFunc func(now time.Time) time.Time
	ran      atomic.Bool
	runFunc  func()
}

func (m *mockJob) Name() string { return m.name }

func (m *mockJob) Next(now time.Time) time.Time {
	if m.nextFunc != nil {
		return m.nextFunc(now)
	}
	return m.next
}

func (m *mockJob) Run(_ context.Context) error {
	m.ran.Store(true)
	if m.runFunc != nil {
		m.runFunc()
	}
	return nil
}

func newTestScheduler() *Scheduler {
	return NewScheduler(zap.NewNop())
}

// registerAndSetNext registra o job e força nextRun para o valor desejado,
// isolando checkAndRun do comportamento de Next() no Register.
func registerAndSetNext(s *Scheduler, job Job, nextRun time.Time) {
	s.Register(job)
	s.jobs[len(s.jobs)-1].nextRun = nextRun
}

// waitJob espera o job terminar — a execução agora é assíncrona (cada job
// roda em goroutine própria).
func waitJob(t *testing.T, job *mockJob) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if job.ran.Load() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// waitShort dá tempo de um job NÃO disparar (usado nos testes negativos).
func waitShort() { time.Sleep(150 * time.Millisecond) }

// weeklyNext simula Next() de um job semanal.
func weeklyNext(nextFixed time.Time) func(time.Time) time.Time {
	return func(now time.Time) time.Time {
		if !now.After(nextFixed) {
			return nextFixed
		}
		return nextFixed.AddDate(0, 0, 7)
	}
}

// --- checkAndRun tests ---

func TestCheckAndRun_FiresAtScheduledTime(t *testing.T) {
	s := newTestScheduler()
	scheduled := time.Date(2026, 7, 17, 10, 0, 0, 0, time.Local)

	job := &mockJob{name: "test", next: scheduled}
	registerAndSetNext(s, job, scheduled)

	s.checkAndRun(context.Background(), scheduled)
	waitJob(t, job)

	if !job.ran.Load() {
		t.Error("job should have fired at scheduled time")
	}
}

// Teste crítico: reproduz o bug original.
func TestCheckAndRun_FiresWhenTickIsAfterScheduledTime(t *testing.T) {
	s := newTestScheduler()
	scheduled := time.Date(2026, 7, 17, 10, 0, 0, 0, time.Local)

	job := &mockJob{
		name:     "sextou",
		next:     scheduled,
		nextFunc: weeklyNext(scheduled),
	}
	registerAndSetNext(s, job, scheduled)

	// Ticker dispara 5 segundos depois do horário agendado
	s.checkAndRun(context.Background(), scheduled.Add(5*time.Second))
	waitJob(t, job)

	if !job.ran.Load() {
		t.Error("job should have fired when tick arrived 5s after scheduled time")
	}
}

func TestCheckAndRun_DoesNotFireBeforeScheduledTime(t *testing.T) {
	s := newTestScheduler()
	scheduled := time.Date(2026, 7, 17, 10, 0, 0, 0, time.Local)

	job := &mockJob{name: "test", next: scheduled}
	registerAndSetNext(s, job, scheduled)

	s.checkAndRun(context.Background(), scheduled.Add(-5*time.Second))
	waitShort()

	if job.ran.Load() {
		t.Error("job should NOT have fired before scheduled time")
	}
}

func TestCheckAndRun_DoesNotFireTwice(t *testing.T) {
	s := newTestScheduler()
	scheduled := time.Date(2026, 7, 17, 10, 0, 0, 0, time.Local)
	// O job roda em goroutine própria → contador sincronizado.
	var mu sync.Mutex
	runCount := 0
	countRuns := func() int {
		mu.Lock()
		defer mu.Unlock()
		return runCount
	}

	job := &mockJob{
		name:     "test",
		nextFunc: weeklyNext(scheduled),
		runFunc: func() {
			mu.Lock()
			runCount++
			mu.Unlock()
		},
	}
	registerAndSetNext(s, job, scheduled)

	// Primeiro tick: executa
	s.checkAndRun(context.Background(), scheduled)
	waitJob(t, job)
	if got := countRuns(); got != 1 {
		t.Fatalf("expected 1 run, got %d", got)
	}

	// Segundo tick 1s depois: nextRun agora é 24/07, now (17/07 + 1s) < 24/07
	s.checkAndRun(context.Background(), scheduled.Add(1*time.Second))
	if got := countRuns(); got != 1 {
		t.Fatalf("should not have run twice, got %d", got)
	}
}

func TestCheckAndRun_RecalculatesNextRunAfterExecution(t *testing.T) {
	s := newTestScheduler()
	scheduled := time.Date(2026, 7, 17, 10, 0, 0, 0, time.Local)
	nextWeek := scheduled.AddDate(0, 0, 7)

	job := &mockJob{
		name:     "test",
		nextFunc: weeklyNext(scheduled),
	}
	registerAndSetNext(s, job, scheduled)

	s.checkAndRun(context.Background(), scheduled)
	waitJob(t, job)
	if !job.ran.Load() {
		t.Fatal("job should have fired")
	}

	// Verifica que nextRun avançou para a próxima semana
	s.mu.Lock()
	entry := s.jobs[0]
	s.mu.Unlock()
	if !entry.nextRun.Equal(nextWeek) {
		t.Errorf("nextRun should be %v, got %v", nextWeek, entry.nextRun)
	}
}

func TestCheckAndRun_WeekdayJobIntegration(t *testing.T) {
	loc, _ := time.LoadLocation("America/Sao_Paulo")

	// Sexta-feira 09:59:50 — 10 segundos antes do horário
	startTime := time.Date(2026, 7, 17, 9, 59, 50, 0, loc)
	scheduled := time.Date(2026, 7, 17, 10, 0, 0, 0, loc)

	job := &mockJob{
		name:     "sextou",
		nextFunc: weeklyNext(scheduled),
	}

	s := newTestScheduler()
	registerAndSetNext(s, job, scheduled)

	// Simula ticks a cada 15 segundos a partir de 09:59:50
	for tick := startTime; tick.Before(scheduled.Add(30 * time.Second)); tick = tick.Add(15 * time.Second) {
		s.checkAndRun(context.Background(), tick)
	}
	waitJob(t, job)

	if !job.ran.Load() {
		t.Error("sextou job should have fired around 10:00")
	}
}

func TestCheckAndRun_MultipleJobsOnlyFiresDue(t *testing.T) {
	s := newTestScheduler()
	now := time.Date(2026, 7, 17, 10, 0, 0, 0, time.Local)

	job1 := &mockJob{name: "sextou", next: now}
	job2 := &mockJob{name: "aniversario", next: now.Add(2 * time.Hour)}
	registerAndSetNext(s, job1, now)
	registerAndSetNext(s, job2, now.Add(2*time.Hour))

	s.checkAndRun(context.Background(), now)
	waitJob(t, job1)

	if !job1.ran.Load() {
		t.Error("sextou should have fired")
	}
	if job2.ran.Load() {
		t.Error("aniversario should NOT have fired")
	}
}

func TestCheckAndRun_RemovesOneShotJobAfterExecution(t *testing.T) {
	s := newTestScheduler()
	now := time.Date(2026, 7, 17, 10, 0, 0, 0, time.Local)

	// nextFunc e runFunc rodam em goroutines diferentes (o job roda em
	// goroutine própria) — o estado precisa de sincronização.
	var mu sync.Mutex
	oneShotDone := false
	job := &mockJob{
		name: "one-shot",
		nextFunc: func(_ time.Time) time.Time {
			mu.Lock()
			defer mu.Unlock()
			if oneShotDone {
				return time.Time{}
			}
			return now
		},
		runFunc: func() {
			mu.Lock()
			oneShotDone = true
			mu.Unlock()
		},
	}
	registerAndSetNext(s, job, now)

	s.mu.Lock()
	registered := len(s.jobs)
	s.mu.Unlock()
	if registered != 1 {
		t.Fatalf("expected 1 job registered, got %d", registered)
	}

	s.checkAndRun(context.Background(), now)
	waitJob(t, job)

	s.mu.Lock()
	jobsLeft := len(s.jobs)
	s.mu.Unlock()
	if jobsLeft != 0 {
		t.Errorf("one-shot job should have been removed, got %d jobs", jobsLeft)
	}
}

func TestCheckAndRun_TickBetweenTwoJobs(t *testing.T) {
	s := newTestScheduler()
	scheduled := time.Date(2026, 7, 17, 10, 0, 0, 0, time.Local)

	job1 := &mockJob{name: "job1", next: scheduled}
	job2 := &mockJob{name: "job2", next: scheduled}
	registerAndSetNext(s, job1, scheduled)
	registerAndSetNext(s, job2, scheduled)

	s.checkAndRun(context.Background(), scheduled.Add(30*time.Second))
	waitJob(t, job1)
	waitJob(t, job2)

	if !job1.ran.Load() || !job2.ran.Load() {
		t.Error("both jobs should have fired")
	}
}

// TestCheckAndRun_BirthdayJobSamePattern testa o padrão do BirthdayJob:
// Next retorna tomorrow 08:00 quando now > today 08:00.
func TestCheckAndRun_BirthdayJobSamePattern(t *testing.T) {
	s := newTestScheduler()
	scheduled := time.Date(2026, 7, 18, 8, 0, 0, 0, time.Local)

	job := &mockJob{
		name: "birthday",
		nextFunc: func(now time.Time) time.Time {
			today8am := time.Date(now.Year(), now.Month(), now.Day(), 8, 0, 0, 0, now.Location())
			if now.After(today8am) {
				return today8am.AddDate(0, 0, 1)
			}
			return today8am
		},
	}
	registerAndSetNext(s, job, scheduled)

	// Ticker às 08:00:10 (10s depois) — deve executar
	s.checkAndRun(context.Background(), scheduled.Add(10*time.Second))
	waitJob(t, job)

	if !job.ran.Load() {
		t.Error("birthday job should have fired 10s after scheduled time")
	}
}
