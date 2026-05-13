package ia

import "strings"

// Palavras-chave para decidir busca web **sem** segunda chamada ao Groq (economia de tokens).
// Mantidas em um único lugar: também orientam o tópico "news" no Tavily quando aplicável.
var webSearchCueKeywords = []string{
	// Tempo / atualidade
	"hoje", "agora", "ontem", "amanhã", "amanha", "essa semana", "esse mês", "esse ano",
	"atual", "atualmente", "recente", "recentemente", "últimas horas", "ultimas horas",

	// Notícias e eventos
	"notícia", "noticia", "novidade", "aconteceu", "acontecendo",
	"anunciou", "anunciaram", "confirmou", "cancelou", "adiou",

	// Preços e mercado
	"preço", "preco", "valor", "cotação", "cotacao", "dólar", "dolar", "euro", "bitcoin",
	"criptomoeda", "bolsa", "inflação", "inflacao",

	// Clima
	"clima", "tempo", "chuva", "temperatura", "previsão", "previsao",

	// Esportes
	"placar", "resultado", "ganhou", "perdeu", "classificação", "classificacao",
	"campeonato", "copa", "jogo de hoje", "rodada",

	// Entretenimento / lançamentos
	"lançou", "lancou", "lançamento", "lancamento", "estreou", "estreia",
	"temporada", "episódio", "episodio", "album", "álbum", "música nova", "musica nova",

	// Busca explícita
	"pesquisa", "pesquise", "busca", "busque", "procura", "procure",
	"me fala sobre", "me conta sobre", "o que é", "o que foi", "quem é", "quem foi",
}

// newsTavilyKeywords é subconjunto orientado a fatos que envelhecem rápido (Tavily topic=news).
var newsTavilyKeywords = []string{
	"hoje", "agora", "notícia", "noticia", "news", "aconteceu",
	"último", "ultimo", "recente", "atualidade", "breaking",
}

// hasWebSearchCue retorna true se o texto sugere fortemente dados externos atuais.
func hasWebSearchCue(lower string) bool {
	for _, kw := range webSearchCueKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// prefersNewsTavilyTopic retorna true quando convém usar topic=news na API Tavily.
func prefersNewsTavilyTopic(lower string) bool {
	for _, kw := range newsTavilyKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}
