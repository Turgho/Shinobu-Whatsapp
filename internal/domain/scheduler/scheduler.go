package scheduler

import (
	"context"
	"time"

	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/gosafe"
	"go.uber.org/zap"
)

type Job interface {
	Name() string
	Next(now time.Time) time.Time
	Run(ctx context.Context) error
}

type Scheduler struct {
	jobs   []*jobEntry
	logger *zap.Logger
	stopCh chan struct{}
}

type jobEntry struct {
	job     Job
	lastRun time.Time
}

func NewScheduler(logger *zap.Logger) *Scheduler {
	return &Scheduler{
		jobs:   make([]*jobEntry, 0),
		logger: logger.Named("SCHEDULER"),
		stopCh: make(chan struct{}),
	}
}

func (s *Scheduler) Register(job Job) {
	s.jobs = append(s.jobs, &jobEntry{job: job})
	next := job.Next(time.Now())
	s.logger.Info("Job registrado",
		zap.String("job", job.Name()),
		zap.String("next_run", next.Format("2006-01-02 15:04 MST")),
	)
}

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
					s.logger.Error("Erro ao executar job",
						zap.String("job", entry.job.Name()),
						zap.Error(err),
					)
				} else {
					s.logger.Info("Job executado com sucesso",
						zap.String("job", entry.job.Name()),
					)
				}
			}()
			cancel()

			entry.lastRun = now
		}
	}
}
