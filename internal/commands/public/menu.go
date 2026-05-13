package public

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/Turgho/Shinobu-Whatsapp/internal/commands"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

const bannerPath = "assets/images/shinobu_banner.png"

// MenuCommand retorna um handler que envia o banner com o menu como legenda.
func MenuCommand(r *commands.Router) commands.HandlerFunc {
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		metas := r.Commands()

		// Ordena primeiro por tipo e depois por nome
		sort.Slice(metas, func(i, j int) bool {
			if metas[i].Type == metas[j].Type {
				return metas[i].Name < metas[j].Name
			}
			return metas[i].Type < metas[j].Type
		})

		// Agrupa comandos por categoria
		grouped := make(map[commands.CommandType][]commands.CommandMeta)

		for _, meta := range metas {
			if meta.Private {
				continue
			}

			grouped[meta.Type] = append(grouped[meta.Type], meta)
		}

		// Ordem fixa das categorias no menu
		order := []commands.CommandType{
			commands.CommandTypeAI,
			commands.CommandTypeDownload,
			commands.CommandTypeMedia,
			commands.CommandTypeFun,
			commands.CommandTypeUtility,
			commands.CommandTypeGroup,
			commands.CommandTypeAdmin,
			commands.CommandTypeOwner,
			commands.CommandTypeNSFW,
		}

		// Nome bonito das categorias
		labels := map[commands.CommandType]string{
			commands.CommandTypeAI:       "🧠 IA",
			commands.CommandTypeDownload: "📥 Downloads",
			commands.CommandTypeMedia:    "🎵 Mídia",
			commands.CommandTypeFun:      "🎉 Diversão",
			commands.CommandTypeUtility:  "🛠️ Utilidades",
			commands.CommandTypeGroup:    "👥 Grupo",
			commands.CommandTypeAdmin:    "🛡️ Administração",
			commands.CommandTypeOwner:    "👑 Owner",
			commands.CommandTypeNSFW:     "🔞 NSFW",
		}

		var sb strings.Builder
		sb.WriteString("🌸━━━━━ *Shinobu* ━━━━━🌸\n\n")

		for _, category := range order {
			cmds := grouped[category]
			if len(cmds) == 0 {
				continue
			}

			sb.WriteString(fmt.Sprintf("━━━━━ %s ━━━━━\n\n", labels[category]))

			for _, meta := range cmds {
				sb.WriteString(fmt.Sprintf("◈ *%s%s*", r.Prefix(), meta.Name))

				for _, arg := range meta.Args {
					if arg.Required {
						sb.WriteString(fmt.Sprintf(" `<%s>`", arg.Name))
					} else {
						sb.WriteString(fmt.Sprintf(" `[%s]`", arg.Name))
					}
				}

				sb.WriteString("\n")

				if meta.Description != "" {
					sb.WriteString(fmt.Sprintf("  ╰ %s\n", meta.Description))
				}

				sb.WriteString("\n")
			}
		}

		sb.WriteString("─────────────────────\n")
		sb.WriteString("💬 Me chame pelo nome para conversar!")

		return sendBannerWithCaption(ctx, client, evt, sb.String())
	}
}

// sendBannerWithCaption faz upload do banner e envia com o texto do menu como legenda.
// reply=true: cita a mensagem quem usou o comando
func sendBannerWithCaption(ctx context.Context, client *whatsmeow.Client, evt *events.Message, caption string) error {
	data, err := os.ReadFile(bannerPath)
	if err != nil {
		return fmt.Errorf("ler banner: %w", err)
	}

	uploaded, err := client.Upload(ctx, data, whatsmeow.MediaImage)
	if err != nil {
		return fmt.Errorf("upload banner: %w", err)
	}

	return whatsapp.SendImage(ctx, client, evt, &uploaded, data, caption, true)
}
