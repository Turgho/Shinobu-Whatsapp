package admin

import (
	"context"
	"fmt"
	"strings"

	"github.com/Turgho/Shinobu-Whatsapp/internal/commands"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
	"go.uber.org/zap"
)

// ComandosCommand lista os comandos e permite desabilitar/habilitar um
// comando específico — sem derrubar o bot inteiro.
//
//	!comandos                      → lista (marcando os desativados)
//	!comandos desabilitar <nome>   → desativa
//	!comandos habilitar <nome>     → reativa
func ComandosCommand(r *commands.Router, logger *zap.Logger) commands.HandlerFunc {
	l := logger.Named("COMANDOS")

	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		if len(args) == 0 {
			return listCommands(ctx, client, evt, r)
		}

		acao := strings.ToLower(args[0])
		if len(args) < 2 {
			return whatsapp.Reply(ctx, client, evt,
				"Use: `!comandos desabilitar <nome>` ou `!comandos habilitar <nome>`")
		}

		nome := strings.ToLower(strings.TrimPrefix(args[1], r.Prefix()))
		if r.Handler(nome) == nil && !r.HasCommand(nome) {
			return whatsapp.Reply(ctx, client, evt,
				fmt.Sprintf("Não conheço o comando *%s*. Use `!comandos` pra ver a lista.", nome))
		}

		switch acao {
		case "desabilitar", "desativa", "off":
			if err := r.SetCommandDisabled(nome, true); err != nil {
				l.Error("falha ao desabilitar comando", zap.String("command", nome), zap.Error(err))
				return whatsapp.Reply(ctx, client, evt, "❌ Não consegui salvar essa mudança.")
			}
			l.Info("comando desabilitado", zap.String("command", nome))
			return whatsapp.Reply(ctx, client, evt,
				fmt.Sprintf("🛠️ Comando *%s* desativado. Os outros continuam funcionando.", nome))

		case "habilitar", "ativa", "on":
			if err := r.SetCommandDisabled(nome, false); err != nil {
				l.Error("falha ao habilitar comando", zap.String("command", nome), zap.Error(err))
				return whatsapp.Reply(ctx, client, evt, "❌ Não consegui salvar essa mudança.")
			}
			l.Info("comando habilitado", zap.String("command", nome))
			return whatsapp.Reply(ctx, client, evt,
				fmt.Sprintf("✅ Comando *%s* ativado de novo.", nome))

		default:
			return whatsapp.Reply(ctx, client, evt,
				"Use: `!comandos desabilitar <nome>` ou `!comandos habilitar <nome>`")
		}
	}
}

// listCommands mostra os comandos e o estado de cada um.
func listCommands(ctx context.Context, client *whatsmeow.Client, evt *events.Message, r *commands.Router) error {
	metas := r.CommandCatalog()

	var b strings.Builder
	b.WriteString("🛠️ *Comandos*\n\n")

	desativados := 0
	for _, meta := range metas {
		estado := "🟢"
		if r.IsCommandDisabled(meta.Name) {
			estado = "🔴"
			desativados++
		}
		fmt.Fprintf(&b, "%s *%s%s*", estado, r.Prefix(), meta.Name)
		if meta.Description != "" {
			fmt.Fprintf(&b, " — %s", meta.Description)
		}
		b.WriteString("\n")
	}

	fmt.Fprintf(&b, "\n%d comando(s) · %d desativado(s)\n", len(metas), desativados)
	fmt.Fprintf(&b, "`%scomandos desabilitar <nome>` · `%scomandos habilitar <nome>`", r.Prefix(), r.Prefix())

	return whatsapp.Reply(ctx, client, evt, b.String())
}
