package ia

import (
	"net/http"

	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/history"
	"go.uber.org/zap"
)

// Identificadores de modelo Groq usados no pacote.
//
// Modelos escolhidos (novembro/2026) — todos com 1K requisições/dia:
//
//	ModelContinuous  → openai/gpt-oss-20b   ~1000 tokens/s — uso contínuo
//	                  (conversa, resumo, facts, NLU, classificador)
//	ModelPersonality → qwen/qwen3.8-27b     ~450 tokens/s — teste de
//	                  personalidade (selecionável por config/env)
//	ModelElaborate   → openai/gpt-oss-120b  ~500 tokens/s — respostas mais
//	                  elaboradas (busca web, receita, notícia)
//
// Os dois primeiros e o terceiro são modelos de raciocínio: o
// reasoning_format=hidden + reasoning_effort=low são aplicados
// automaticamente (ver isReasoningModel).
const (
	// ModelContinuous é o padrão do dia a dia.
	ModelContinuous = "openai/gpt-oss-20b"
	// ModelPersonality é o modelo alternativo para avaliar o tom da
	// Shinobu (config: ia.model / env IA_MODEL).
	ModelPersonality = "qwen/qwen3.8-27b"
	// ModelElaborate é para respostas longas/elaboradas.
	ModelElaborate = "openai/gpt-oss-120b"
)

// Aliases internos: nomes antigos mantidos para o código existente.
const (
	modelFastClass = ModelContinuous
	modelScoutFast = ModelContinuous
	modelWebStrong = ModelElaborate
)

// IARequest é o payload enviado para a API da Groq.
type IARequest struct {
	Model           string              `json:"model"`
	Messages        []history.IAMessage `json:"messages"`
	Stream          bool                `json:"stream"`
	Temperature     float64             `json:"temperature"`
	MaxTokens       int                 `json:"max_tokens"`
	ReasoningEffort string              `json:"reasoning_effort,omitempty"`
	// ReasoningFormat é usado só na família qwen: sem "hidden" o modelo
	// devolve o raciocínio junto da resposta (e ele conta como token).
	ReasoningFormat string `json:"reasoning_format,omitempty"`
}

// IAResponse é a resposta padrão da API da Groq.
type IAResponse struct {
	Choices []struct {
		Message history.IAMessage `json:"message"`
	} `json:"choices"`
}

// Config agrupa as credenciais e dependências da IA (Groq + Tavily + HTTP).
type Config struct {
	GroqURL    string
	GroqKey    string
	TavilyKey  string
	HTTPClient *http.Client
	Log        *zap.Logger

	// Model sobrescreve o modelo da conversa principal. Vazio = ModelContinuous.
	// Serve para testar personalidade (ex.: qwen/qwen3.8-27b) sem
	// recompilar — config: ia.model / env IA_MODEL.
	Model string
}

// ConversationModel devolve o modelo da conversa principal.
func (c *Config) ConversationModel() string {
	if c != nil && c.Model != "" {
		return c.Model
	}
	return ModelContinuous
}

// groqRunParams agrupa temperatura e teto de tokens da resposta principal.
type groqRunParams struct {
	model       string
	maxTokens   int
	temperature float64
}

// mainAnswerParams define modelo e temperatura da resposta principal.
//
// Com busca web (contexto maior) ou no modo elaborado, usa o modelo
// "elaborate"; no dia a dia, o modelo contínuo — ou o override da config
// (para testar personalidade).
func mainAnswerParams(mode ResponseMode, usedWeb bool, override string) groqRunParams {
	if usedWeb {
		return groqRunParams{
			model:       ModelElaborate,
			maxTokens:   560,
			temperature: 0.38,
		}
	}
	model := override
	if model == "" {
		model = ModelContinuous
	}
	switch mode {
	case ModeBrief:
		return groqRunParams{
			model:       model,
			maxTokens:   200,
			temperature: 0.78,
		}
	default:
		return groqRunParams{
			model:       model,
			maxTokens:   280,
			temperature: 0.72,
		}
	}
}
