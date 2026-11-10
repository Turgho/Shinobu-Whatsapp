package bot

import (
	"context"
	"time"

	"github.com/Turgho/Shinobu-Whatsapp/internal/commands"
	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/gosafe"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"go.uber.org/zap"
)

// Handler conecta eventos do whatsmeow ao Router de comandos em gorrotinas seguras.
type Handler struct {
	client *whatsmeow.Client
	router *commands.Router
	log    *zap.Logger

	// ownerJID (opcional) — destino dos avisos graves (sessão caiu,
	// logou fora, ban temporário).
	ownerJID *types.JID
}

// NewHandler cria um handler de eventos WhatsApp com logger para gosafe.Go.
func NewHandler(client *whatsmeow.Client, router *commands.Router, log *zap.Logger) *Handler {
	return &Handler{
		client: client,
		router: router,
		log:    log,
	}
}

// SetOwnerJID define o JID do dono — usado para avisar quando a sessão
// cai ou é invalidada (sem isso, o bot fica mudo sem ninguém saber).
func (h *Handler) SetOwnerJID(jid *types.JID) {
	h.ownerJID = jid
}

// WhatsApp Status (Stories) — notas de implementação:
//
// Mensagens de status chegam via events.Message com:
//
//	evt.Info.Chat     = status@broadcast (Server: "broadcast")
//	evt.Info.Sender   = JID do contato que postou
//	evt.Info.IsGroup  = false
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

	case *events.Connected:
		h.log.Info("Conectado ao WhatsApp")

	case *events.Disconnected:
		// A biblioteca reconecta sozinha (EnableAutoReconnect é true por
		// padrão); aqui só registramos para o dono ter visibilidade.
		h.log.Warn("Desconectado do WhatsApp — a biblioteca tenta reconectar")

	case *events.LoggedOut:
		h.log.Error("Sessão do WhatsApp invalidada (logged out)",
			zap.Bool("on_connect", v.OnConnect),
			zap.String("reason", v.Reason.String()),
		)
		h.avisarDono("⚠️ A sessão do WhatsApp foi encerrada. Preciso parear de novo pra voltar a te atender.")

	case *events.StreamReplaced:
		h.log.Error("Sessão substituída por outro dispositivo conectado")
		h.avisarDono("⚠️ Entrei em outro dispositivo com a mesma conta. Este bot parou.")

	case *events.ConnectFailure:
		h.log.Warn("Falha ao conectar no WhatsApp",
			zap.String("reason", v.Reason.String()),
			zap.String("message", v.Message),
		)

	case *events.TemporaryBan:
		h.log.Error("Ban temporário no WhatsApp",
			zap.String("code", v.String()),
			zap.Duration("expire", v.Expire),
		)
		h.avisarDono("⚠️ O WhatsApp baneou este bot temporariamente. Aviso quando normalizar.")
	}
}

// avisarDono manda um aviso ao dono (quando o JID é conhecido). Falha de
// envio não é fatal — o log já registrou o evento.
func (h *Handler) avisarDono(msg string) {
	if h.ownerJID == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := whatsapp.SendTextToJID(ctx, h.client, *h.ownerJID, msg, nil); err != nil {
		h.log.Warn("Falha ao avisar o dono sobre o estado da conexão", zap.Error(err))
	}
}
