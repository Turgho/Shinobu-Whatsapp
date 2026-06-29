package public

import (
	"context"
	"strings"

	"github.com/Turgho/Shinobu-Whatsapp/internal/commands"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/history"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/ia"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

func FatoCommand(
	ctx context.Context,
	client *whatsmeow.Client,
	evt *events.Message,
	_ []string,
	aiCfg *ia.Config,
) error {
	msgs := []history.IAMessage{
		{Role: "system", Content: "Compartilhe um fato curioso e surpreendente em português. Máximo 3 frases. Comece diretamente com o fato."},
	}

	answer, err := ia.QuickChat(ctx, aiCfg.HTTPClient, aiCfg.GroqURL, aiCfg.GroqKey, ia.ModelScoutFast, msgs, 0.85, 150)
	if err != nil {
		return whatsapp.Reply(ctx, client, evt, msgFatoFail)
	}

	return whatsapp.Reply(ctx, client, evt, "🤯 "+strings.TrimSpace(answer))
}

func FatoHandler(aiCfg *ia.Config) commands.HandlerFunc {
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		return FatoCommand(ctx, client, evt, args, aiCfg)
	}
}
