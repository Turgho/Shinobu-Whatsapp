package public

import (
	"context"

	"github.com/Turgho/YuukoWhatsapp/internal/utils"
	"github.com/Turgho/YuukoWhatsapp/pkg/birthday"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// BirthdayCommand gerencia aniversários do grupo.
//
// Em DM (apenas o dono):
//
//	!aniversário salvar @pessoa DD/MM — salva aniversário de outra pessoa
//	!aniversário remover @pessoa      — remove aniversário de outra pessoa
//
// Em grupos:
//
//	!aniversário DD/MM   — salva o próprio aniversário
//	!aniversário lista   — lista aniversários do grupo
//	!aniversário remover — remove o próprio aniversário
func BirthdayCommand(ownerNumber string) func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		// HandleDM já filtra por DM, dono e prefixo !aniversário.
		birthday.HandleDM(ctx, client, evt, ownerNumber)

		if !evt.Info.IsGroup {
			return nil
		}

		if len(args) == 0 {
			return utils.SendText(ctx, client, evt,
				"🎂 *Uso:*\n"+
					"!aniversário DD/MM — salva seu aniversário\n"+
					"!aniversário lista — lista aniversários do grupo\n"+
					"!aniversário remover — remove seu aniversário",
				true,
			)
		}

		return birthday.HandleGroup(ctx, client, evt, args)
	}
}
