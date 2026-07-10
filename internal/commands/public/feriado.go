package public

import (
	"context"
	"fmt"
	"strings"

	"github.com/Turgho/Shinobu-Whatsapp/internal/commands"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/feriado"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// UFs válidas do Brasil.
var validUFs = map[string]bool{
	"AC": true, "AL": true, "AP": true, "AM": true, "BA": true,
	"CE": true, "DF": true, "ES": true, "GO": true, "MA": true,
	"MT": true, "MS": true, "MG": true, "PA": true, "PB": true,
	"PR": true, "PE": true, "PI": true, "RJ": true, "RN": true,
	"RS": true, "RO": true, "RR": true, "SC": true, "SP": true,
	"SE": true, "TO": true,
}

func FeriadoCommand(
	ctx context.Context,
	client *whatsmeow.Client,
	evt *events.Message,
	args []string,
	feriadoClient *feriado.FeriadosClient,
) error {
	uf := ""
	if len(args) > 0 {
		candidate := strings.ToUpper(strings.TrimSpace(args[0]))
		if len(candidate) == 2 && validUFs[candidate] {
			uf = candidate
		}
	}

	list, err := feriadoClient.Upcoming(ctx, 5, uf)
	if err != nil {
		return whatsapp.Reply(ctx, client, evt, msgFeriadoFail)
	}

	if len(list) == 0 {
		return whatsapp.Reply(ctx, client, evt, msgFeriadoNone)
	}

	var b strings.Builder
	if uf != "" {
		fmt.Fprintf(&b, "🗓 Próximos feriados — %s\n\n", uf)
	} else {
		b.WriteString("🗓 Próximos feriados nacionais\n\n")
	}

	for _, f := range list {
		fmt.Fprintf(&b, "📅 %s — %s\n\n", f.Data, f.Nome)
	}

	return whatsapp.Reply(ctx, client, evt, strings.TrimSpace(b.String()))
}

func FeriadoHandler(feriadoClient *feriado.FeriadosClient) commands.HandlerFunc {
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		return FeriadoCommand(ctx, client, evt, args, feriadoClient)
	}
}
