package admin

import (
	"context"

	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

type maintenanceToggler interface {
	SetMaintenance(bool)
	IsMaintenance() bool
}

func ManutencaoCommand(toggler maintenanceToggler) func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		on := !toggler.IsMaintenance()
		toggler.SetMaintenance(on)

		if on {
			return whatsapp.Reply(ctx, client, evt, "🔧 *Modo manutenção ativado.* Nenhum comando será processado até eu ser desativado.")
		}
		return whatsapp.Reply(ctx, client, evt, "✅ *Modo manutenção desativado.* Comandos liberados novamente.")
	}
}
