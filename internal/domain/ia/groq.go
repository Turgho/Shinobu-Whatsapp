package ia

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/httpclient"
)

func groqChat(ctx context.Context, groqURL, groqKey string, req IARequest) (IAResponse, error) {
	var zero IAResponse
	if groqURL == "" {
		return zero, fmt.Errorf("GROQ_URL não definido")
	}
	if groqKey == "" {
		return zero, fmt.Errorf("GROQ_API_KEY não definido")
	}

	body, err := json.Marshal(req)
	if err != nil {
		return zero, fmt.Errorf("groq: serializar request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, groqURL, bytes.NewBuffer(body))
	if err != nil {
		return zero, fmt.Errorf("groq: criar request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+groqKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpclient.Client.Do(httpReq)
	if err != nil {
		return zero, fmt.Errorf("groq: http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return zero, fmt.Errorf("groq: status %d: %s", resp.StatusCode, string(b))
	}

	var iaResp IAResponse
	if err := json.NewDecoder(resp.Body).Decode(&iaResp); err != nil {
		return zero, fmt.Errorf("groq: decodificar: %w", err)
	}
	if len(iaResp.Choices) == 0 {
		return zero, fmt.Errorf("groq: resposta sem choices")
	}
	return iaResp, nil
}
