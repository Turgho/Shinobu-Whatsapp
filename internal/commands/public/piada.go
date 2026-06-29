package public

import (
	"context"
	"fmt"
	"math/rand"
	"strings"

	"github.com/Turgho/Shinobu-Whatsapp/internal/commands"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/history"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/ia"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// piadaQueries define consultas variadas para buscar piadas na web.
var piadaQueries = []string{
	"piadas curtas engraçadas português brasileiro",
	"melhores piadas de humor brasileiro",
	"piadas de situacao engraçadas",
	"piadas limpas para WhatsApp",
	"piadas de cotidiano engraçadas",
	"piadas rápidas para fazer rir",
}

func PiadaCommand(
	ctx context.Context,
	client *whatsmeow.Client,
	evt *events.Message,
	_ []string,
	aiCfg *ia.Config,
) error {
	var webContext string

	if aiCfg.TavilyKey != "" {
		query := piadaQueries[rand.Intn(len(piadaQueries))]
		result, err := ia.SearchWeb(ctx, aiCfg.TavilyKey, query)
		if err == nil && result != "" {
			webContext = result
		}
	}

	var msgs []history.IAMessage

	if webContext != "" {
		msgs = []history.IAMessage{
			{Role: "system", Content: msgPiadaSystemWeb},
			{Role: "user", Content: fmt.Sprintf("Escolha UMA piada deste contexto e conte em português brasileiro:\n\n%s", webContext)},
		}
	} else {
		msgs = []history.IAMessage{
			{Role: "system", Content: msgPiadaSystem},
		}
	}

	answer, err := ia.QuickChat(ctx, aiCfg.HTTPClient, aiCfg.GroqURL, aiCfg.GroqKey, ia.ModelScoutFast, msgs, 0.85, 150)
	if err != nil {
		return whatsapp.Reply(ctx, client, evt, msgPiadaFail)
	}

	answer = strings.TrimSpace(answer)
	if answer == "" {
		return whatsapp.Reply(ctx, client, evt, msgPiadaFail)
	}

	return whatsapp.Reply(ctx, client, evt, answer)
}

func PiadaHandler(aiCfg *ia.Config) commands.HandlerFunc {
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		return PiadaCommand(ctx, client, evt, args, aiCfg)
	}
}
