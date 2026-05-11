package public

import (
	"context"
	"fmt"
	"time"

	"github.com/Turgho/YuukoWhatsapp/internal/utils"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// PingCommand responde com latência estimada desde o timestamp da mensagem recebida.
func PingCommand(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
	latency := time.Since(evt.Info.Timestamp).Milliseconds()

	return utils.SendText(ctx, client, evt, fmt.Sprintf("🏓 Pong!\n📡 Latência: `%dms`", latency), true)
}
