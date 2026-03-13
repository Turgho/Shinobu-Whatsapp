package bot

import (
	"github.com/Turgho/YuukoWhatsapp/internal/commands"
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
		h.router.HandleMessage(v)
	}
}
