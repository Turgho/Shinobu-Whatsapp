package commands

import (
	"strings"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
	"go.uber.org/zap"
)

type Router struct {
	commands    map[string]Command
	middlewares []Middleware
	prefix      string
	client      *whatsmeow.Client
	log         *zap.Logger
}

func NewRouter(prefix string, client *whatsmeow.Client, log *zap.Logger) *Router {
	return &Router{
		commands: make(map[string]Command),
		prefix:   prefix,
		client:   client,
		log:      log,
	}
}

func (r *Router) RegisterCommand(name string, cmd Command) {
	r.commands[name] = cmd

	r.log.Info(
		"Comando registrado",
		zap.String("command", name),
	)
}

func (r *Router) Use(m Middleware) {
	r.middlewares = append(r.middlewares, m)
}

func (r *Router) HandleMessage(evt *events.Message) {
	r.log.Debug(
		"Mensagem recebida",
		zap.String("sender", evt.Info.Sender.User),
		zap.String("chat", evt.Info.Chat.String()),
	)

	msg := getTextMessage(evt)

	if msg == "" || !strings.HasPrefix(msg, r.prefix) {
		return
	}

	parts := strings.Fields(strings.TrimPrefix(msg, r.prefix))
	if len(parts) == 0 {
		return
	}

	cmdName := parts[0]
	args := parts[1:]

	// roda middlewares
	for _, m := range r.middlewares {
		if !m(cmdName, evt) {
			return
		}
	}

	cmd, ok := r.commands[cmdName]
	if !ok {
		r.log.Warn("Comando não encontrado",
			zap.String("command", cmdName),
		)
		return
	}

	start := time.Now()

	r.log.Info(
		"Executando comando",
		zap.String("command", cmdName),
		zap.String("user", evt.Info.Sender.User),
		zap.Strings("args", args),
	)

	err := cmd(r.client, evt, args)

	duration := time.Since(start)

	if err != nil {
		r.log.Error("Erro no comando",
			zap.String("command", cmdName),
			zap.Error(err),
		)
		return
	}

	r.log.Info("Comando executado",
		zap.String("command", cmdName),
		zap.Duration("duration", duration),
	)
}

func getTextMessage(evt *events.Message) string {
	if evt.Message == nil {
		return ""
	}

	// Mensagem simples de texto
	msg := evt.Message.GetConversation()

	// Mensagem extendida (como reply)
	if msg == "" && evt.Message.GetExtendedTextMessage() != nil {
		msg = evt.Message.GetExtendedTextMessage().GetText()
	}

	// Legenda de imagem
	if msg == "" && evt.Message.GetImageMessage() != nil {
		msg = evt.Message.GetImageMessage().GetCaption()
	}

	// Legenda de vídeo
	if msg == "" && evt.Message.GetVideoMessage() != nil {
		msg = evt.Message.GetVideoMessage().GetCaption()
	}

	// Legenda de documento
	if msg == "" && evt.Message.GetDocumentMessage() != nil {
		msg = evt.Message.GetDocumentMessage().GetCaption()
	}

	return strings.TrimSpace(strings.ToLower(msg))
}
