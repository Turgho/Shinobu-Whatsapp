package admin

import (
	"context"
	"strings"

	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/ignore"
	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/phone"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
	"go.uber.org/zap"
)

func IgnoreCommand(ignoreStore *ignore.Store, logger *zap.Logger) func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
	l := logger.Named("IGNORE")
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		if len(args) == 0 {
			return usageIgnore(ctx, client, evt)
		}

		switch strings.ToLower(args[0]) {
		case "lista":
			return handleIgnoreList(ctx, client, evt, ignoreStore)

		case "remover":
			if len(args) < 2 {
				return whatsapp.SendText(ctx, client, evt,
					"❌ Use: `!ignorar remover <número>`", true)
			}
			removed, err := ignoreStore.Remove(phone.NormalizeNumber(args[1]))
			if err != nil {
				l.Error("falha ao remover da lista de ignorados", zap.Error(err))
				return whatsapp.SendText(ctx, client, evt,
					"❌ Não consegui atualizar a lista de ignorados. Tenta de novo.", true)
			}
			if !removed {
				return whatsapp.SendText(ctx, client, evt,
					"⚠️ Número não estava na lista de ignorados.", true)
			}
			return whatsapp.SendText(ctx, client, evt,
				"✅ Número removido da lista de ignorados.", true)

		default:
			err := ignoreStore.Add(phone.NormalizeNumber(args[0]))
			if err != nil {
				l.Error("falha ao adicionar na lista de ignorados", zap.Error(err))
				return whatsapp.SendText(ctx, client, evt,
					"❌ Não consegui atualizar a lista de ignorados. Tenta de novo.", true)
			}
			return whatsapp.SendText(ctx, client, evt,
				"✅ Número adicionado à lista de ignorados.", true)
		}
	}
}

func handleIgnoreList(ctx context.Context, client *whatsmeow.Client, evt *events.Message, ignoreStore *ignore.Store) error {
	list := ignoreStore.List()
	if len(list) == 0 {
		return whatsapp.SendText(ctx, client, evt,
			"📋 Nenhum número ignorado.", true)
	}
	return whatsapp.SendText(ctx, client, evt,
		"🚫 *Números ignorados:*\n"+strings.Join(list, "\n"), true)
}

func usageIgnore(ctx context.Context, client *whatsmeow.Client, evt *events.Message) error {
	return whatsapp.SendText(ctx, client, evt,
		"🚫 *Ignorar números*\n\n"+
			"`!ignorar 5511999999999` — ignora um número\n"+
			"`!ignorar remover 5511999999999` — para de ignorar\n"+
			"`!ignorar lista` — lista números ignorados",
		true)
}
