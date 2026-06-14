package commands

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/history"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/ignore"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
	"go.uber.org/zap"
)

const (
	handlerTimeoutMention = 30 * time.Second
	handlerTimeoutCommand = 120 * time.Second
)

type rateLimitEntry struct {
	count   int
	resetAt time.Time
}

type Router struct {
	commands    map[string]command
	middlewares []Middleware
	prefix      string
	client      *whatsmeow.Client
	log         *zap.Logger
	store       *history.Store

	rateLimitMu    sync.Mutex
	rateLimitMap   map[string]*rateLimitEntry
	rateLimitMax   int
	rateLimitEvery time.Duration

	botJID string
}

func NewRouter(prefix string, client *whatsmeow.Client, log *zap.Logger, store *history.Store) *Router {
	return &Router{
		commands:       make(map[string]command),
		prefix:         prefix,
		client:         client,
		log:            log,
		store:          store,
		rateLimitMap:   make(map[string]*rateLimitEntry),
		rateLimitMax:   10,
		rateLimitEvery: time.Minute,
	}
}

func (r *Router) SetRateLimit(max int, every time.Duration) {
	r.rateLimitMax = max
	r.rateLimitEvery = every
}

func (r *Router) SetBotJID(jid string) {
	r.botJID = jid
}

// checkRateLimit implementa rate limiting por chave (sender JID).
// Usa janela fixa: cada chave tem um contador e resetAt.
// Quando resetAt passa, o contador é zerado.
// Se o contador excede o máximo, a requisição é bloqueada.
func (r *Router) checkRateLimit(key string) bool {
	r.rateLimitMu.Lock()
	defer r.rateLimitMu.Unlock()

	now := time.Now()
	entry, ok := r.rateLimitMap[key]
	if !ok || now.After(entry.resetAt) {
		r.rateLimitMap[key] = &rateLimitEntry{
			count:   1,
			resetAt: now.Add(r.rateLimitEvery),
		}
		return true
	}

	entry.count++
	if entry.count > r.rateLimitMax {
		return false
	}
	return true
}

func (r *Router) RegisterCommand(meta CommandMeta, handler HandlerFunc) {
	r.commands[meta.Name] = command{Meta: meta, Handler: handler}
	r.log.Info("Comando registrado",
		zap.String("command", meta.Name),
		zap.Bool("private", meta.Private),
	)
}

func (r *Router) Use(m Middleware) {
	r.middlewares = append(r.middlewares, m)
}

func (r *Router) Commands() []CommandMeta {
	metas := make([]CommandMeta, 0, len(r.commands))
	for _, cmd := range r.commands {
		metas = append(metas, cmd.Meta)
	}
	return metas
}

func (r *Router) HasCommand(name string) bool {
	_, ok := r.commands[name]
	return ok
}

func (r *Router) IsPrivate(name string) bool {
	cmd, ok := r.commands[name]
	return ok && cmd.Meta.Private
}

func (r *Router) Prefix() string {
	return r.prefix
}

// HandleMessage processa cada mensagem no pipeline:
// 1. Extrai texto visível
// 2. Verifica se o remetente está na lista de ignorados
// 3. Verifica rate limit
// 4. Se é menção → IA via handleShinobuMention
// 5. Se é comando prefixado → handlePrefixedCommand
func (r *Router) HandleMessage(evt *events.Message) {
	msg := whatsapp.VisibleTextFromEvent(evt)
	if msg == "" {
		return
	}

	sender := evt.Info.Sender.String()
	senderUser := evt.Info.Sender.User
	if ignore.IsIgnored(sender) || ignore.IsIgnored(senderUser) {
		r.log.Debug("Mensagem ignorada", zap.String("sender", sender))
		return
	}

	if !r.checkRateLimit(sender) {
		return
	}

	if isMentioned(msg, r.botJID) {
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

func (r *Router) Handler(name string) HandlerFunc {
	cmd, ok := r.commands[name]
	if !ok {
		return nil
	}
	return cmd.Handler
}

// runMiddlewares executa a cadeia de middlewares em ordem.
// Se algum retornar false, o pipeline é interrompido.
func (r *Router) runMiddlewares(cmdName string, evt *events.Message) bool {
	for _, m := range r.middlewares {
		if !m(cmdName, evt) {
			return false
		}
	}
	return true
}

// handleShinobuMention executa o handler shinobu (IA) quando o bot é mencionado.
// Tem timeout menor (30s) porque menções em grupo precisam de resposta rápida.
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

// handlePrefixedCommand executa um comando registrado com prefixo.
// Passa pelos middlewares, executa com timeout e loga duração.
func (r *Router) handlePrefixedCommand(evt *events.Message, cmdName string, args []string) {
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

// isMentioned verifica se a mensagem menciona a Shinobu.
// Usa dois critérios: o nome "shinobu" (case insensitive) e
// a menção explícita via @jid (para grupos).
func isMentioned(msg, botJID string) bool {
	lower := strings.ToLower(msg)
	if strings.Contains(lower, "shinobu") {
		return true
	}
	if botJID != "" && strings.Contains(msg, botJID) {
		return true
	}
	return false
}
