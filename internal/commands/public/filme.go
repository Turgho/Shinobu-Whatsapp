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

func FilmeCommand(
	ctx context.Context,
	client *whatsmeow.Client,
	evt *events.Message,
	args []string,
	aiCfg *ia.Config,
) error {
	genero := "qualquer um"
	if len(args) > 0 {
		genero = strings.Join(args, " ")
	}

	msgs := []history.IAMessage{
		{Role: "system", Content: "Recomende um filme para assistir. Se um gênero for fornecido, use-o. Responda com: título, ano, gênero e uma frase de por que vale assistir. Máximo 4 linhas. Sem introduções."},
		{Role: "user", Content: fmt.Sprintf("Recomende um filme. Gênero preferido: %s", genero)},
	}

	answer, err := ia.QuickChat(ctx, aiCfg.HTTPClient, aiCfg.GroqURL, aiCfg.GroqKey, ia.ModelScoutFast, msgs, 0.85, 150)
	if err != nil {
		return whatsapp.Reply(ctx, client, evt, msgFilmeFail)
	}

	reply := fmt.Sprintf("🎬 %s", strings.TrimSpace(answer))
	return whatsapp.Reply(ctx, client, evt, reply)
}

func FilmeHandler(aiCfg *ia.Config) commands.HandlerFunc {
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		return FilmeCommand(ctx, client, evt, args, aiCfg)
	}
}
