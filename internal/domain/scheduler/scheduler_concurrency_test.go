package scheduler

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// O mesmo job não pode ser disparado duas vezes ao mesmo tempo: o ticker,
// o ForceRun e o RunCheck podem coincidir (ex.: !agenda registrando um
// lembrete de 1 minuto na hora do tick).
func TestCheckAndRunNaoDisparaEmParalelo(t *testing.T) {
	s := newTestScheduler()
	scheduled := time.Now().Add(-time.Second)

	var concurrent atomic.Int32
	var maxConcurrent atomic.Int32
	release := make(chan struct{})

	job := &mockJob{
		name: "lento",
		next: scheduled,
		runFunc: func() {
			cur := concurrent.Add(1)
			for {
				old := maxConcurrent.Load()
				if cur <= old || maxConcurrent.CompareAndSwap(old, cur) {
					break
				}
			}
			<-release // segura o job em execução
			concurrent.Add(-1)
		},
	}
	registerAndSetNext(s, job, scheduled)

	// primeiro disparo
	s.checkAndRun(context.Background(), scheduled)
	waitConcurrent(t, &concurrent, 1)

	// segundo disparo enquanto o primeiro ainda roda → deve ser ignorado
	s.checkAndRun(context.Background(), scheduled.Add(time.Second))
	time.Sleep(100 * time.Millisecond)

	if got := maxConcurrent.Load(); got != 1 {
		t.Errorf("job rodou %d vezes em paralelo, esperava 1", got)
	}

	close(release)
}

// ForceRun com job em execução deve falhar em vez de duplicar o envio.
func TestForceRunRecusaJobEmExecucao(t *testing.T) {
	s := newTestScheduler()
	scheduled := time.Now().Add(-time.Second)

	var running atomic.Int32
	release := make(chan struct{})
	job := &mockJob{
		name: "ocupado",
		next: scheduled,
		runFunc: func() {
			running.Store(1)
			<-release
		},
	}
	registerAndSetNext(s, job, scheduled)

	s.checkAndRun(context.Background(), scheduled)
	waitConcurrent(t, &running, 1)

	if err := s.ForceRun(context.Background(), "ocupado"); err == nil {
		t.Error("ForceRun deveria recusar um job já em execução")
	}

	close(release)
}

// waitConcurrent espera o contador chegar ao valor esperado.
func waitConcurrent(t *testing.T, counter *atomic.Int32, want int32) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if counter.Load() == want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
}
