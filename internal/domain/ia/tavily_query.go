package ia

import "strings"

func tavilyTopicFromQuery(query string) string {
	if prefersNewsTavilyTopic(strings.ToLower(query)) {
		return "news"
	}
	return "general"
}

func tavilyDaysFromQuery(query string) int {
	lower := strings.ToLower(query)
	switch {
	case prefersNewsTavilyTopic(lower):
		return 3
	case isPriceQuery(lower):
		return 7
	default:
		return 0
	}
}

// enrichQueryBR complementa queries de preço/produto com contexto brasileiro
// para evitar resultados em dólar ou de outros países.
func enrichQueryBR(query, lowerQuery string) string {
	if isPriceQuery(lowerQuery) {
		return query + " preço brasil reais"
	}
	return query
}

func isPriceQuery(lowerQuery string) bool {
	keywords := []string{"quanto custa", "preço", "preco", "valor", "comprar", "mais barato", "promoção", "promocao"}
	for _, k := range keywords {
		if strings.Contains(lowerQuery, k) {
			return true
		}
	}
	return false
}
