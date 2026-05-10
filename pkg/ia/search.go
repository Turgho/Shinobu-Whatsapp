package ia

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/Turgho/YuukoWhatsapp/pkg/history"
)

// shouldSearch decide se a pergunta precisa de busca web.
// Primeiro verifica keywords óbvias para evitar chamada extra ao modelo.
// Se não encaixar em nenhuma keyword, usa o Scout para classificar.
func shouldSearch(ctx context.Context, prompt string) bool {
	lower := strings.ToLower(prompt)

	// Keywords que indicam necessidade de informação atual sem precisar chamar o modelo
	keywords := []string{
		// Tempo/data
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

		// Entretenimento/lançamentos
		"lançou", "lancou", "lançamento", "lancamento", "estreou", "estreia",
		"temporada", "episódio", "episodio", "album", "álbum", "música nova", "musica nova",

		// Busca explícita
		"pesquisa", "pesquise", "busca", "busque", "procura", "procure",
		"me fala sobre", "me conta sobre", "o que é", "o que foi", "quem é", "quem foi",
	}
	for _, kw := range keywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}

	// Sem keyword óbvia — chama o Scout para classificar
	return shouldSearchLLM(ctx, prompt)
}

// shouldSearchLLM faz uma chamada leve ao Scout para decidir se a pergunta
// precisa de informações atuais da internet. Retorna true se sim.
// MaxTokens=5 e Temperature=0 garantem resposta mínima e determinística.
func shouldSearchLLM(ctx context.Context, prompt string) bool {
	groqURL := os.Getenv("GROQ_URL")
	groqKey := os.Getenv("GROQ_API_KEY")

	body, err := json.Marshal(IARequest{
		Model: "meta-llama/llama-4-scout-17b-16e-instruct",
		Messages: []history.IAMessage{
			{
				Role: "system",
				Content: "Responda apenas com 'sim' ou 'nao'. Sem pontuação, sem explicação. " +
					"Responda 'sim' somente se a pergunta depender de informação atual, externa, específica da internet, preços, notícias, clima, esportes, lançamentos ou dados que podem estar desatualizados. " +
					"Caso contrário, responda 'nao'.",
			},
			{Role: "user", Content: prompt},
		},
		Stream:      false,
		Temperature: 0, // Determinístico: sem criatividade na classificação
		MaxTokens:   5, // Só precisa de "sim" ou "nao"
	})
	if err != nil {
		return false
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, groqURL, bytes.NewBuffer(body))
	if err != nil {
		return false
	}
	req.Header.Set("Authorization", "Bearer "+groqKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	var iaResp IAResponse
	if err := json.NewDecoder(resp.Body).Decode(&iaResp); err != nil {
		return false
	}

	if len(iaResp.Choices) == 0 {
		return false
	}

	answer := strings.ToLower(strings.TrimSpace(iaResp.Choices[0].Message.Content))
	return strings.HasPrefix(answer, "sim")
}
