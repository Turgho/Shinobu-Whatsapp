package ia

// Identificadores de modelo Groq usados no pacote.
const (
	modelScoutFast = "meta-llama/llama-4-scout-17b-16e-instruct" // conversa + classificação leve
	modelWebStrong = "llama-3.3-70b-versatile"                   // respostas com contexto Tavily
)

// groqRunParams agrupa temperatura e teto de tokens da resposta principal.
// Valores calibrados para: (1) conversa natural sem respostas truncadas demais;
// (2) modo web mais factual, com espaço para síntese de várias fontes.
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
