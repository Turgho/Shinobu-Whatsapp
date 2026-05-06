package public

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/Turgho/YuukoWhatsapp/internal/commands"
	"github.com/Turgho/YuukoWhatsapp/internal/utils"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// MenuCommand retorna um handler que lista todos os comandos públicos registrados.
// É gerado automaticamente a partir dos metadados — adicionar um novo comando
// com RegisterCommand já o faz aparecer aqui sem nenhuma alteração manual.
func MenuCommand(r *commands.Router) commands.HandlerFunc {
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		metas := r.Commands()

		// Ordena por nome para exibição consistente
		sort.Slice(metas, func(i, j int) bool {
			return metas[i].Name < metas[j].Name
		})

		var sb strings.Builder
		sb.WriteString("🤖 *Comandos disponíveis:*\n\n")

		for _, meta := range metas {
			if meta.Private {
				continue // oculta comandos de admin no menu público
			}

			// Linha do comando com seus argumentos
			sb.WriteString(fmt.Sprintf("*%s%s*", r.Prefix(), meta.Name))
			for _, arg := range meta.Args {
				if arg.Required {
					sb.WriteString(fmt.Sprintf(" `<%s>`", arg.Name))
				} else {
					sb.WriteString(fmt.Sprintf(" `[%s]`", arg.Name))
				}
			}
			sb.WriteString("\n")

			if meta.Description != "" {
				sb.WriteString(fmt.Sprintf("  _%s_\n", meta.Description))
			}
			sb.WriteString("\n")
		}

		return utils.Reply(ctx, client, evt, strings.TrimRight(sb.String(), "\n"))
	}
}
