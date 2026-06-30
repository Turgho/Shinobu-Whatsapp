package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/gosafe"
	"go.uber.org/zap"
)

// Job é uma tarefa agendável. Next retorna a próxima execução (zero time para one-shot já executado).
type Job interface {
	Name() string
	Next(now time.Time) time.Time
	Run(ctx context.Context) error
}

// Scheduler gerencia execução periódica de jobs com ticker de 15 segundos.
type Scheduler struct {
	mu     sync.Mutex
	jobs   []*jobEntry
	logger *zap.Logger
	stopCh chan struct{}
}

type jobEntry struct {
	job     Job
	lastRun time.Time
}

// NewScheduler cria um scheduler vazio. Deve ser iniciado com Start.
func NewScheduler(logger *zap.Logger) *Scheduler {
	return &Scheduler{
		jobs:   make([]*jobEntry, 0),
		logger: logger.Named("SCHEDULER"),
		stopCh: make(chan struct{}),
	}
}

// Register adiciona um job ao scheduler. Jobs one-shot são removidos após execução.
func (s *Scheduler) Register(job Job) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.jobs = append(s.jobs, &jobEntry{job: job})
	next := job.Next(time.Now())
	s.logger.Info("Job registrado",
		zap.String("job", job.Name()),
		zap.String("next_run", next.Format("2006-01-02 15:04 MST")),
	)
}

// Unregister remove um job pelo nome.
func (s *Scheduler) Unregister(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.unregisterLocked(name)
}

func (s *Scheduler) unregisterLocked(name string) {
	for i, entry := range s.jobs {
		if entry.job.Name() == name {
			s.jobs = append(s.jobs[:i], s.jobs[i+1:]...)
			s.logger.Info("Job desregistrado", zap.String("job", name))
			return
		}
	}
}

// Start inicia o loop do scheduler em uma goroutine segura (gosafe.Go).
func (s *Scheduler) Start(ctx context.Context) {
	gosafe.Go(s.logger, func() {
		s.run(ctx)
	})
}

// Stop sinaliza o scheduler para parar na próxima iteração.
func (s *Scheduler) Stop() {
	select {
	case <-s.stopCh:
	default:
		close(s.stopCh)
	}
}

func (s *Scheduler) run(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	s.logger.Info("Scheduler iniciado", zap.Duration("interval", 15*time.Second))

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

// RunCheck força uma verificação imediata dos jobs (chamado após registrar um job novo).
func (s *Scheduler) RunCheck() {
	s.checkAndRun(time.Now())
}

// checkAndRun itera sobre os jobs registrados, executa os que estão no horário.
// O mutex é liberado durante a execução do job (linhas Unlock/Lock) para não
// bloquear registro de novos jobs ou o ticker enquanto um job roda.
func (s *Scheduler) checkAndRun(now time.Time) {
	s.mu.Lock()

	var toRemove []string
	for _, entry := range s.jobs {
		next := entry.job.Next(now)
		if now.After(next) || now.Equal(next) {
			if !entry.lastRun.IsZero() && entry.lastRun.After(next) {
				continue
			}

			s.logger.Info("Executando job",
				zap.String("job", entry.job.Name()),
				zap.Time("run_at", next),
			)

			s.mu.Unlock()
			jobCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			start := time.Now()
			func() {
				defer func() {
					if r := recover(); r != nil {
						s.logger.Error("Job panickou",
							zap.String("job", entry.job.Name()),
							zap.Any("panic", r),
						)
					}
				}()
				if err := entry.job.Run(jobCtx); err != nil {
					s.logger.Error("Erro no job",
						zap.String("job", entry.job.Name()),
						zap.Error(err),
					)
				} else {
					s.logger.Info("Job executado",
						zap.String("job", entry.job.Name()),
						zap.Duration("duration", time.Since(start)),
					)
				}
			}()
			cancel()
			s.mu.Lock()

			entry.lastRun = now

			if entry.job.Next(now).IsZero() {
				toRemove = append(toRemove, entry.job.Name())
				s.logger.Info("Job removido após execução",
					zap.String("job", entry.job.Name()),
				)
			}
		}
	}

	for _, name := range toRemove {
		s.unregisterLocked(name)
	}
	s.mu.Unlock()
}
