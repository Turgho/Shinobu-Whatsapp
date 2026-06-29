package public

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Turgho/Shinobu-Whatsapp/internal/commands"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

func ContagemCommand(
	ctx context.Context,
	client *whatsmeow.Client,
	evt *events.Message,
	args []string,
) error {
	if len(args) < 2 {
		return whatsapp.Reply(ctx, client, evt, msgContagemUsage)
	}

	// Último arg é a data, resto é o nome do evento.
	dateStr := args[len(args)-1]
	eventName := strings.Join(args[:len(args)-1], " ")

	target, err := parseAgendaTime(dateStr)
	if err != nil {
		return whatsapp.Reply(ctx, client, evt, msgContagemUsage)
	}

	now := time.Now()
	if target.Before(now) {
		return whatsapp.Reply(ctx, client, evt, msgContagemPast)
	}

	days := int(time.Until(target).Hours() / 24)

	var line string
	switch {
	case days == 0:
		line = "É hoje! 🎉"
	case days == 1:
		line = "É amanhã! 🎉"
	default:
		line = fmt.Sprintf("Faltam %d dias — %s", days, target.Format("02/01/2006"))
	}

	reply := fmt.Sprintf("⏳ *%s*\n\n%s", eventName, line)
	return whatsapp.Reply(ctx, client, evt, reply)
}

func ContagemHandler() commands.HandlerFunc {
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		return ContagemCommand(ctx, client, evt, args)
	}
}
