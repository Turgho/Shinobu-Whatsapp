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
	loc *time.Location,
) error {
	if len(args) < 2 {
		return whatsapp.Reply(ctx, client, evt, msgContagemUsage)
	}

	// Último arg é a data, resto é o nome do evento.
	dateStr := args[len(args)-1]
	eventName := capitalizeWords(strings.Join(args[:len(args)-1], " "))

	target, err := parseAgendaTime(dateStr, loc)
	if err != nil {
		return whatsapp.Reply(ctx, client, evt, msgContagemUsage)
	}

	now := time.Now().In(loc)
	if target.Before(now) {
		return whatsapp.Reply(ctx, client, evt, msgContagemPast)
	}

	days := int(time.Until(target).Hours() / 24)

	var line string
	switch days {
	case 0:
		line = "É hoje! 🎉"
	case 1:
		line = "É amanhã! 🎉"
	default:
		line = fmt.Sprintf("Faltam %d dias — %s", days, target.Format("02/01/2006"))
	}

	reply := fmt.Sprintf("⏳ *%s*\n\n%s", eventName, line)
	return whatsapp.Reply(ctx, client, evt, reply)
}

func ContagemHandler(loc *time.Location) commands.HandlerFunc {
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		return ContagemCommand(ctx, client, evt, args, loc)
	}
}

// capitalizeWords capitaliza a primeira letra de cada palavra.
func capitalizeWords(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		if len(w) > 0 {
			runes := []rune(w)
			if runes[0] >= 'a' && runes[0] <= 'z' {
				runes[0] = runes[0] - 32
			}
			words[i] = string(runes)
		}
	}
	return strings.Join(words, " ")
}
