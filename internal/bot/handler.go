package bot

import (
	"log"

	"github.com/Turgho/Shinobu-Whatsapp/internal/commands"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

type Handler struct {
	client *whatsmeow.Client
	router *commands.Router
}

func NewHandler(client *whatsmeow.Client, router *commands.Router) *Handler {
	return &Handler{
		client: client,
		router: router,
	}
}

func (h *Handler) EventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		// Goroutine para concorrencia e poder usar comando mais de uma vez
		log.Printf("RawMessage: %+v\n", v.RawMessage)
		go h.router.HandleMessage(v)
	}
}
