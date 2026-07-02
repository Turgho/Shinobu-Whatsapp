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

		// Play
		"plau":    "play",
		"plei":    "play",
		"musica":  "play",
		"música":  "play",
		"musicas": "play",
		"músicas": "play",
		"audio":   "play",
		"áudio":   "play",
		"som":     "play",

		// Sticker
		"stiker":     "sticker",
		"figurinha":  "sticker",
		"figurinhas": "sticker",
		"fig":        "sticker",
		"figu":       "sticker",
		"stick":      "sticker",

		// Clima
		"clim":     "clima",
		"tempo":    "clima",
		"previsao": "clima",
		"previsão": "clima",

		// Agenda
		"agenda":   "agenda",
		"lembrete": "agenda",
		"lembrar":  "agenda",
		"recordar": "agenda",

		// Aniversário
		"aniver":      "aniversário",
		"aniversario": "aniversário",
		"niver":       "aniversário",
		"anivers":     "aniversário",

		// Cotação
		"cotação": "cotacao",
		"cot":     "cotacao",
		"dolar":   "cotacao",
		"dólar":   "cotacao",
		"euro":    "cotacao",
		"moeda":   "cotacao",

		// Feriado
		"feriados": "feriado",

		// Notícias
		"news":     "noticia",
		"noticias": "noticia",
		"notícia":  "noticia",
		"notícias": "noticia",
		"jornal":   "noticia",

		// Receita
		"receitas": "receita",
		"comida":   "receita",
		"cozinhar": "receita",

		// Piada
		"piadas": "piada",
		"zoeira": "piada",

		// Curiosidade
		"fatos":        "fato",
		"curiosidade":  "fato",
		"curiosidades": "fato",

		// Filme
		"filmes": "filme",
		"movie":  "filme",
		"cinema": "filme",
		"serie":  "filme",
		"série":  "filme",

		// Contagem
		"dias":      "contagem",
		"contador":  "contagem",
		"countdown": "contagem",

		// Unsticker
		"us":               "unsticker",
		"desticker":        "unsticker",
		"tirarfigurinha":   "unsticker",
		"figurinha2imagem": "unsticker",
		"imagem":           "unsticker",

		// Tradução
		"traduzir":  "traduz",
		"translate": "traduz",
		"tradução":  "traduz",
		"traducao":  "traduz",
	}

	for alias, target := range aliases {
		r.RegisterAlias(alias, target)
	}
}
