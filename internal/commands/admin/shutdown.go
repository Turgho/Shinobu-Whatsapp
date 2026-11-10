package admin

import (
	"context"
	"fmt"
	"time"

	"github.com/Turgho/Shinobu-Whatsapp/internal/commands"

	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// ShutdownCommand encerra o bot de forma limpa: pede o encerramento via
// hook (o app.Run cancela o contexto e roda os defer). Antes usava
// os.Exit(0), que pulava logger.Sync() e db.Close().
func ShutdownCommand(requestShutdown func() bool) commands.HandlerFunc {
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		// A mensagem de despedida vai antes de encerrar.
		if err := whatsapp.Reply(ctx, client, evt, "A mimir patrão 😴..."); err != nil {
			return fmt.Errorf("falha ao enviar mensagem antes do shutdown: %w", err)
		}

		// Aguarda meio segundo para garantir envio da mensagem
		time.Sleep(500 * time.Millisecond)

		if !requestShutdown() {
			return whatsapp.Reply(ctx, client, evt, "❌ Não consegui encerrar agora.")
		}
		return nil
	}
}
