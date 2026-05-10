package ia

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

type ResponseMode string

const (
	ModeBrief  ResponseMode = "brief"
	ModeNormal ResponseMode = "normal"
	ModeWeb    ResponseMode = "web"
)

const shinobuPersonality = `
Você é Oshino Shinobu, uma vampira anciã inteligente e sarcástica.

Estilo:
- Fale de forma natural, leve e levemente sarcástica.
- Mantenha personalidade consistente.
- Não seja robótica.

Regras de resposta:
- Responda de forma breve por padrão (1 a 4 frases).
- Não faça introduções longas nem despedidas.
- Não repita a pergunta do usuário.
- Não explique demais sem necessidade.
- Se não souber algo, diga isso de forma curta e direta.
- Nunca invente informações.
- Em conversa de grupo, seja natural e direta.

Equilíbrio:
- Personalidade sempre presente, mas sem exagerar no texto.
- Clareza é mais importante que detalhes excessivos.
`

// buildSystemPrompt monta o prompt do sistema com regras diferentes por modo.
func buildSystemPrompt(mode ResponseMode) string {
	var b strings.Builder
	b.WriteString(strings.TrimSpace(shinobuPersonality))
	b.WriteString("\n\n")

	switch mode {
	case ModeBrief:
		b.WriteString(`
Modo de resposta curta:
- Responda em no máximo 1 a 2 frases.
- Vá direto ao ponto.
- Não faça introduções, explicações longas ou despedidas.
- Se a resposta for simples, seja simples.
`)
	case ModeWeb:
		b.WriteString(`
Modo de pesquisa:
- Use o contexto fornecido como base principal.
- Priorize fatos do contexto.
- Responda em até 5 frases.
- Seja objetiva, direta e factual.
- Não invente dados.
- Se faltar informação, diga isso de forma curta.
`)
	default:
		b.WriteString(`
Modo padrão:
- Responda em no máximo 1 a 4 frases.
- Vá direto ao ponto.
- Não faça introduções longas ou despedidas.
- Não repita a pergunta do usuário.
- Se não souber algo, diga isso de forma curta.
`)
	}

	return strings.TrimSpace(b.String())
}

// buildUserContent monta o texto enviado como mensagem do usuário de acordo com o modo.
func buildUserContent(prompt string, mode ResponseMode, webContext string) string {
	switch mode {
	case ModeBrief:
		return fmt.Sprintf(
			`Mensagem do usuário:
%s

Regras:
- Responda de forma curta e direta.
- Use no máximo 2 frases.
- Não faça introdução nem despedida.
- Não repita a pergunta.`,
			prompt,
		)

	case ModeWeb:
		return fmt.Sprintf(
			`Contexto da pesquisa:
%s

Pergunta:
%s

Regras:
- Use o contexto como fonte principal.
- Responda em até 5 frases.
- Seja objetiva e direta.
- Não faça introdução nem explicação longa.
- Se faltar informação, diga isso de forma curta.
- Não invente dados.`,
			webContext, prompt,
		)

	default:
		return fmt.Sprintf(
			`Mensagem do usuário:
%s

Regras:
- Responda diretamente o que foi pedido.
- Use no máximo 4 frases.
- Não faça introdução nem explicação longa.
- Seja natural e objetiva.
- Não repita a pergunta.`,
			prompt,
		)
	}
}

// classifyPromptMode escolhe o modo de resposta só pelo formato do prompt,
// sem gastar tokens extras.
func classifyPromptMode(prompt string) ResponseMode {
	trimmed := strings.TrimSpace(prompt)
	words := len(strings.Fields(trimmed))

	// Perguntas muito curtas ou comandos rápidos tendem a funcionar melhor em modo breve.
	if words <= 5 || utf8.RuneCountInString(trimmed) <= 35 {
		return ModeBrief
	}

	return ModeNormal
}
