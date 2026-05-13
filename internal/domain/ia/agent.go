package ia

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/history"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// agentIAMessage estende history.IAMessage com suporte a tool_calls.
type agentIAMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	Name       string     `json:"name,omitempty"`
	ToolCalls  []toolCall `json:"tool_calls,omitempty"`
}

// agentIARequest estende IARequest com suporte a tools.
type agentIARequest struct {
	Model       string           `json:"model"`
	Messages    []agentIAMessage `json:"messages"`
	Tools       []toolSchema     `json:"tools,omitempty"`
	Stream      bool             `json:"stream"`
	Temperature float64          `json:"temperature"`
	MaxTokens   int              `json:"max_tokens"`
}

// agentIAResponse é a resposta do Groq com suporte a tool_calls.
type agentIAResponse struct {
	Choices []struct {
		Message      agentIAMessage `json:"message"`
		FinishReason string         `json:"finish_reason"`
	} `json:"choices"`
}

// agentChat é análogo ao groqChat mas suporta tools e tool_calls na resposta.
func agentChat(ctx context.Context, groqURL, groqKey string, req agentIARequest) (agentIAResponse, error) {
	var zero agentIAResponse

	body, err := json.Marshal(req)
	if err != nil {
		return zero, fmt.Errorf("agent: serializar: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, groqURL, bytes.NewBuffer(body))
	if err != nil {
		return zero, fmt.Errorf("agent: criar request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+groqKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return zero, fmt.Errorf("agent: http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return zero, fmt.Errorf("agent: status %d: %s", resp.StatusCode, string(b))
	}

	var result agentIAResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return zero, fmt.Errorf("agent: decodificar: %w", err)
	}
	if len(result.Choices) == 0 {
		return zero, fmt.Errorf("agent: resposta sem choices")
	}
	return result, nil
}

// AskAgent é o equivalente de AskIA mas com suporte a tool calling.
// Usa o mesmo contexto, histórico e system prompt da Shinobu.
// Se o modelo não chamar nenhuma ferramenta, retorna a resposta direta.
func AskAgent(
	ctx context.Context,
	client *whatsmeow.Client,
	evt *events.Message,
	registry *ToolRegistry,
	chat, prompt string,
	isOwner bool,
	store *history.Store,
) (string, error) {
	groqURL := os.Getenv("GROQ_URL")
	groqKey := os.Getenv("GROQ_API_KEY")

	prompt = cleanPrompt(prompt)
	mode := classifyPromptMode(prompt)

	// Monta mensagens base (mesmo padrão do AskIA).
	base := baseSystemMessages(mode, isOwner)
	base = appendPersistentAndRecent(ctx, base, chat, store)

	// Converte history.IAMessage → agentIAMessage.
	messages := toAgentMessages(base)
	messages = append(messages, agentIAMessage{Role: "user", Content: prompt})

	// Primeira chamada com tools disponíveis.
	params := mainAnswerParams(mode, false)
	resp, err := agentChat(ctx, groqURL, groqKey, agentIARequest{
		Model:       params.model,
		Messages:    messages,
		Tools:       registry.Schemas(),
		Stream:      false,
		Temperature: params.temperature,
		MaxTokens:   params.maxTokens,
	})
	if err != nil {
		return "", fmt.Errorf("agent: primeira chamada: %w", err)
	}

	choice := resp.Choices[0]

	// Sem tool calls → resposta direta, mesmo comportamento do AskIA.
	if len(choice.Message.ToolCalls) == 0 {
		return choice.Message.Content, nil
	}

	// Com tool calls → executa cada ferramenta e coleta resultados.
	messages = append(messages, choice.Message)

	for _, tc := range choice.Message.ToolCalls {
		result, err := registry.Execute(ctx, client, evt, tc.Function.Name, tc.Function.Arguments)
		if err != nil {
			result = fmt.Sprintf("erro: %s", err.Error())
		}
		messages = append(messages, agentIAMessage{
			Role:       "tool",
			ToolCallID: tc.ID,
			Name:       tc.Function.Name,
			Content:    result,
		})
	}

	// Segunda chamada — resposta final em linguagem natural.
	finalResp, err := agentChat(ctx, groqURL, groqKey, agentIARequest{
		Model:       modelWebStrong, // 70b para síntese final
		Messages:    messages,
		Stream:      false,
		Temperature: 0.65,
		MaxTokens:   300,
	})
	if err != nil {
		return "", fmt.Errorf("agent: chamada final: %w", err)
	}

	return finalResp.Choices[0].Message.Content, nil
}

// toAgentMessages converte o slice base (history.IAMessage) para agentIAMessage.
func toAgentMessages(msgs []history.IAMessage) []agentIAMessage {
	out := make([]agentIAMessage, len(msgs))
	for i, m := range msgs {
		out[i] = agentIAMessage{Role: m.Role, Content: m.Content}
	}
	return out
}
