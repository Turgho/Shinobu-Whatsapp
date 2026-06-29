package public

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Turgho/Shinobu-Whatsapp/internal/commands"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/history"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/ia"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/sticker"
	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/configs"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// Duplicate cache: se o mesmo prompt chegar no mesmo chat em menos de 30s,
// reusa a resposta anterior. Comum no WhatsApp por dupla notificação.
var dupCache struct {
	mu sync.Mutex
	m  map[string]dupEntry
}

type dupEntry struct {
	answer string
	at     time.Time
}

const dupCacheTTL = 30 * time.Second

func dupKey(chat, prompt string) string {
	h := sha256.Sum256([]byte(chat + "\x00" + prompt))
	return fmt.Sprintf("%x", h[:8])
}

func ShinobuCommand(store *history.Store, cfg *configs.Config, stickerStore *sticker.Store) commands.HandlerFunc {
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		chat := evt.Info.Chat.String()
		sender := evt.Info.Sender.User
		isOwner := sender == cfg.Owner.Number
		prompt := strings.Join(args, " ")

		prompt = strings.TrimSpace(
			strings.NewReplacer("shinobu", "", "Shinobu", "").Replace(prompt),
		)

		if len(args) == 0 || prompt == "" {
			return whatsapp.Reply(ctx, client, evt, "Hmph... fala logo, tolo.")
		}

		mentions := []string{}
		if ext := evt.Message.GetExtendedTextMessage(); ext != nil {
			mentions = ext.GetContextInfo().GetMentionedJID()
		}

		_ = store.Save(ctx, chat, sender, prompt)

		// Extrai fatos sobre o usuário e salva na memória persistente.
		if store.UserMemory != nil {
			for key, value := range history.ExtractFactsFromPrompt(prompt) {
				_ = store.UserMemory.SaveFact(ctx, chat, sender, key, value)
			}
		}

		// Cache de duplicatas: mesmo prompt no mesmo chat em <30s.
		if cached := getCachedAnswer(chat, prompt); cached != "" {
			return sendShinobuReply(ctx, client, evt, cached, false, mentions, stickerStore)
		}

		iaCfg := &ia.Config{
			GroqURL:    cfg.Groq.URL,
			GroqKey:    cfg.Groq.APIKey,
			TavilyKey:  cfg.Tavily.APIKey,
			HTTPClient: &http.Client{Timeout: 30 * time.Second},
		}

		answer, usedSearch, err := ia.AskIA(ctx, iaCfg, chat, prompt, isOwner, sender, store)
		if err != nil {
			return whatsapp.Reply(ctx, client, evt, msgShinobuFail)
		}

		_ = store.Save(ctx, chat, history.AssistantSenderName, answer)
		setCachedAnswer(chat, prompt, answer)

		return sendShinobuReply(ctx, client, evt, answer, usedSearch, mentions, stickerStore)
	}
}

func getCachedAnswer(chat, prompt string) string {
	dupCache.mu.Lock()
	defer dupCache.mu.Unlock()

	e, ok := dupCache.m[dupKey(chat, prompt)]
	if !ok || time.Since(e.at) > dupCacheTTL {
		return ""
	}
	return e.answer
}

func setCachedAnswer(chat, prompt, answer string) {
	dupCache.mu.Lock()
	defer dupCache.mu.Unlock()

	if dupCache.m == nil {
		dupCache.m = make(map[string]dupEntry)
	}
	dupCache.m[dupKey(chat, prompt)] = dupEntry{
		answer: answer,
		at:     time.Now(),
	}
}

// sendShinobuReply envia a resposta e, se houve busca web, anexa sticker smart_ruby.
func sendShinobuReply(ctx context.Context, client *whatsmeow.Client, evt *events.Message, answer string, usedSearch bool, mentions []string, stickerStore *sticker.Store) error {
	if len(mentions) > 0 {
		if err := whatsapp.ReplyWithMentions(ctx, client, evt, answer, mentions); err != nil {
			return err
		}
	} else {
		if err := whatsapp.Reply(ctx, client, evt, answer); err != nil {
			return err
		}
	}
	if usedSearch {
		_ = sticker.Send(ctx, client, evt, "smart_ruby", false, stickerStore)
	}
	return nil
}
