package app

import "github.com/Turgho/Shinobu-Whatsapp/internal/commands"

// registerAliases mapeia atalhos e erros comuns de digitação para comandos reais.
func registerAliases(r *commands.Router) {
	aliases := map[string]string{
		// Shortcuts
		"p": "play",
		"s": "sticker",
		"m": "menu",
		"c": "clima",
		"e": "efeito",
		"a": "aniversário",

		// Common misspellings
		"plau":        "play",
		"plei":        "play",
		"stiker":      "sticker",
		"figurinha":   "sticker",
		"clim":        "clima",
		"tempo":       "clima",
		"aniver":      "aniversário",
		"aniversario": "aniversário",
		"lembrete":    "agenda",

		// Cotação
		"cotação": "cotacao",
		"dolar":   "cotacao",
		"dólar":   "cotacao",
		"euro":    "cotacao",
		"cot":     "cotacao",

		// Feriado
		"feriados": "feriado",

		// Notícia
		"noticias": "noticia",
		"notícia":  "noticia",
		"notícias": "noticia",
		"news":     "noticia",

		// Receita
		"receitas": "receita",

		// Piada
		"piadas": "piada",

		// Fato
		"fatos":       "fato",
		"curiosidade": "fato",

		// Filme
		"filmes": "filme",
		"movie":  "filme",

		// Contagem
		"dias":      "contagem",
		"countdown": "contagem",

		// Unsticker
		"desticker":        "unsticker",
		"figurinha2imagem": "unsticker",
		"us":               "unsticker",

		// Traduz
		"traduzir":  "traduz",
		"translate": "traduz",
	}

	for alias, target := range aliases {
		r.RegisterAlias(alias, target)
	}
}
