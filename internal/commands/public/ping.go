package public

import (
	"context"
	"fmt"
	"time"

	"github.com/Turgho/YuukoWhatsapp/internal/utils"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// Responde com "Pong!" quando o comando "ping" for recebido
func PingCommand(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
	latency := time.Since(evt.Info.Timestamp).Milliseconds()

	return utils.Reply(ctx, client, evt, fmt.Sprintf("🏓 Pong!\n📡 Latência: `%dms`", latency))
}
