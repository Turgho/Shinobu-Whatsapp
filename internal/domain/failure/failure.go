// Package failure classifica falhas para a resposta ao usuário.
//
// A taxonomia já existia dentro do domain/music (ErrCategory +
// CategoryText) — aqui ela sai do subsistema de música e passa a ser
// compartilhada: clima, geocoding, feriados e cotações também passam a
// dizer COMO falharam, e o router usa isso para responder em vez de
// silenciar.
//
// Não é um framework de erros: é uma categoria + uma frase.
package failure

import (
	"context"
	"errors"
	"net"
)

// Category classifica a falha.
type Category int

const (
	// Unknown é a categoria padrão (não classificável).
	Unknown Category = iota
	// Unavailable: serviço externo fora do ar ou lento demais.
	Unavailable
	// Forbidden: o serviço recusou a chamada (403, 429, bot protection).
	Forbidden
	// InvalidInput: a entrada do usuário não faz sentido.
	InvalidInput
	// NotFound: o recurso procurado não existe.
	NotFound
)

// UserMessage devolve a resposta amigável (na voz da Shinobu), sem
// detalhe técnico.
func (c Category) UserMessage() string {
	switch c {
	case Unavailable:
		return "Não consegui falar com o serviço agora. Tenta de novo daqui a pouco."
	case Forbidden:
		return "O serviço recusou a minha chamada agora. Tenta mais tarde."
	case InvalidInput:
		return "Não entendi esse pedido. Tenta de novo ou manda !menu pra ver o que dá pra fazer."
	case NotFound:
		return "Não achei isso aqui."
	default:
		return "Algo deu errado do meu lado. Tenta de novo?"
	}
}

// Categorizer permite que um erro declare a própria categoria.
type Categorizer interface {
	FailureCategory() Category
}

// statusCoder é implementado por erros que carregam o status HTTP.
type statusCoder interface {
	StatusCode() int
}

// Classify deduz a categoria de um erro:
// 1) erro que declara a categoria (Categorizer);
// 2) timeout/cancelamento (contexto ou rede);
// 3) status HTTP (403/429 → Forbidden, 404 → NotFound, 5xx → Unavailable);
// 4) Unknown.
func Classify(err error) Category {
	if err == nil {
		return Unknown
	}

	var cat Categorizer
	if errors.As(err, &cat) {
		return cat.FailureCategory()
	}

	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return Unavailable
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return Unavailable
	}

	var status statusCoder
	if errors.As(err, &status) {
		switch code := status.StatusCode(); {
		case code == 403 || code == 429:
			return Forbidden
		case code == 404:
			return NotFound
		case code >= 500:
			return Unavailable
		}
	}

	return Unknown
}
