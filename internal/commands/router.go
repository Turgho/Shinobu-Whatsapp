package commands

import (
	"context"
	"strings"
	"time"

	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/history"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
	"go.uber.org/zap"
)

const (
	handlerTimeoutMention = 30 * time.Second
	handlerTimeoutCommand = 60 * time.Second
)

// Router associa prefixo, handlers, middlewares e o store compartilhado (ex.: histórico da IA).
type Router struct {
	commands    map[string]command
	middlewares []Middleware
	prefix      string
	client      *whatsmeow.Client
	log         *zap.Logger
	store       *history.Store
}

// NewRouter cria um router vazio; use RegisterCommand e Use antes de HandleMessage.
func NewRouter(prefix string, client *whatsmeow.Client, log *zap.Logger, store *history.Store) *Router {
	return &Router{
		commands: make(map[string]command),
		prefix:   prefix,
		client:   client,
		log:      log,
		store:    store,
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

// Use adiciona um middleware ao pipeline (ordem de registro = ordem de execução).
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

// HandleMessage despacha texto de mensagem: atalho por menção "shinobu" ou comando !nome.
func (r *Router) HandleMessage(evt *events.Message) {
	msg := whatsapp.VisibleTextFromEvent(evt)
	if msg == "" {
		return
	}

	if isMentioned(msg) {
		r.handleShinobuMention(evt, msg)
		return
	}

	if !strings.HasPrefix(msg, r.prefix) {
		return
	}

	parts := strings.Fields(strings.TrimPrefix(msg, r.prefix))
	if len(parts) == 0 {
		return
	}

	cmdName := strings.ToLower(parts[0])
	r.handlePrefixedCommand(evt, cmdName, parts[1:])
}

// runMiddlewares executa a cadeia; o nome do comando alimenta middlewares que dependem dele (ex.: privado).
func (r *Router) runMiddlewares(cmdName string, evt *events.Message) bool {
	for _, m := range r.middlewares {
		if !m(cmdName, evt) {
			return false
		}
	}
	return true
}

func (r *Router) handleShinobuMention(evt *events.Message, msg string) {
	if !r.runMiddlewares("shinobu", evt) {
		return
	}

	cmd, ok := r.commands["shinobu"]
	if !ok {
		return
	}

	r.log.Info("Shinobu mencionada",
		zap.String("user", evt.Info.Sender.User),
		zap.String("msg", msg),
	)

	ctx, cancel := context.WithTimeout(context.Background(), handlerTimeoutMention)
	defer cancel()

	args := []string{msg}
	if err := cmd.Handler(ctx, r.client, evt, args); err != nil {
		r.log.Error("Erro ao responder menção", zap.Error(err))
	}
}

func (r *Router) handlePrefixedCommand(evt *events.Message, cmdName string, args []string) {
	// Middlewares antes do log: mensagens filtradas não poluem o terminal.
	if !r.runMiddlewares(cmdName, evt) {
		return
	}

	r.log.Info("Comando recebido",
		zap.String("command", cmdName),
		zap.String("user", evt.Info.Sender.User),
		zap.Strings("args", args),
	)

	cmd, ok := r.commands[cmdName]
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), handlerTimeoutCommand)
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
		zap.String("date", time.Now().Format("2006-01-02 15:04:05")),
	)
}

// isMentioned detecta atalho por palavra-chave (case-insensitive), sem depender do prefixo.
func isMentioned(msg string) bool {
	return strings.Contains(strings.ToLower(msg), "shinobu")
}
