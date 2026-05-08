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
func MenuCommand(r *commands.Router) commands.HandlerFunc {
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		metas := r.Commands()

		// Ordena por nome para exibição consistente
		sort.Slice(metas, func(i, j int) bool {
			return metas[i].Name < metas[j].Name
		})

		var sb strings.Builder

		sb.WriteString("🤖 *Menu de comandos*\n")
		sb.WriteString("━━━━━━━━━━━━━━\n")
		sb.WriteString("┃\n")

		for _, meta := range metas {
			if meta.Private {
				continue
			}

			sb.WriteString(fmt.Sprintf("┣ *%s%s*",
				r.Prefix(),
				meta.Name,
			))

			for _, arg := range meta.Args {
				if arg.Required {
					sb.WriteString(fmt.Sprintf(" `<%s>`", arg.Name))
				} else {
					sb.WriteString(fmt.Sprintf(" `[%s]`", arg.Name))
				}
			}

			sb.WriteString("\n")

			if meta.Description != "" {
				sb.WriteString(fmt.Sprintf("┃ %s\n", meta.Description))
			}

			sb.WriteString("┃\n")
		}

		sb.WriteString("┗━━━━━━━━━━━━━━")

		return utils.Reply(ctx, client, evt, sb.String())
	}
}
