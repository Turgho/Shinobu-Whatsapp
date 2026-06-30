package ia

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	tavilyURL       = "https://api.tavily.com/search"
	maxContentChars = 900
	maxResults      = 6
)

type tavilyRequest struct {
	Query             string   `json:"query"`
	APIKey            string   `json:"api_key"`
	SearchDepth       string   `json:"search_depth"`
	Topic             string   `json:"topic"`
	MaxResults        int      `json:"max_results"`
	Days              int      `json:"days,omitempty"`
	IncludeAnswer     bool     `json:"include_answer"`
	IncludeRawContent bool     `json:"include_raw_content"`
	Country           string   `json:"country"`
	IncludeDomains    []string `json:"include_domains,omitempty"`
}

type tavilyResponse struct {
	Answer  string `json:"answer"`
	Results []struct {
		Title   string  `json:"title"`
		Content string  `json:"content"`
		URL     string  `json:"url"`
		Score   float64 `json:"score"`
	} `json:"results"`
}

func searchWeb(ctx context.Context, apiKey, query string) (string, error) {
	if apiKey == "" {
		return "", fmt.Errorf("TAVILY_API_KEY não configurada")
	}

	lowerQuery := strings.ToLower(query)

	req := tavilyRequest{
		Query:             enrichQueryBR(query, lowerQuery),
		APIKey:            apiKey,
		SearchDepth:       "advanced",
		Topic:             tavilyTopicFromQuery(query),
		MaxResults:        maxResults,
		Days:              tavilyDaysFromQuery(query),
		IncludeAnswer:     true,
		IncludeRawContent: true,
		Country:           "brazil",
	}

	if isPriceQuery(lowerQuery) {
		req.IncludeDomains = []string{
			"mercadolivre.com.br",
			"amazon.com.br",
			"buscape.com.br",
			"zoom.com.br",
			"kabum.com.br",
		}
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

// formatResults monta o contexto web como bloco de texto: resumo + fontes ranqueadas.
// Filtra resultados com score < 0.3 e trunca conteúdo longo para 900 caracteres.
func formatResults(result tavilyResponse) string {
	var sb strings.Builder

	if result.Answer != "" {
		sb.WriteString("Resumo: ")
		sb.WriteString(result.Answer)
		sb.WriteString("\n\n")
	}

	sb.WriteString("Fontes:\n")
	for i, r := range result.Results {
		// Filtra resultados com score baixo (Tavily retorna de 0 a 1).
		// 0.3 é um threshold empírico que elimina ruído sem perder fontes boas.
		if r.Score < 0.3 {
			continue
		}

		content := r.Content
		if utf8.RuneCountInString(content) > maxContentChars {
			runes := []rune(content)
			content = string(runes[:maxContentChars]) + "…"
		}

		sb.WriteString(fmt.Sprintf("[%d] %s\n%s\n(%s)\n\n", i+1, r.Title, content, r.URL))
	}

	return strings.TrimSpace(sb.String())
}


