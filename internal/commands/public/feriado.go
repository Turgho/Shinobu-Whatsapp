package public

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/Turgho/Shinobu-Whatsapp/internal/commands"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/feriado"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

func FeriadoCommand(
	ctx context.Context,
	client *whatsmeow.Client,
	evt *events.Message,
	args []string,
	feriadoClient *feriado.FeriadoClient,
) error {
	n := 5
	if len(args) > 0 {
		if v, err := strconv.Atoi(args[0]); err == nil && v >= 1 && v <= 10 {
			n = v
		}
	}

	list, err := feriadoClient.Upcoming(ctx, n)
	if err != nil {
		return whatsapp.Reply(ctx, client, evt, msgFeriadoFail)
	}

	if len(list) == 0 {
		return whatsapp.Reply(ctx, client, evt, msgFeriadoNone)
	}

	var b strings.Builder
	b.WriteString("🗓 Próximos feriados nacionais\n\n")
	for _, f := range list {
		// "2026-01-01" → "01/01"
		date := f.Date
		if len(date) >= 10 {
			date = date[8:10] + "/" + date[5:7]
		}
		b.WriteString(fmt.Sprintf("📅 %s — %s\n\n", date, f.Name))
	}

	return whatsapp.Reply(ctx, client, evt, strings.TrimSpace(b.String()))
}

func FeriadoHandler(feriadoClient *feriado.FeriadoClient) commands.HandlerFunc {
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		return FeriadoCommand(ctx, client, evt, args, feriadoClient)
	}
}
