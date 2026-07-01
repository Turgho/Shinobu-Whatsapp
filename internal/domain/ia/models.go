package ia

import (
	"net/http"

	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/history"
	"go.uber.org/zap"
)

// Identificadores de modelo Groq usados no pacote.
//
// Orçamento de rate limit Groq (plano free, junho 2026):
//
//	modelFastClass  → 14.4K RPD, 6K TPM  — classificação e extração JSON
//	modelScoutFast  →  1K RPD, 30K TPM  — conversa, resumo, personalidade
//	modelWebStrong (gpt-oss-120b) → 1K RPD, 8K TPM, 200K TPD  — respostas com contexto Tavily
const (
	ModelFastClass = "llama-3.1-8b-instant"                      // classificação e extração estruturada
	ModelScoutFast = "meta-llama/llama-4-scout-17b-16e-instruct" // conversa, resumo
	ModelWebStrong = "openai/gpt-oss-120b"                       // respostas com contexto Tavily
)

// Aliases internos para compatibilidade com código existente.
const (
	modelFastClass = ModelFastClass
	modelScoutFast = ModelScoutFast
	modelWebStrong = ModelWebStrong
)

// IARequest é o payload enviado para a API da Groq.
type IARequest struct {
	Model       string              `json:"model"`
	Messages    []history.IAMessage `json:"messages"`
	Stream      bool                `json:"stream"`
	Temperature float64             `json:"temperature"`
	MaxTokens   int                 `json:"max_tokens"`
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
}

// groqRunParams agrupa temperatura e teto de tokens da resposta principal.
type groqRunParams struct {
	model       string
	maxTokens   int
	temperature float64
}

func mainAnswerParams(mode ResponseMode, usedWeb bool) groqRunParams {
	if usedWeb {
		return groqRunParams{
			model:       modelWebStrong,
			maxTokens:   560,
			temperature: 0.38,
		}
	}
	switch mode {
	case ModeBrief:
		return groqRunParams{
			model:       modelScoutFast,
			maxTokens:   200,
			temperature: 0.78,
		}
	default:
		return groqRunParams{
			model:       modelScoutFast,
			maxTokens:   280,
			temperature: 0.72,
		}
	}
}
