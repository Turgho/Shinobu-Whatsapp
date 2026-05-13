package commands

import (
	"context"
	"slices"

	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/uptime"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"go.mau.fi/whatsmeow/types/events"
	"go.uber.org/zap"
)

// sendMiddlewareNotice envia texto ao usuário e registra falha de envio sem alterar o fluxo do middleware.
func sendMiddlewareNotice(
	ctx context.Context,
	r *Router,
	evt *events.Message,
	cmd, logUser, msg, warn string,
) {
	if err := whatsapp.SendText(ctx, r.client, evt, msg, true); err != nil {
		r.log.Warn(warn,
			zap.String("command", cmd),
			zap.String("user", logUser),
			zap.Error(err),
		)
	}
}

// IgnoreSelfMiddleware ignora mensagens enviadas pelo próprio bot.
func IgnoreSelfMiddleware(cmd string, evt *events.Message) bool {
	return !evt.Info.IsFromMe
}

// IgnoreOldMessagesMiddleware ignora mensagens anteriores ao início do bot.
// Evita reagir a mensagens antigas ao reiniciar.
func IgnoreOldMessagesMiddleware(cmd string, evt *events.Message) bool {
	return evt.Info.Timestamp.After(uptime.ProcessStartTime())
}

// CommandNotFoundMiddleware notifica o usuário quando o comando não existe.
// Deve vir após IgnoreOldMessages para não responder a mensagens antigas.
func CommandNotFoundMiddleware(r *Router) Middleware {
	return func(cmd string, evt *events.Message) bool {
		if r.HasCommand(cmd) {
			return true
		}
		ctx := context.Background()
		msg := "❌ Comando não encontrado. Use *" + r.Prefix() + "menu* para ver os disponíveis."
		sendMiddlewareNotice(ctx, r, evt, cmd, evt.Info.Sender.User, msg, "Falha ao notificar comando não encontrado")
		return false
	}
}

// PrivateCommandsMiddleware bloqueia comandos marcados como Private=true para
// usuários sem permissão. Usa os metadados do Router — sem mapa manual externo.
func PrivateCommandsMiddleware(r *Router, owner string, admins []string) Middleware {
	return func(cmd string, evt *events.Message) bool {
		if !r.IsPrivate(cmd) {
			return true
		}
		jid := evt.Info.Sender.String()
		if jid == owner || slices.Contains(admins, jid) {
			return true
		}
		ctx := context.Background()
		msg := "🔒 Você não tem permissão para usar este comando."
		sendMiddlewareNotice(ctx, r, evt, cmd, jid, msg, "Falha ao notificar acesso negado")
		return false
	}
}
