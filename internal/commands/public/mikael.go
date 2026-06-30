package public

import (
	"context"
	"fmt"

	"github.com/Turgho/Shinobu-Whatsapp/internal/commands"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/mikael"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
	"go.uber.org/zap"
)

func MikaelHandler(mStore *mikael.Store, log *zap.Logger) commands.HandlerFunc {
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		chat := evt.Info.Chat.String()
		count, err := mStore.CountWord(ctx, chat, "pix")
		if err != nil {
			log.Error("Erro ao contar palavras do Mikael",
				zap.String("chat", chat),
				zap.Error(err),
			)
			return whatsapp.Reply(ctx, client, evt, "❌ Não consegui contar as mensagens.")
		}

		return whatsapp.Reply(ctx, client, evt, fmt.Sprintf("📊 Mikael escreveu 'pix' %d vezes neste grupo.", count))
	}
}
