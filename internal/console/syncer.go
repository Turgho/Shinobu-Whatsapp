package console

import (
	"context"
	"sort"
	"time"

	"github.com/Turgho/Shinobu-Whatsapp/internal/commands"
	"go.uber.org/zap"
)

// Intervalos da sincronização.
const (
	// PollInterval é de quanto em quanto tempo o bot busca o estado
	// desejado no Console (latência máxima de um toggle vindo do Console).
	PollInterval = 15 * time.Second
	// ReportInterval é o intervalo do report (catálogo + estado aplicado).
	ReportInterval = 60 * time.Second
)

// Syncer mantém o estado dos comandos do bot em sincronia com o Console.
type Syncer struct {
	client *Client
	router *commands.Router
	log    *zap.Logger
}

// NewSyncer cria o sincronizador.
func NewSyncer(client *Client, router *commands.Router, log *zap.Logger) *Syncer {
	if log == nil {
		log = zap.NewNop()
	}
	return &Syncer{client: client, router: router, log: log.Named("SYNC")}
}

// Start roda a sincronização em background (goroutine com recover).
func (s *Syncer) Start(ctx context.Context) {
	if !s.client.Enabled() {
		s.log.Info("Sincronização com o Console desativada (sem CONSOLE_URL/CONSOLE_TOKEN)")
		return
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.log.Error("panic na sincronização", zap.Any("panic", r))
			}
		}()

		ticker := time.NewTicker(PollInterval)
		defer ticker.Stop()

		lastReport := time.Time{}
		for {
			s.poll(ctx)
			if time.Since(lastReport) >= ReportInterval {
				if err := s.report(ctx); err != nil {
					s.log.Warn("Falha ao reportar estado ao Console", zap.Error(err))
				} else {
					lastReport = time.Now()
				}
			}

			select {
			case <-ctx.Done():
				s.log.Info("Sincronização encerrada")
				return
			case <-ticker.C:
			}
		}
	}()
}

// poll busca o estado desejado e aplica no bot.
func (s *Syncer) poll(ctx context.Context) {
	state, err := s.client.FetchState(ctx)
	if err != nil {
		s.log.Warn("Falha ao buscar estado no Console", zap.Error(err))
		return
	}

	changed, err := s.router.CommandState().Apply(state.Disabled)
	if err != nil {
		s.log.Error("Falha ao aplicar estado dos comandos", zap.Error(err))
		return
	}
	if changed {
		s.log.Info("Estado dos comandos aplicado",
			zap.Strings("desabilitados", state.Disabled),
		)
	}
}

// report manda o catálogo e o estado aplicado.
func (s *Syncer) report(ctx context.Context) error {
	metas := s.router.CommandCatalog()
	catalog := make([]CatalogCommand, 0, len(metas))
	for _, meta := range metas {
		catalog = append(catalog, CatalogCommand{
			Name:        meta.Name,
			Description: meta.Description,
			Type:        string(meta.Type),
			Private:     meta.Private,
		})
	}
	sort.Slice(catalog, func(i, j int) bool { return catalog[i].Name < catalog[j].Name })

	disabled := s.router.CommandState().Disabled()
	sort.Strings(disabled)

	return s.client.Report(ctx, Report{
		Catalog:   catalog,
		Disabled:  disabled,
		AppliedAt: time.Now().UTC().Format(time.RFC3339),
	})
}
