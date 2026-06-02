package admin

import (
	"context"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

func RestartCommand(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
	if err := whatsapp.Reply(ctx, client, evt, "🔄 Reiniciando... já volto!"); err != nil {
		return fmt.Errorf("falha ao enviar mensagem antes do restart: %w", err)
	}

	time.Sleep(500 * time.Millisecond)

	client.Disconnect()

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("falha ao obter caminho do executável: %w", err)
	}

	// syscall.Exec substitui o processo atual pelo binário (mesmo PID).
	// Isso funciona em qualquer host Linux porque mantém o mesmo processo
	// para o systemd/supervisor, herdando stdin/stdout/stderr e env vars.
	if err := syscall.Exec(execPath, os.Args, os.Environ()); err != nil {
		return fmt.Errorf("falha ao reiniciar: %w", err)
	}

	return nil
}
