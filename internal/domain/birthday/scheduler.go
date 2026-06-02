package birthday

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/gosafe"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

const notifyHour = 8

func StartScheduler(client *whatsmeow.Client) {
	gosafe.Go(func() {
		loc, err := time.LoadLocation("America/Sao_Paulo")
		if err != nil {
			fmt.Printf("[birthday] erro timezone: %v\n", err)
			return
		}

		for {
			now := time.Now().In(loc)
			next := nextTrigger(now, notifyHour)

			fmt.Printf("[birthday] próxima verificação: %s\n",
				next.Format("02/01 15:04"))

			<-time.After(time.Until(next))
			checkAndNotify(client)
		}
	})
}

// nextTrigger calcula o próximo horário de verificação de aniversários.
// Se agora já passou do horário alvo, agenda para amanhã no mesmo horário.
// Isso garante que a notificação seja enviada (aproximadamente) uma vez por dia.
func nextTrigger(now time.Time, hour int) time.Time {
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, now.Location())
	if now.After(next) {
		next = next.Add(24 * time.Hour)
	}
	return next
}

// checkAndNotify consulta aniversariantes do dia e envia mensagem para cada grupo.
// Usa o fuso America/Sao_Paulo e os dados da store local.
func checkAndNotify(client *whatsmeow.Client) {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		fmt.Println("[birthday] erro timezone:", err)
		return
	}

	now := time.Now().In(loc)
	birthdays := TodayEntries(now.Day(), int(now.Month()))

	if len(birthdays) == 0 {
		fmt.Println("[birthday] nenhum aniversariante hoje")
		return
	}

	for groupJID, entries := range birthdays {
		jid, err := types.ParseJID(groupJID)
		if err != nil {
			fmt.Printf("[birthday] JID inválido %s: %v\n", groupJID, err)
			continue
		}

		mentions, msg := buildMessage(entries)

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		err = whatsapp.SendTextToJID(ctx, client, jid, msg, mentions)
		cancel()

		if err != nil {
			fmt.Printf("[birthday] erro ao enviar para %s: %v\n", groupJID, err)
		} else {
			fmt.Printf("[birthday] mensagem enviada para %s (%d aniversariante(s))\n",
				groupJID, len(entries))
		}
	}
}

func buildMessage(entries []Entry) (mentions []string, msg string) {
	var sb strings.Builder
	sb.WriteString("🎂 *Parabéns!* 🎉\n\n")

	for _, e := range entries {
		sb.WriteString(fmt.Sprintf("🎈 @%s faz aniversário hoje!\n", e.Name))
		mentions = append(mentions, e.JID)
	}

	sb.WriteString("\n@all venha parabenizar! 🥳")
	mentions = append(mentions, "all@broadcast")

	return mentions, sb.String()
}
