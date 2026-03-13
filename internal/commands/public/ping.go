package public

import (
	"fmt"
	"time"

	"github.com/Turgho/YuukoWhatsapp/internal/utils"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// Responde com "Pong!" quando o comando "ping" for recebido
func PingCommand(client *whatsmeow.Client, evt *events.Message, args []string) error {
	start := time.Now()

	err := utils.Reply(client, evt, "🏓 Pong!")
	if err != nil {
		return err
	}

	latency := time.Since(start).Milliseconds()

	return utils.Reply(client, evt, fmt.Sprintf("📡 Latência : `%dms`", latency))
}
