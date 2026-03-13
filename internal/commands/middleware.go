package commands

import (
	"fmt"
	"slices"

	"github.com/Turgho/YuukoWhatsapp/internal/utils"
	"go.mau.fi/whatsmeow/types/events"
	"go.uber.org/zap"
)

// PrivateCommandsMiddleware retorna um Middleware que bloqueia comandos privados
// para usuários que não são donos ou admins
func PrivateCommandsMiddleware(owner string, admins []string, privateCommands map[string]bool) Middleware {
	return func(cmd string, evt *events.Message) bool {
		if privateCommands[cmd] {
			return PrivateOnlyMiddleware(evt, owner, admins)
		}
		return true // comandos públicos
	}
}

// Middleware para feedback de comando não encontrado
func CommandNotFoundMiddleware(r *Router) Middleware {
	return func(cmd string, evt *events.Message) bool {
		if _, ok := r.commands[cmd]; !ok {
			// Envia mensagem de erro para o usuário
			if err := utils.Reply(r.client, evt, "❌ Comando não encontrado"); err != nil {
				r.log.Warn("Falha ao notificar usuário sobre comando não encontrado",
					zap.String("command", cmd),
					zap.String("user", evt.Info.Sender.User),
					zap.Error(err),
				)
			}
			return false // bloqueia execução do Router
		}
		return true
	}
}

// Middlare que ignora mensagens do próprio BOT
func IgnoreSelfMiddleware(cmd string, evt *events.Message) bool {
	return !evt.Info.IsFromMe
}

// Middleware que ignora mensagens antigas
func IgnoreOldMessagesMiddleware(cmd string, evt *events.Message) bool {
	upTime := utils.SinceUptime()
	return evt.Info.Timestamp.After(upTime) // compara com o tempo de início do bot
}

// Middleware que bloqueia comandos privados para quem não for dono/admin
func PrivateOnlyMiddleware(evt *events.Message, owner string, admins []string) bool {
	fmt.Println("OWNER_JID:", owner)

	jid := evt.Info.Sender.String()
	if jid == owner {
		return true
	}
	return slices.Contains(admins, jid)
}
