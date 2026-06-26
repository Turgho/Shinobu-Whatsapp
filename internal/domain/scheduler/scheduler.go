package scheduler

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Job é a interface que define um trabalho agendado.
type Job interface {
	Name() string
	Next(now time.Time) time.Time
	Run(ctx context.Context) error
}

// Scheduler gerencia a execução de jobs agendados.
type Scheduler struct {
	jobs     []*jobEntry
	logger   *zap.Logger
	stopCh   chan struct{}
	stopOnce sync.Once
	wg       sync.WaitGroup
}

type jobEntry struct {
	job     Job
	lastRun time.Time
}

// NewScheduler cria um novo scheduler.
func NewScheduler(logger *zap.Logger) *Scheduler {
	return &Scheduler{
		jobs:   make([]*jobEntry, 0),
		logger: logger.Named("SCHEDULER"),
		stopCh: make(chan struct{}),
	}
}

// Register registra um novo job no scheduler.
func (s *Scheduler) Register(job Job) {
	s.jobs = append(s.jobs, &jobEntry{job: job})
	s.logger.Info("Job registrado", zap.String("job", job.Name()))
}

// Start inicia o loop do scheduler em uma goroutine.
// Verifica a cada minuto quais jobs devem ser executados.
func (s *Scheduler) Start(ctx context.Context) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.run(ctx)
	}()
}

// Stop para o scheduler e aguarda a finalização da goroutine.
func (s *Scheduler) Stop() {
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
	s.wg.Wait()
}

func (s *Scheduler) run(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	s.logger.Info("Scheduler iniciado")

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Scheduler parado (context cancelado)")
			return
		case <-s.stopCh:
			s.logger.Info("Scheduler parado")
			return
		case now := <-ticker.C:
			s.checkAndRun(now)
		}
	}
}

func (s *Scheduler) checkAndRun(now time.Time) {
	for _, entry := range s.jobs {
		next := entry.job.Next(now)
		if now.After(next) || now.Equal(next) {
			if !entry.lastRun.IsZero() && entry.lastRun.After(next) {
				continue
			}

			s.logger.Info("Executando job",
				zap.String("job", entry.job.Name()),
				zap.Time("scheduled", next),
			)

			jobCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			if err := entry.job.Run(jobCtx); err != nil {
				s.logger.Error("Erro ao executar job",
					zap.String("job", entry.job.Name()),
					zap.Error(err),
				)
			} else {
				s.logger.Info("Job executado com sucesso",
					zap.String("job", entry.job.Name()),
				)
			}
			cancel()

			entry.lastRun = now
		}
	}
}
