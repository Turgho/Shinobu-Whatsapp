package public

import (
	"context"
	"fmt"

	"github.com/Turgho/Shinobu-Whatsapp/internal/commands"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/joke"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

func PiadaCommand(
	ctx context.Context,
	client *whatsmeow.Client,
	evt *events.Message,
	_ []string,
	jokeClient *joke.JokeClient,
) error {
	if err := jokeClient.Ping(ctx); err != nil {
		return whatsapp.Reply(ctx, client, evt, msgPiadaFail)
	}

	j, err := jokeClient.Fetch(ctx)
	if err != nil {
		return whatsapp.Reply(ctx, client, evt, msgPiadaFail)
	}

	var text string
	switch j.Type {
	case "twopart":
		text = fmt.Sprintf("%s\n\n%s", j.Setup, j.Delivery)
	case "single":
		text = j.Joke
	}

	return whatsapp.Reply(ctx, client, evt, text)
}

func PiadaHandler(jokeClient *joke.JokeClient) commands.HandlerFunc {
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		return PiadaCommand(ctx, client, evt, args, jokeClient)
	}
}
