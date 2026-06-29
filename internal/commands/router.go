// Package commands implementa o router de comandos do bot, incluindo
// registro, middlewares, rate limit, dispatch via prefixo e NLU.
package commands

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/history"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/ia"
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

// Router coordena registro de comandos, middlewares, rate limit e dispatch.
// Mensagens entram via HandleMessage e são roteadas para handlers ou NLU.
type Router struct {
	commands    map[string]command
	aliases     map[string]string // alias → canonical name
	middlewares []Middleware
	prefix      string
	client      *whatsmeow.Client
	log         *zap.Logger
	store       *history.Store
	ignoreStore *ignore.Store

	rateLimitMu    sync.Mutex
	rateLimitMap   map[string]*rateLimitEntry
	rateLimitMax   int
	rateLimitEvery time.Duration

	botJID          string
	maintenance     atomic.Bool
	aiConfig        *ia.Config
	nlpGroupTrigger bool
}

func NewRouter(prefix string, client *whatsmeow.Client, log *zap.Logger, store *history.Store, ignoreStore *ignore.Store) *Router {
	return &Router{
		commands:       make(map[string]command),
		aliases:        make(map[string]string),
		prefix:         prefix,
		client:         client,
		log:            log,
		store:          store,
		ignoreStore:    ignoreStore,
		rateLimitMap:   make(map[string]*rateLimitEntry),
		rateLimitMax:   10,
		rateLimitEvery: time.Minute,
	}
}

// --- Setters de configuração ---

func (r *Router) SetBotJID(jid string) {
	r.botJID = jid
}

func (r *Router) SetAIConfig(cfg *ia.Config) {
	r.aiConfig = cfg
	r.log.Info("AI config definido", zap.Bool("groq", cfg.GroqKey != ""), zap.Bool("tavily", cfg.TavilyKey != ""))
}

func (r *Router) SetNLPGroupTrigger(on bool) {
	r.nlpGroupTrigger = on
	r.log.Info("NLP group trigger alterado", zap.Bool("enabled", on))
}

func (r *Router) SetMaintenance(on bool) {
	r.maintenance.Store(on)
	r.log.Info("Modo manutenção alterado", zap.Bool("on", on))
}

func (r *Router) IsMaintenance() bool {
	return r.maintenance.Load()
}

// --- Registro de comandos e aliases ---

func (r *Router) RegisterCommand(meta CommandMeta, handler HandlerFunc) {
	r.commands[meta.Name] = command{Meta: meta, Handler: handler}
	r.log.Info("Comando registrado",
		zap.String("command", meta.Name),
		zap.Bool("private", meta.Private),
	)
}

func (r *Router) RegisterAlias(alias, target string) {
	if _, ok := r.commands[alias]; ok {
		r.log.Warn("Alias conflita com comando existente", zap.String("alias", alias))
		return
	}
	if _, ok := r.commands[target]; !ok {
		r.log.Warn("Alias aponta para comando inexistente", zap.String("alias", alias), zap.String("target", target))
		return
	}
	r.aliases[alias] = target
	r.log.Debug("Alias registrado", zap.String("alias", alias), zap.String("target", target))
}

func (r *Router) resolveAlias(name string) string {
	if target, ok := r.aliases[name]; ok {
		return target
	}
	return name
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
	_, ok := r.commands[r.resolveAlias(name)]
	return ok
}

func (r *Router) IsPrivate(name string) bool {
	cmd, ok := r.commands[r.resolveAlias(name)]
	return ok && cmd.Meta.Private
}

func (r *Router) Prefix() string {
	return r.prefix
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

// --- Pipeline de mensagens ---

// HandleMessage processa cada mensagem no pipeline:
//  1. Extrai texto visível
//  2. Verifica se o remetente está na lista de ignorados
//  3. Verifica rate limit
//  4. Se tem prefixo → comando normal (handlePrefixedCommand)
//  5. Se não tem prefixo E (DM ou menção) → NLU via handleNaturalLanguage
//  6. Caso contrário → ignora
func (r *Router) HandleMessage(evt *events.Message) {
	// Ignora mensagens de broadcast e status do WhatsApp.
	// Status updates usam chat JID "status@broadcast" (Server: "broadcast").
	// Broadcast lists usam JIDs como "123456@broadcast".
	// Ambos devem ser ignorados — respostas a status são visíveis publicamente.
	if evt.Info.Chat.Server == "broadcast" {
		r.log.Debug("Mensagem de broadcast ignorada",
			zap.String("chat", evt.Info.Chat.String()),
			zap.String("sender", evt.Info.Sender.String()),
		)
		return
	}

	msg := whatsapp.VisibleTextFromEvent(evt)
	if msg == "" {
		return
	}

	sender := evt.Info.Sender.String()
	senderUser := evt.Info.Sender.User
	if r.ignoreStore.IsIgnored(sender) || r.ignoreStore.IsIgnored(senderUser) {
		r.log.Debug("Mensagem ignorada", zap.String("sender", sender))
		return
	}

	rateLimitKey := evt.Info.Sender.ToNonAD().String()
	if !r.checkRateLimit(rateLimitKey) {
		return
	}

	if strings.HasPrefix(msg, r.prefix) {
		r.handlePrefixedMessage(evt, msg)
		return
	}

	if r.aiConfig != nil && r.isNLApplicable(evt, msg) {
		r.handleNaturalLanguage(evt, msg)
	}
}

func (r *Router) handlePrefixedMessage(evt *events.Message, msg string) {
	parts := strings.Fields(strings.TrimPrefix(msg, r.prefix))
	if len(parts) == 0 {
		return
	}

	cmdName := strings.ToLower(parts[0])

	if r.IsMaintenance() && r.resolveAlias(cmdName) != "manutencao" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		maintenanceMsg := "😅 *O bot está em manutenção!*\n\nComandos temporariamente desativados. Tente novamente mais tarde."
		_ = whatsapp.Reply(ctx, r.client, evt, maintenanceMsg)
		return
	}

	r.handlePrefixedCommand(evt, cmdName, parts[1:])
}

// handlePrefixedCommand executa um comando registrado com prefixo.
// Passa pelos middlewares, executa com timeout e loga duração.
func (r *Router) handlePrefixedCommand(evt *events.Message, cmdName string, args []string) {
	if !r.runMiddlewares(cmdName, evt) {
		return
	}

	cmdName = r.resolveAlias(cmdName)

	r.log.Info("Comando recebido",
		append(eventFields(evt),
			zap.String("command", cmdName),
			zap.Strings("args", args),
		)...,
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
			append(eventFields(evt),
				zap.String("command", cmdName),
				zap.Error(err),
			)...,
		)
		return
	}

	r.log.Info("Comando executado",
		append(eventFields(evt),
			zap.String("command", cmdName),
			zap.Duration("duration", time.Since(start)),
			zap.String("date", time.Now().Format("2006-01-02 15:04:05")),
		)...,
	)
}

// eventFields retorna campos de log comuns extraídos de um evento.
func eventFields(evt *events.Message) []zap.Field {
	chat := evt.Info.Chat.String()
	sender := evt.Info.Sender.ToNonAD().String()
	return []zap.Field{
		zap.String("sender", sender),
		zap.String("chat", chat),
		zap.Bool("is_group", strings.HasSuffix(chat, "@g.us")),
	}
}
