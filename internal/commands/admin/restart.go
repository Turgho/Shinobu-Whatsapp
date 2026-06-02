package admin

import (
	"context"
	"fmt"
	"os"
	"os/exec"
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

	cmd := exec.Command(execPath, os.Args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("falha ao iniciar novo processo: %w", err)
	}

	os.Exit(0)
	return nil
}
