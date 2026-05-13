package birthday

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

const notifyHour = 8 // 8h da manhã

// StartScheduler inicia o loop em background que verifica aniversários todo dia às 8h.
// Deve ser chamado uma vez no main após o cliente conectar.
func StartScheduler(client *whatsmeow.Client) {
	go func() {
		for {
			now := time.Now()
			next := nextTrigger(now, notifyHour)
			fmt.Printf("[birthday] próxima verificação: %s\n", next.Format("02/01 15:04"))

			<-time.After(time.Until(next))
			checkAndNotify(client)
		}
	}()
}

// nextTrigger calcula o próximo horário de disparo.
// Se já passou das 8h hoje, agenda para amanhã.
func nextTrigger(now time.Time, hour int) time.Time {
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, now.Location())
	if now.After(next) {
		next = next.Add(24 * time.Hour)
	}
	return next
}

// checkAndNotify verifica aniversariantes do dia e envia mensagem em cada grupo.
func checkAndNotify(client *whatsmeow.Client) {
	now := time.Now()
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

// buildMessage monta a mensagem de aniversário com @all e menções dos aniversariantes.
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
