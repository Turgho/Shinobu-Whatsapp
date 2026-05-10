package public

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/Turgho/YuukoWhatsapp/internal/commands"
	"github.com/Turgho/YuukoWhatsapp/internal/utils"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

const bannerPath = "assets/images/shinobu_banner.png"

// MenuCommand retorna um handler que envia o banner seguido da lista de comandos públicos.
func MenuCommand(r *commands.Router) commands.HandlerFunc {
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		// Envia o banner se o arquivo existir; falha silenciosa para não bloquear o menu.
		if err := sendBanner(ctx, client, evt); err != nil {
			fmt.Println("[menu] banner não enviado:", err)
		}

		metas := r.Commands()
		sort.Slice(metas, func(i, j int) bool {
			return metas[i].Name < metas[j].Name
		})

		var sb strings.Builder
		sb.WriteString("╔══════════════════╗\n")
		sb.WriteString("║   🌸  *Shinobu*  🌸   ║\n")
		sb.WriteString("╚══════════════════╝\n\n")

		for _, meta := range metas {
			if meta.Private {
				continue
			}

			// Nome e argumentos do comando.
			sb.WriteString(fmt.Sprintf("◈ *%s%s*", r.Prefix(), meta.Name))
			for _, arg := range meta.Args {
				if arg.Required {
					sb.WriteString(fmt.Sprintf(" `<%s>`", arg.Name))
				} else {
					sb.WriteString(fmt.Sprintf(" `[%s]`", arg.Name))
				}
			}
			sb.WriteString("\n")

			// Descrição indentada abaixo do comando.
			if meta.Description != "" {
				sb.WriteString(fmt.Sprintf("  ╰ %s\n", meta.Description))
			}

			sb.WriteString("\n")
		}

		sb.WriteString("─────────────────────\n")
		sb.WriteString("💬 Me chame pelo nome para conversar!")

		return utils.SendText(ctx, client, evt, sb.String(), true)
	}
}

// sendBanner faz upload e envia o banner como imagem sem citação.
func sendBanner(ctx context.Context, client *whatsmeow.Client, evt *events.Message) error {
	data, err := os.ReadFile(bannerPath)
	if err != nil {
		return fmt.Errorf("ler banner: %w", err)
	}

	uploaded, err := client.Upload(ctx, data, whatsmeow.MediaImage)
	if err != nil {
		return fmt.Errorf("upload banner: %w", err)
	}

	// reply=false: banner vai solto, sem ficar preso na citação da mensagem original.
	return utils.SendImage(ctx, client, evt, &uploaded, "", false)
}
