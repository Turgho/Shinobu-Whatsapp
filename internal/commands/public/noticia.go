package public

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Turgho/Shinobu-Whatsapp/internal/commands"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/history"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/ia"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

func NoticiaCommand(
	ctx context.Context,
	client *whatsmeow.Client,
	evt *events.Message,
	_ []string,
	aiCfg *ia.Config,
	loc *time.Location,
) error {
	if aiCfg.TavilyKey == "" {
		return whatsapp.Reply(ctx, client, evt, msgNoticiaFail)
	}

	webContext, err := ia.SearchWeb(ctx, aiCfg.HTTPClient, aiCfg.TavilyKey, "principais notícias do Brasil hoje")
	if err != nil || webContext == "" {
		return whatsapp.Reply(ctx, client, evt, msgNoticiaFail)
	}

	system := msgNoticiaSystem
	user := fmt.Sprintf("Resuma estas notícias:\n\n%s", webContext)

	msgs := []history.IAMessage{
		{Role: "system", Content: system},
		{Role: "user", Content: user},
	}

	answer, err := ia.QuickChat(ctx, aiCfg.HTTPClient, aiCfg.GroqURL, aiCfg.GroqKey, ia.ModelScoutFast, msgs, 0.2, 350)
	if err != nil {
		return whatsapp.Reply(ctx, client, evt, msgNoticiaFail)
	}

	now := time.Now().In(loc).Format("02/01/2006 15:04")
	reply := fmt.Sprintf("📰 Notícias de hoje\n\n%s\n\n🕐 %s", strings.TrimSpace(answer), now)

	return whatsapp.Reply(ctx, client, evt, reply)
}

func NoticiaHandler(aiCfg *ia.Config, loc *time.Location) commands.HandlerFunc {
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		return NoticiaCommand(ctx, client, evt, args, aiCfg, loc)
	}
}
