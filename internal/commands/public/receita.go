package public

import (
	"context"
	"fmt"
	"strings"

	"github.com/Turgho/Shinobu-Whatsapp/internal/commands"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/history"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/ia"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

func ReceitaCommand(
	ctx context.Context,
	client *whatsmeow.Client,
	evt *events.Message,
	args []string,
	aiCfg *ia.Config,
) error {
	if len(args) == 0 {
		return whatsapp.Reply(ctx, client, evt, msgReceitaNoQuery)
	}

	query := strings.Join(args, " ")
	tavilyQuery := query + " receita ingredientes modo de preparo"

	webContext, err := ia.SearchWeb(ctx, aiCfg.HTTPClient, aiCfg.TavilyKey, tavilyQuery)
	if err != nil || webContext == "" {
		return whatsapp.Reply(ctx, client, evt, msgReceitaFail)
	}

	system := "Você é um chef de cozinha que explica receitas de forma clara e direta. Extraia os ingredientes e o modo de preparo das informações fornecidas."
	user := fmt.Sprintf("Monte a receita para: %s\n\nInformações encontradas:\n%s\n\nFormato:\n🍳 Receita de %s\nIngredientes:\n- item 1\n- item 2\n\nModo de preparo:\n1. passo 1\n2. passo 2", query, webContext, query)

	msgs := []history.IAMessage{
		{Role: "system", Content: system},
		{Role: "user", Content: user},
	}

	answer, err := ia.QuickChat(ctx, aiCfg.HTTPClient, aiCfg.GroqURL, aiCfg.GroqKey, ia.ModelScoutFast, msgs, 0.72, 800)
	if err != nil {
		return whatsapp.Reply(ctx, client, evt, msgReceitaFail)
	}

	return whatsapp.Reply(ctx, client, evt, strings.TrimSpace(answer))
}

func ReceitaHandler(aiCfg *ia.Config) commands.HandlerFunc {
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		return ReceitaCommand(ctx, client, evt, args, aiCfg)
	}
}
