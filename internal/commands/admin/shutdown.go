package admin

import (
	"fmt"
	"os"
	"time"

	"github.com/Turgho/YuukoWhatsapp/internal/utils"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// ShutdownCommand desconecta o bot e encerra a aplicação
func ShutdownCommand(client *whatsmeow.Client, evt *events.Message, args []string) error {
	// Envia mensagem de despedida
	if err := utils.Reply(client, evt, "A mimir patrão 😴..."); err != nil {
		return fmt.Errorf("falha ao enviar mensagem antes do shutdown: %w", err)
	}

	// Aguarda meio segundo para garantir envio da mensagem
	time.Sleep(500 * time.Millisecond)

	// Desconecta o client (ignora erro, pois a aplicação vai sair)
	client.Disconnect()

	// Sai da aplicação com código 0 (sem erro)
	os.Exit(0)

	return nil // nunca vai chegar aqui
}
