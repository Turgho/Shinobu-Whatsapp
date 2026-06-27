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

// dispatchableCommands são os comandos públicos que a NLU pode despachar.
// Comandos de admin/owner não entram aqui — exigem verificação de dono.
var dispatchableCommands = map[string]bool{
	"clima":       true,
	"play":        true,
	"sticker":     true,
	"efeito":      true,
	"aniversário": true,
}

func dispatchableCommand(name string) bool {
	return dispatchableCommands[name]
}

type rateLimitEntry struct {
	count   int
	resetAt time.Time
}

type Router struct {
	commands    map[string]command
	aliases     map[string]string // alias → canonical name
	middlewares []Middleware
	prefix      string
	client      *whatsmeow.Client
	log         *zap.Logger
	store       *history.Store

	rateLimitMu    sync.Mutex
	rateLimitMap   map[string]*rateLimitEntry
	rateLimitMax   int
	rateLimitEvery time.Duration

	botJID      string
	maintenance atomic.Bool
	aiConfig    *ia.Config
}

func NewRouter(prefix string, client *whatsmeow.Client, log *zap.Logger, store *history.Store) *Router {
	return &Router{
		commands:       make(map[string]command),
		aliases:        make(map[string]string),
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

// StartRateLimitCleanup inicia uma goroutine que remove entradas expiradas do rateLimitMap a cada 10 minutos.
func (r *Router) StartRateLimitCleanup(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				r.cleanRateLimit()
			}
		}
	}()
}

func (r *Router) cleanRateLimit() {
	r.rateLimitMu.Lock()
	defer r.rateLimitMu.Unlock()

	now := time.Now()
	for key, entry := range r.rateLimitMap {
		if now.After(entry.resetAt) {
			delete(r.rateLimitMap, key)
		}
	}
}

func (r *Router) SetBotJID(jid string) {
	r.botJID = jid
}

func (r *Router) SetAIConfig(cfg *ia.Config) {
	r.aiConfig = cfg
	r.log.Info("AI config definido", zap.Bool("groq", cfg.GroqKey != ""), zap.Bool("tavily", cfg.TavilyKey != ""))
}

func (r *Router) SetMaintenance(on bool) {
	r.maintenance.Store(on)
	r.log.Info("Modo manutenção alterado", zap.Bool("on", on))
}

func (r *Router) IsMaintenance() bool {
	return r.maintenance.Load()
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

// HandleMessage processa cada mensagem no pipeline:
// 1. Extrai texto visível
// 2. Verifica se o remetente está na lista de ignorados
// 3. Verifica rate limit
// 4. Se tem prefixo → comando normal (handlePrefixedCommand)
// 5. Se não tem prefixo E (DM ou menção) → NLU via handleNaturalLanguage
// 6. Caso contrário → ignora
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

	if strings.HasPrefix(msg, r.prefix) {
		r.handlePrefixedMessage(evt, msg)
		return
	}

	if r.aiConfig != nil && r.isNLApplicable(evt, msg) {
		r.handleNaturalLanguage(evt, msg)
	}
}

func (r *Router) isNLApplicable(evt *events.Message, msg string) bool {
	chat := evt.Info.Chat.String()
	isGroup := strings.HasSuffix(chat, "@g.us")
	if !isGroup {
		return true
	}
	return isMentioned(msg, r.botJID)
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

// handleNaturalLanguage processa mensagens sem prefixo via NLU.
// 1. Detecta intento via Groq (com timeout de 8s)
// 2. Se comando válido → passa pelo pipeline de middlewares
// 3. Se aprovado → despacha para o handler
// 4. Se falhar ou sem comando → fallback para IA geral
func (r *Router) handleNaturalLanguage(evt *events.Message, msg string) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	intent, err := ia.DetectIntent(ctx, r.aiConfig, msg)
	if err != nil {
		r.log.Debug("NLU: falha na detecção de intento", zap.Error(err))
		r.handleShinobuMention(evt, msg)
		return
	}

	if intent.Command != "" {
		if !dispatchableCommand(intent.Command) {
			r.log.Debug("NLU: comando não-despachável", zap.String("command", intent.Command))
			r.handleShinobuMention(evt, msg)
			return
		}

		if !r.runMiddlewares(intent.Command, evt) {
			return
		}

		handler := r.Handler(intent.Command)
		if handler == nil {
			r.log.Debug("NLU: handler não encontrado", zap.String("command", intent.Command))
			r.handleShinobuMention(evt, msg)
			return
		}

		r.log.Info("NLU: intento detectado e despachado",
			append(eventFields(evt),
				zap.String("command", intent.Command),
				zap.Strings("args", intent.Args),
				zap.String("raw_message", msg),
			)...,
		)

		dispatchCtx, cancelDispatch := context.WithTimeout(context.Background(), handlerTimeoutCommand)
		defer cancelDispatch()
		handler(dispatchCtx, r.client, evt, intent.Args)
		return
	}

	r.log.Info("NLU: sem comando, fallback IA",
		append(eventFields(evt), zap.String("msg", msg))...,
	)
	r.handleShinobuMention(evt, msg)
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
		append(eventFields(evt),
			zap.String("msg", msg),
		)...,
	)

	ctx, cancel := context.WithTimeout(context.Background(), handlerTimeoutMention)
	defer cancel()

	args := []string{msg}
	if err := cmd.Handler(ctx, r.client, evt, args); err != nil {
		r.log.Error("Erro ao responder menção",
			append(eventFields(evt), zap.Error(err))...,
		)
	}
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
