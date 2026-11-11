package public

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/Turgho/Shinobu-Whatsapp/internal/commands"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/mikael"
	"github.com/Turgho/Shinobu-Whatsapp/internal/identity"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
	"go.uber.org/zap"
)

// MikaelHandler mostra a contagem do Mikael no grupo.
//
// Uso (qualquer um):
//
//	!mikael             → "📊 Mikael escreveu 'pix' N vezes neste grupo."
//
// Uso (apenas o dono):
//
//	!mikael somar <n>    → soma n ao contador (n pode ser negativo)
//	!mikael ajustar <n>  → define o contador para n
//
// O ajuste pelo dono existe porque a contagem é histórica: a tabela de
// mensagens é limpa periodicamente, então recalcular por ela perderia o
// histórico. Ajustar o número é a forma de não perder.
func MikaelHandler(mStore *mikael.Store, owner identity.Owner, log *zap.Logger) commands.HandlerFunc {
	l := log.Named("MIKAEL")

	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		chat := evt.Info.Chat.String()
		sender := evt.Info.Sender.ToNonAD()

		// Ajuste de contagem — só o dono.
		if len(args) >= 2 && owner.Matches(sender.User, sender.String()) {
			acao := strings.ToLower(args[0])
			valor, err := strconv.Atoi(strings.TrimSpace(args[1]))
			if err != nil {
				return whatsapp.Reply(ctx, client, evt, "❌ O valor tem que ser um número inteiro.")
			}

			var total int
			switch acao {
			case "somar", "soma", "+":
				total, err = mStore.AddCount(ctx, chat, "pix", valor)
			case "ajustar", "define", "=":
				total, err = mStore.SetCount(ctx, chat, "pix", valor)
			default:
				return whatsapp.Reply(ctx, client, evt,
					"Use: `!mikael somar <n>` ou `!mikael ajustar <n>`.")
			}
			if err != nil {
				l.Error("erro ao ajustar contagem", zap.Error(err))
				return whatsapp.Reply(ctx, client, evt, "❌ Não consegui salvar a contagem.")
			}

			l.Info("contagem ajustada",
				zap.String("chat", chat),
				zap.String("acao", acao),
				zap.Int("valor", valor),
				zap.Int("total", total),
			)
			return whatsapp.Reply(ctx, client, evt,
				fmt.Sprintf("✅ Contagem ajustada: agora são *%d* 'pix' neste grupo.", total))
		}

		count, err := mStore.CountWord(ctx, chat, "pix")
		if err != nil {
			l.Error("erro ao contar mensagens do Mikael", zap.String("chat", chat), zap.Error(err))
			return whatsapp.Reply(ctx, client, evt, "❌ Não consegui contar as mensagens.")
		}

		return whatsapp.Reply(ctx, client, evt,
			fmt.Sprintf("📊 Mikael escreveu 'pix' %d vezes neste grupo.", count))
	}
}
