package public

import (
	"context"

	"github.com/Turgho/Shinobu-Whatsapp/internal/commands"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/history"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/ia"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

func PiadaCommand(
	ctx context.Context,
	client *whatsmeow.Client,
	evt *events.Message,
	_ []string,
	aiCfg *ia.Config,
) error {
	msgs := []history.IAMessage{
		{Role: "system", Content: "Conte uma piada curta e engraçada em português brasileiro. Máximo 4 linhas. Sem introduções."},
	}

	answer, err := ia.QuickChat(ctx, aiCfg.HTTPClient, aiCfg.GroqURL, aiCfg.GroqKey, ia.ModelScoutFast, msgs, 0.85, 150)
	if err != nil {
		return whatsapp.Reply(ctx, client, evt, msgPiadaFail)
	}

	return whatsapp.Reply(ctx, client, evt, answer)
}

func PiadaHandler(aiCfg *ia.Config) commands.HandlerFunc {
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		return PiadaCommand(ctx, client, evt, args, aiCfg)
	}
}
