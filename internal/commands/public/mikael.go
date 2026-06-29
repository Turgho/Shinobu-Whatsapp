package public

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/Turgho/Shinobu-Whatsapp/internal/commands"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/history"
	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/configs"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

func MikaelCommand(
	ctx context.Context,
	client *whatsmeow.Client,
	evt *events.Message,
	_ []string,
	store *history.Store,
	cfg *configs.MikaelConfig,
) error {
	if cfg.LID == "" {
		return whatsapp.Reply(ctx, client, evt, "❌ LID do Mikael não configurado.")
	}

	chat := evt.Info.Chat.String()
	if !isMikaelGroup(chat, cfg) {
		return whatsapp.Reply(ctx, client, evt, "❌ Este comando só funciona em grupos autorizados.")
	}

	// Normaliza o LID para o formato completo (@s.whatsapp.net)
	lid := cfg.LID
	if !strings.Contains(lid, "@") {
		lid = lid + "@s.whatsapp.net"
	}

	count, err := store.CountWordInMessages(ctx, chat, lid, "pix")
	if err != nil {
		return whatsapp.Reply(ctx, client, evt, "❌ Não consegui contar as mensagens.")
	}

	return whatsapp.Reply(ctx, client, evt, fmt.Sprintf("📊 Mikael escreveu 'pix' %d vezes neste grupo.", count))
}

func isMikaelGroup(chat string, cfg *configs.MikaelConfig) bool {
	if cfg == nil {
		return false
	}
	return slices.Contains(cfg.Groups, chat)
}

func MikaelHandler(store *history.Store, cfg *configs.MikaelConfig) commands.HandlerFunc {
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		return MikaelCommand(ctx, client, evt, args, store, cfg)
	}
}
