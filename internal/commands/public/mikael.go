package public

import (
	"context"
	"fmt"

	"github.com/Turgho/Shinobu-Whatsapp/internal/commands"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/history"
	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/configs"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// mikaelLid é o lid (user JID) do Mikael.
const mikaelLid = "5511999998888" // Substitua pelo lid real do Mikael

func MikaelCommand(
	ctx context.Context,
	client *whatsmeow.Client,
	evt *events.Message,
	_ []string,
	store *history.Store,
	cfg *configs.MikaelConfig,
) error {
	chat := evt.Info.Chat.String()
	if !isMikaelGroup(chat, cfg) {
		return whatsapp.Reply(ctx, client, evt, "❌ Este comando só funciona em grupos autorizados.")
	}

	count, err := store.CountWordInMessages(ctx, chat, mikaelLid, "pix")
	if err != nil {
		return whatsapp.Reply(ctx, client, evt, "❌ Não consegui contar as mensagens.")
	}

	return whatsapp.Reply(ctx, client, evt, fmt.Sprintf("📊 Mikael escreveu 'pix' %d vezes neste grupo.", count))
}

func isMikaelGroup(chat string, cfg *configs.MikaelConfig) bool {
	if cfg == nil {
		return false
	}
	for _, group := range cfg.Groups {
		if group == chat {
			return true
		}
	}
	return false
}

func MikaelHandler(store *history.Store, cfg *configs.MikaelConfig) commands.HandlerFunc {
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		return MikaelCommand(ctx, client, evt, args, store, cfg)
	}
}
