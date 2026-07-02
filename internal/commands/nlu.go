package commands

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/ia"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"go.mau.fi/whatsmeow/types/events"
	"go.uber.org/zap"
)

// handleNaturalLanguage processa mensagens sem prefixo via NLU.
// 1. Detecta intento via Groq (com timeout de 8s)
// 2. Se comando válido → passa pelo pipeline de middlewares
// 3. Se aprovado → despacha para o handler
// 4. Se falhar ou sem comando → fallback para IA geral
func (r *Router) handleNaturalLanguage(evt *events.Message, msg string) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	quotedText := whatsapp.ExtractQuotedText(evt)

	intent, err := ia.DetectIntent(ctx, r.aiConfig, msg, quotedText)
	if err != nil {
		r.log.Debug("NLU: falha na detecção de intento", zap.Error(err))
		r.handleShinobuMention(evt, msg, quotedText)
		return
	}

	if intent.Command != "" {
		if !ia.DispatchableCommand(intent.Command) {
			r.log.Debug("NLU: comando não-despachável", zap.String("command", intent.Command))
			r.handleShinobuMention(evt, msg, quotedText)
			return
		}

		if !r.runMiddlewares(intent.Command, evt) {
			return
		}

		handler := r.Handler(intent.Command)
		if handler == nil {
			r.log.Debug("NLU: handler não encontrado", zap.String("command", intent.Command))
			r.handleShinobuMention(evt, msg, quotedText)
			return
		}

		r.log.Info("NLU: intento detectado e despachado",
			append(eventFields(evt),
				zap.String("command", intent.Command),
				zap.Strings("args", intent.Args),
				zap.String("raw_message", msg),
				zap.Bool("has_context", quotedText != ""),
			)...,
		)

		dispatchCtx, cancelDispatch := context.WithTimeout(context.Background(), handlerTimeoutCommand)
		defer cancelDispatch()
		if err := handler(dispatchCtx, r.client, evt, intent.Args); err != nil {
			if errors.Is(err, ErrNotACommand) {
				r.log.Debug("NLU: handler rejeitou comando, fallback IA",
					zap.String("command", intent.Command),
					zap.Error(err),
				)
				r.handleShinobuMention(evt, msg, quotedText)
				return
			}
			r.log.Warn("NLU: handler error",
				append(eventFields(evt),
					zap.String("command", intent.Command),
					zap.Error(err),
				)...,
			)
		}
		return
	}

	r.log.Info("NLU: sem comando, fallback IA",
		append(eventFields(evt), zap.String("msg", msg))...,
	)
	r.handleShinobuMention(evt, msg, quotedText)
}

// handleShinobuMention executa o handler shinobu (IA) quando o bot é mencionado
// ou quando cai no fallback da NLU. Se houver contexto de reply, anexa o texto
// da mensagem anterior do bot para a IA continuar a conversa naturalmente.
// Tem timeout menor (30s) porque menções em grupo precisam de resposta rápida.
func (r *Router) handleShinobuMention(evt *events.Message, msg string, quotedContext string) {
	if !r.runMiddlewares("shinobu", evt) {
		return
	}

	cmd, ok := r.commands["shinobu"]
	if !ok {
		return
	}

	if quotedContext != "" {
		msg = fmt.Sprintf("Contexto da conversa (mensagem anterior do bot):\n%s\n\nNova mensagem do usuário:\n%s", quotedContext, msg)
	}

	r.log.Info("Shinobu mencionada",
		append(eventFields(evt),
			zap.String("msg", msg),
			zap.Bool("has_context", quotedContext != ""),
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
