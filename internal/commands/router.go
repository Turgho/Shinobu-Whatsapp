package commands

import (
	"context"
	"strings"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
	"go.uber.org/zap"
)

type Router struct {
	commands    map[string]command
	middlewares []Middleware
	prefix      string
	client      *whatsmeow.Client
	log         *zap.Logger
}

func NewRouter(prefix string, client *whatsmeow.Client, log *zap.Logger) *Router {
	return &Router{
		commands: make(map[string]command),
		prefix:   prefix,
		client:   client,
		log:      log,
	}
}

// RegisterCommand registra um comando com seus metadados e handler.
func (r *Router) RegisterCommand(meta CommandMeta, handler HandlerFunc) {
	r.commands[meta.Name] = command{Meta: meta, Handler: handler}
	r.log.Info("Comando registrado",
		zap.String("command", meta.Name),
		zap.Bool("private", meta.Private),
	)
}

// Use adiciona um middleware ao pipeline.
func (r *Router) Use(m Middleware) {
	r.middlewares = append(r.middlewares, m)
}

// Commands retorna os metadados de todos os comandos registrados.
func (r *Router) Commands() []CommandMeta {
	metas := make([]CommandMeta, 0, len(r.commands))
	for _, cmd := range r.commands {
		metas = append(metas, cmd.Meta)
	}
	return metas
}

// HasCommand verifica se um comando está registrado.
func (r *Router) HasCommand(name string) bool {
	_, ok := r.commands[name]
	return ok
}

// IsPrivate verifica se um comando registrado está marcado como privado.
func (r *Router) IsPrivate(name string) bool {
	cmd, ok := r.commands[name]
	return ok && cmd.Meta.Private
}

// Prefix retorna o prefixo configurado no router.
func (r *Router) Prefix() string {
	return r.prefix
}

// HandleMessage é o ponto de entrada para eventos de mensagem do WhatsApp.
func (r *Router) HandleMessage(evt *events.Message) {
	msg := getTextMessage(evt)

	// Descarta silenciosamente mensagens sem prefixo — a grande maioria
	if msg == "" || !strings.HasPrefix(msg, r.prefix) {
		return
	}

	parts := strings.Fields(strings.TrimPrefix(msg, r.prefix))
	if len(parts) == 0 {
		return
	}

	cmdName := parts[0]
	args := parts[1:]

	// Middlewares rodam antes do log — mensagens antigas e do próprio bot
	// são descartadas aqui sem aparecer no terminal
	for _, m := range r.middlewares {
		if !m(cmdName, evt) {
			return
		}
	}

	// Só loga mensagens que passaram em todos os filtros
	r.log.Info("Comando recebido",
		zap.String("command", cmdName),
		zap.String("user", evt.Info.Sender.User),
		zap.Strings("args", args),
	)

	cmd, ok := r.commands[cmdName]
	if !ok {
		// Não deveria chegar aqui — CommandNotFoundMiddleware já tratou
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	start := time.Now()

	if err := cmd.Handler(ctx, r.client, evt, args); err != nil {
		r.log.Error("Erro no comando",
			zap.String("command", cmdName),
			zap.String("user", evt.Info.Sender.User),
			zap.Error(err),
		)
		return
	}

	r.log.Info("Comando executado",
		zap.String("command", cmdName),
		zap.Duration("duration", time.Since(start)),
	)
}

// getTextMessage extrai o texto de uma mensagem, suportando vários tipos.
func getTextMessage(evt *events.Message) string {
	if evt.Message == nil {
		return ""
	}

	msg := evt.Message.GetConversation()

	if msg == "" && evt.Message.GetExtendedTextMessage() != nil {
		msg = evt.Message.GetExtendedTextMessage().GetText()
	}
	if msg == "" && evt.Message.GetImageMessage() != nil {
		msg = evt.Message.GetImageMessage().GetCaption()
	}
	if msg == "" && evt.Message.GetVideoMessage() != nil {
		msg = evt.Message.GetVideoMessage().GetCaption()
	}
	if msg == "" && evt.Message.GetDocumentMessage() != nil {
		msg = evt.Message.GetDocumentMessage().GetCaption()
	}

	return strings.TrimSpace(strings.ToLower(msg))
}
