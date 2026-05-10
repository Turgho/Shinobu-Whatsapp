package ia

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

const tavilyURL = "https://api.tavily.com/search"

type tavilyRequest struct {
	Query         string `json:"query"`
	APIKey        string `json:"api_key"`
	SearchDepth   string `json:"search_depth"` // "basic" ou "advanced"
	MaxResults    int    `json:"max_results"`
	IncludeAnswer bool   `json:"include_answer"` // resumo automático
}

type tavilyResponse struct {
	Answer  string `json:"answer"`
	Results []struct {
		Content string `json:"content"`
		URL     string `json:"url"`
	} `json:"results"`
}

func searchWeb(ctx context.Context, query string) (string, error) {
	tavilyKey := os.Getenv("TAVILY_API_KEY")

	body, err := json.Marshal(tavilyRequest{
		Query:         query,
		APIKey:        tavilyKey,
		SearchDepth:   "basic",
		MaxResults:    3,
		IncludeAnswer: true,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tavilyURL, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("tavily retornou status %d", resp.StatusCode)
	}

	var result tavilyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	// Prioriza o resumo automático do Tavily
	if result.Answer != "" {
		return result.Answer, nil
	}

	// Fallback: junta os conteúdos dos resultados
	var parts []string
	for _, r := range result.Results {
		content := r.Content
		if len(content) > 300 {
			content = content[:300] // evita contexto gigante
		}
		parts = append(parts, content)
	}
	return strings.Join(parts, "\n\n"), nil
}
