package bot

import (
	"github.com/Turgho/Shinobu-Whatsapp/internal/commands"
	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/gosafe"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
	"go.uber.org/zap"
)

// Handler conecta eventos do whatsmeow ao Router de comandos em gorrotinas seguras.
type Handler struct {
	client *whatsmeow.Client
	router *commands.Router
	log    *zap.Logger
}

// NewHandler cria um handler de eventos WhatsApp com logger para gosafe.Go.
func NewHandler(client *whatsmeow.Client, router *commands.Router, log *zap.Logger) *Handler {
	return &Handler{
		client: client,
		router: router,
		log:    log,
	}
}

// WhatsApp Status (Stories) — notas de implementação:
//
// Mensagens de status chegam via events.Message com:
//   evt.Info.Chat     = status@broadcast (Server: "broadcast")
//   evt.Info.Sender   = JID do contato que postou
//   evt.Info.IsGroup  = false
//
// Respostas a status via Reply() usam evt.Info.Chat como destino,
// o que envia a mensagem de volta ao status broadcast — comportamento
// indesejado. O filtro em HandleMessage descarta esses eventos
// verificando chat.Server == "broadcast".
//
// Broadcast lists (listas de transmissão) também usam Server="broadcast",
// mas com User diferente de "status" (ex: "123456@broadcast").
// Ambos são ignorados pelo mesmo filtro.
//
// Reações a status (Option B) exigem ContextInfo com o stanza ID
// do status original — documentar quando implementado.
func (h *Handler) EventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		// Goroutine para concorrencia e poder usar comando mais de uma vez
		// log.Printf("RawMessage: %+v\n", v.RawMessage)
		gosafe.Go(h.log, func() { h.router.HandleMessage(v) })
	}
}
