package public

import (
	"context"
	"fmt"
	"strings"

	"github.com/Turgho/Shinobu-Whatsapp/internal/commands"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/history"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/ia"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

func TraduzCommand(
	ctx context.Context,
	client *whatsmeow.Client,
	evt *events.Message,
	args []string,
	aiCfg *ia.Config,
) error {
	text, idioma := extractTextAndLanguage(args, evt)

	if text == "" {
		return whatsapp.Reply(ctx, client, evt, msgTraduzNoText)
	}

	system := fmt.Sprintf("Você é um tradutor preciso. Traduza o texto fornecido para %s. Retorne APENAS a tradução, sem explicações.", idioma)

	msgs := []history.IAMessage{
		{Role: "system", Content: system},
		{Role: "user", Content: text},
	}

	answer, err := ia.QuickChat(ctx, aiCfg.HTTPClient, aiCfg.GroqURL, aiCfg.GroqKey, ia.ModelFastClass, msgs, 0, 300)
	if err != nil {
		return whatsapp.Reply(ctx, client, evt, msgTraduzFail)
	}

	reply := fmt.Sprintf("🌐 Tradução para %s\n%s", idioma, strings.TrimSpace(answer))
	return whatsapp.Reply(ctx, client, evt, reply)
}

// extractTextAndLanguage extrai o texto a traduzir e o idioma alvo dos args ou mensagem citada.
func extractTextAndLanguage(args []string, evt *events.Message) (text, idioma string) {
	idioma = "português (pt-BR)"

	// Tenta extrair idioma dos args.
	if len(args) >= 2 {
		lastTwo := strings.ToLower(strings.Join(args[len(args)-2:], " "))
		switch {
		case strings.Contains(lastTwo, "para inglês"), strings.Contains(lastTwo, "para ingles"):
			idioma = "inglês"
			args = args[:len(args)-2]
		case strings.Contains(lastTwo, "para espanhol"):
			idioma = "espanhol"
			args = args[:len(args)-2]
		}
	}

	// Texto dos args.
	if len(args) > 0 {
		text = strings.Join(args, " ")
	}

	// Se for reply, usa o texto citado.
	if text == "" {
		if ext := evt.Message.GetExtendedTextMessage(); ext != nil {
			if quoted := ext.GetContextInfo().GetQuotedMessage(); quoted != nil {
				if c := quoted.GetConversation(); c != "" {
					text = c
				}
				if c := quoted.GetExtendedTextMessage().GetText(); c != "" {
					text = c
				}
			}
		}
	}

	return text, idioma
}

func TraduzHandler(aiCfg *ia.Config) commands.HandlerFunc {
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		return TraduzCommand(ctx, client, evt, args, aiCfg)
	}
}
