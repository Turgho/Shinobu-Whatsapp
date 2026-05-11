package ia

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	tavilyURL       = "https://api.tavily.com/search"
	maxContentChars = 900 // por resultado — suficiente sem explodir o contexto
	maxResults      = 6   // mais fontes = resposta mais embasada
)

// tavilyRequest é o payload enviado à API do Tavily.
type tavilyRequest struct {
	Query             string `json:"query"`
	APIKey            string `json:"api_key"`
	SearchDepth       string `json:"search_depth"` // "basic" ou "advanced"
	Topic             string `json:"topic"`        // "general" ou "news"
	MaxResults        int    `json:"max_results"`
	IncludeAnswer     bool   `json:"include_answer"`      // resumo automático do Tavily
	IncludeRawContent bool   `json:"include_raw_content"` // conteúdo completo das páginas
}

type tavilyResponse struct {
	Answer  string `json:"answer"`
	Results []struct {
		Title   string  `json:"title"`
		Content string  `json:"content"`
		URL     string  `json:"url"`
		Score   float64 `json:"score"` // relevância (0-1)
	} `json:"results"`
}

// searchWeb busca informações na web via Tavily e retorna um contexto
// formatado para ser incluído no prompt da IA.
func searchWeb(ctx context.Context, query string) (string, error) {
	tavilyKey := os.Getenv("TAVILY_API_KEY")
	if tavilyKey == "" {
		return "", fmt.Errorf("TAVILY_API_KEY não configurada")
	}

	req := tavilyRequest{
		Query:             query,
		APIKey:            tavilyKey,
		SearchDepth:       "advanced", // mais preciso que "basic"
		Topic:             detectTopic(query),
		MaxResults:        maxResults,
		IncludeAnswer:     true,
		IncludeRawContent: true,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("tavily: marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, tavilyURL, bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("tavily: request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("tavily: http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("tavily: status %d", resp.StatusCode)
	}

	var result tavilyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("tavily: decode: %w", err)
	}

	return formatResults(result), nil
}

// formatResults monta o contexto que vai para o prompt da IA.
// Prioriza o resumo automático do Tavily; usa os resultados individuais como complemento.
func formatResults(result tavilyResponse) string {
	var sb strings.Builder

	// Resumo automático do Tavily — geralmente já é suficiente.
	if result.Answer != "" {
		sb.WriteString("Resumo: ")
		sb.WriteString(result.Answer)
		sb.WriteString("\n\n")
	}

	// Resultados individuais filtrados por relevância mínima.
	sb.WriteString("Fontes:\n")
	for i, r := range result.Results {
		if r.Score < 0.3 { // descarta resultados pouco relevantes
			continue
		}

		content := r.Content
		if utf8.RuneCountInString(content) > maxContentChars {
			// Trunca respeitando caracteres multibyte (emojis, acentos).
			runes := []rune(content)
			content = string(runes[:maxContentChars]) + "…"
		}

		sb.WriteString(fmt.Sprintf("[%d] %s\n%s\n(%s)\n\n", i+1, r.Title, content, r.URL))
	}

	return strings.TrimSpace(sb.String())
}

// detectTopic escolhe o índice de busca do Tavily com base na query.
// Queries sobre eventos recentes usam "news" para dados mais frescos.
func detectTopic(query string) string {
	newsKeywords := []string{
		"hoje", "agora", "notícia", "news", "aconteceu",
		"último", "recente", "atualidade", "breaking",
	}
	q := strings.ToLower(query)
	for _, kw := range newsKeywords {
		if strings.Contains(q, kw) {
			return "news"
		}
	}
	return "general"
}
