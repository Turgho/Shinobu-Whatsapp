// Package commands implementa o router de comandos do bot, incluindo
// registro, middlewares, rate limit, dispatch via prefixo e NLU.
package commands

import (
	"context"
	"errors"
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
	handlerTimeoutCommand = 180 * time.Second
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

	messageHooks    []func(ctx context.Context, evt *events.Message, msg string)
	shutdownHook    func()
	botJID          string
	maintenance     atomic.Bool
	aiConfig        *ia.Config
	nlpGroupTrigger bool
	intentEnabled   bool

	albumCoordinator *AlbumCoordinator

	// rootCtx é o contexto da aplicação (cancelado no encerramento). Os
	// handlers derivatives dele — assim o shutdown interrompe o trabalho em
	// andamento em vez de deixá-lo rodando até o timeout.
	rootCtx context.Context
}

// NewRouter cria um router com prefixo de comando, client WhatsApp e dependências.
// Rate limit padrão: 10 mensagens por minuto por sender.
func NewRouter(prefix string, client *whatsmeow.Client, log *zap.Logger, store *history.Store, ignoreStore *ignore.Store) *Router {
	r := &Router{
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

	r.albumCoordinator = NewAlbumCoordinator(log)
	r.albumCoordinator.SetBatchHandler(func(cmdName string, args []string, items []*events.Message) {
		r.dispatchBatch(cmdName, args, items)
	})

	return r
}

// --- Setters de configuração ---

// SetRootContext define o contexto raiz da aplicação (cancelado no
// encerramento do bot).
func (r *Router) SetRootContext(ctx context.Context) {
	r.rootCtx = ctx
}

// ctx devolve o contexto raiz da aplicação (ou Background se não definido).
func (r *Router) ctx() context.Context {
	if r.rootCtx != nil {
		return r.rootCtx
	}
	return context.Background()
}

// SetBotJID define o JID do bot para detecção de menção em mensagens.
func (r *Router) SetBotJID(jid string) {
	r.botJID = jid
}

// SetAIConfig configura o provider de IA (Groq + Tavily) para NLU e respostas.
func (r *Router) SetAIConfig(cfg *ia.Config) {
	r.aiConfig = cfg
	r.log.Info("AI config definido", zap.Bool("groq", cfg.GroqKey != ""), zap.Bool("tavily", cfg.TavilyKey != ""))
}

// AIConfig devolve a config de IA em uso (mesma instância do router: com
// logger e HTTP client compartilhados). Retorna nil se não configurada.
func (r *Router) AIConfig() *ia.Config {
	return r.aiConfig
}

// SetNLPGroupTrigger ativa/desativa heurísticas NLU em grupos sem menção explícita.
func (r *Router) SetNLPGroupTrigger(on bool) {
	r.nlpGroupTrigger = on
	r.log.Info("NLP group trigger alterado", zap.Bool("enabled", on))
}

// SetIntentEnabled ativa/desativa o NLU/DetectIntent por completo.
// Quando false, mensagens com menção vão direto para conversa IA.
func (r *Router) SetIntentEnabled(enabled bool) {
	r.intentEnabled = enabled
	r.log.Info("Intent/NLU config alterada", zap.Bool("enabled", enabled))
}

// SetMaintenance ativa/desativa modo manutenção (comandos bloqueados, exceto manutencao).
func (r *Router) SetMaintenance(on bool) {
	r.maintenance.Store(on)
	r.log.Info("Modo manutenção alterado", zap.Bool("on", on))
}

// SetShutdownHook registra a função que encerra o bot de forma limpa
// (usada pelo comando !shutdown — assim os defer do app rodam).
func (r *Router) SetShutdownHook(fn func()) {
	r.shutdownHook = fn
}

// RequestShutdown pede o encerramento limpo do bot. Sem hook registrado,
// nada acontece (o comando fica inerte em vez de matar o processo).
func (r *Router) RequestShutdown() bool {
	if r.shutdownHook == nil {
		return false
	}
	r.shutdownHook()
	return true
}

// AddMessageHook registra uma função chamada para cada mensagem recebida,
// antes do roteamento (prefixo ou NLU). Útil para logging ou armazenamento
// específico sem poluir o router.
func (r *Router) AddMessageHook(hook func(ctx context.Context, evt *events.Message, msg string)) {
	r.messageHooks = append(r.messageHooks, hook)
}

// IsMaintenance retorna true se o bot está em modo manutenção.
func (r *Router) IsMaintenance() bool {
	return r.maintenance.Load()
}

// --- Registro de comandos e aliases ---

// RegisterCommand registra um comando no router com metadados e handler.
func (r *Router) RegisterCommand(meta CommandMeta, handler HandlerFunc) {
	r.commands[meta.Name] = command{Meta: meta, Handler: handler}
	r.log.Info("Comando registrado",
		zap.String("command", meta.Name),
		zap.Bool("private", meta.Private),
	)
}

// RegisterBatchCommand registra um comando com handler batch para albums.
// O BatchHandler é chamado quando o comando é usado em um album (múltiplas imagens/vídeos).
func (r *Router) RegisterBatchCommand(meta CommandMeta, handler HandlerFunc, batchHandler BatchHandlerFunc) {
	r.commands[meta.Name] = command{Meta: meta, Handler: handler, BatchHandler: batchHandler}
	r.log.Info("Comando batch registrado",
		zap.String("command", meta.Name),
		zap.Bool("private", meta.Private),
	)
}

// RegisterAlias registra um alias que redireciona para um comando existente.
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

// Use adiciona um middleware ao pipeline (executado na ordem de registro).
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

// HasCommand verifica se um comando (ou alias) está registrado.
func (r *Router) HasCommand(name string) bool {
	_, ok := r.commands[r.resolveAlias(name)]
	return ok
}

// IsPrivate retorna true se o comando é marcado como privado (requer owner/admin).
func (r *Router) IsPrivate(name string) bool {
	cmd, ok := r.commands[r.resolveAlias(name)]
	return ok && cmd.Meta.Private
}

// Prefix retorna o prefixo de comando configurado (ex: "!").
func (r *Router) Prefix() string {
	return r.prefix
}

// Handler retorna a HandlerFunc registrada para o nome, ou nil se não existir.
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

	// Itens filho de album (sem texto): bufferiza e aguarda dispatch.
	// Se tem texto (comando na legenda), deixa o handlePrefixedCommand detectar o album.
	if HasMediaAlbumAssociation(evt) {
		if whatsapp.VisibleTextFromEvent(evt) == "" {
			parentID := ParentMessageID(evt)
			if parentID != "" {
				r.albumCoordinator.Bufferize(parentID, evt, "", nil, 0)
			}
			return
		}
		// Tem texto: continua no pipeline — handlePrefixedCommand vai detectar o album.
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

	hookCtx, hookCancel := context.WithTimeout(r.ctx(), 30*time.Second)
	for _, hook := range r.messageHooks {
		hook(hookCtx, evt, msg)
	}
	hookCancel()

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
		ctx, cancel := context.WithTimeout(r.ctx(), 10*time.Second)
		defer cancel()
		maintenanceMsg := "😅 *O bot está em manutenção!*\n\nComandos temporariamente desativados. Tente novamente mais tarde."
		if err := whatsapp.Reply(ctx, r.client, evt, maintenanceMsg); err != nil {
			r.log.Warn("Falha ao enviar mensagem de manutenção", zap.Error(err))
		}
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

	// Album: se o comando tem BatchHandler e a mensagem é de album,
	// redireciona para o AlbumCoordinator em vez de executar direto.
	if cmd.BatchHandler != nil && (IsAlbumMessage(evt) || HasMediaAlbumAssociation(evt)) {
		parentID := evt.Info.ID
		expected := AlbumExpectedCount(evt)

		// Se é item filho (sem AlbumMessage), herda expected do entry existente.
		if expected == 0 && HasMediaAlbumAssociation(evt) {
			if refID := ParentMessageID(evt); refID != "" {
				parentID = refID
			}
		}

		r.log.Info("Album detectado, buffering",
			append(eventFields(evt),
				zap.String("command", cmdName),
				zap.String("parentID", parentID),
				zap.Int("expected", expected),
			)...,
		)
		if r.albumCoordinator.Bufferize(parentID, evt, cmdName, args, expected) {
			r.albumCoordinator.Dispatch(parentID, func(cn string, a []string, items []*events.Message) {
				r.dispatchBatch(cn, a, items)
			})
		}
		return
	}

	ctx, cancel := context.WithTimeout(r.ctx(), handlerTimeoutCommand)
	defer cancel()

	start := time.Now()
	if err := cmd.Handler(ctx, r.client, evt, args); err != nil {
		// ErrNotACommand = "este handler não é o dono desta entrada".
		// No caminho NLU isso cai na conversa da IA; no caminho com prefixo
		// o usuário ficaria sem resposta nenhuma — então avisamos.
		if errors.Is(err, ErrNotACommand) {
			r.log.Info("Comando não reconhecido para esta entrada",
				append(eventFields(evt),
					zap.String("command", cmdName),
				)...,
			)
			r.replyNotACommand(ctx, evt, err)
			return
		}

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

// replyNotACommand responde ao usuário quando o handler não reconheceu a
// entrada. Sem isso o comando terminava em silêncio — o pior sintoma para
// quem usa o bot.
func (r *Router) replyNotACommand(ctx context.Context, evt *events.Message, err error) {
	msg := notACommandMessage(err)
	if sendErr := whatsapp.Reply(ctx, r.client, evt, msg); sendErr != nil {
		r.log.Warn("Falha ao responder comando não reconhecido", zap.Error(sendErr))
	}
}

// notACommandMessage escolhe a mensagem amigável para o usuário quando o
// handler não reconheceu a entrada.
func notACommandMessage(err error) string {
	const fallback = "Não entendi esse pedido. Tenta de novo ou manda !menu pra ver o que dá pra fazer."
	var nce *NotACommandError
	if errors.As(err, &nce) && nce.Message != "" {
		return nce.Message
	}
	return fallback
}

// dispatchBatch despacha um batch de itens de album para o BatchHandler do comando.
func (r *Router) dispatchBatch(cmdName string, args []string, items []*events.Message) {
	cmd, ok := r.commands[cmdName]
	if !ok || cmd.BatchHandler == nil {
		r.log.Warn("album: comando batch não encontrado",
			zap.String("command", cmdName),
		)
		return
	}

	r.log.Info("Album batch despachado",
		zap.String("command", cmdName),
		zap.Int("items", len(items)),
	)

	ctx, cancel := context.WithTimeout(r.ctx(), handlerTimeoutCommand)
	defer cancel()

	start := time.Now()
	if err := cmd.BatchHandler(ctx, r.client, items, args); err != nil {
		r.log.Error("Erro no batch command",
			zap.String("command", cmdName),
			zap.Error(err),
		)
		return
	}

	r.log.Info("Batch command executado",
		zap.String("command", cmdName),
		zap.Int("items", len(items)),
		zap.Duration("duration", time.Since(start)),
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
