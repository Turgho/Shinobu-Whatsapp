package commands

import (
	"context"
	"slices"

	"github.com/Turgho/YuukoWhatsapp/internal/utils"
	"go.mau.fi/whatsmeow/types/events"
	"go.uber.org/zap"
)

// IgnoreSelfMiddleware ignora mensagens enviadas pelo próprio bot.
func IgnoreSelfMiddleware(cmd string, evt *events.Message) bool {
	return !evt.Info.IsFromMe
}

// IgnoreOldMessagesMiddleware ignora mensagens anteriores ao início do bot.
// Evita reagir a mensagens antigas ao reiniciar.
func IgnoreOldMessagesMiddleware(cmd string, evt *events.Message) bool {
	return evt.Info.Timestamp.After(utils.SinceUptime())
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
		if err := utils.Reply(ctx, r.client, evt, msg); err != nil {
			r.log.Warn("Falha ao notificar comando não encontrado",
				zap.String("command", cmd),
				zap.String("user", evt.Info.Sender.User),
				zap.Error(err),
			)
		}
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
		if err := utils.Reply(ctx, r.client, evt, "🔒 Você não tem permissão para usar este comando."); err != nil {
			r.log.Warn("Falha ao notificar acesso negado",
				zap.String("command", cmd),
				zap.String("user", jid),
				zap.Error(err),
			)
		}
		return false
	}
}
