package public

import (
	"context"

	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/birthday"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// BirthdayCommand gerencia aniversários — apenas em grupos.
//
// Qualquer membro:
//
//	!aniversario DD/MM            — salva o próprio aniversário
//	!aniversario lista            — lista aniversários do grupo
//	!aniversario remover          — remove o próprio aniversário
//
// Dono/admins:
//
//	!aniversario salvar @pessoa DD/MM — salva de outra pessoa
//	!aniversario remover @pessoa      — remove de outra pessoa
func BirthdayCommand(ownerNumber string, admins []string) func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		if !evt.Info.IsGroup {
			return whatsapp.SendText(ctx, client, evt,
				"❌ Este comando só funciona em grupos.", true)
		}

		return birthday.HandleGroup(ctx, client, evt, args, ownerNumber, admins)
	}
}
