package ia

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Turgho/Shinobu-Whatsapp/internal/commands"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// Tool representa uma ferramenta disponível para a IA chamar.
// Cada tool corresponde a um comando registrado no router.
type Tool struct {
	Meta    commands.CommandMeta
	Handler commands.HandlerFunc
}

// toolSchema é o formato JSON que o Groq espera para definir uma ferramenta.
type toolSchema struct {
	Type     string       `json:"type"` // sempre "function"
	Function toolFunction `json:"function"`
}

type toolFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  toolParameters `json:"parameters"`
}

type toolParameters struct {
	Type       string              `json:"type"` // sempre "object"
	Properties map[string]toolProp `json:"properties"`
	Required   []string            `json:"required,omitempty"`
}

type toolProp struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

// ToolRegistry mantém os comandos que a IA pode chamar.
type ToolRegistry struct {
	tools map[string]Tool
}

// NewToolRegistry cria um registry vazio.
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{tools: make(map[string]Tool)}
}

// Register adiciona um comando ao registry.
// Apenas comandos não-privados são registrados por padrão.
func (r *ToolRegistry) Register(meta commands.CommandMeta, handler commands.HandlerFunc) {
	if meta.Private {
		return // não expõe comandos admin para a IA
	}
	r.tools[meta.Name] = Tool{Meta: meta, Handler: handler}
}

// RegisterFromRouter popula o registry a partir de um router existente.
func (r *ToolRegistry) RegisterFromRouter(router *commands.Router) {
	for _, meta := range router.Commands() {
		if handler := router.Handler(meta.Name); handler != nil {
			r.Register(meta, handler)
		}
	}
}

// Schemas retorna os schemas JSON para enviar ao Groq.
func (r *ToolRegistry) Schemas() []toolSchema {
	schemas := make([]toolSchema, 0, len(r.tools))
	for _, t := range r.tools {
		schemas = append(schemas, buildSchema(t.Meta))
	}
	return schemas
}

// Execute chama o handler de um comando com os args extraídos pela IA.
func (r *ToolRegistry) Execute(
	ctx context.Context,
	client *whatsmeow.Client,
	evt *events.Message,
	name string,
	argsJSON string,
) (string, error) {
	t, ok := r.tools[name]
	if !ok {
		return "", fmt.Errorf("ferramenta %q não encontrada", name)
	}

	// Extrai os args do JSON que o modelo retorna.
	args, err := parseToolArgs(t.Meta, argsJSON)
	if err != nil {
		return "", fmt.Errorf("args inválidos para %q: %w", name, err)
	}

	fmt.Printf("[ia/agent] executando ferramenta: %s args=%v\n", name, args)

	if err := t.Handler(ctx, client, evt, args); err != nil {
		return "", fmt.Errorf("erro ao executar %q: %w", name, err)
	}

	return fmt.Sprintf("Comando `!%s` executado com sucesso.", name), nil
}

// buildSchema converte um CommandMeta em toolSchema para o Groq.
func buildSchema(meta commands.CommandMeta) toolSchema {
	props := make(map[string]toolProp)
	required := []string{}

	for _, arg := range meta.Args {
		props[arg.Name] = toolProp{
			Type:        "string",
			Description: arg.Name,
		}
		if arg.Required {
			required = append(required, arg.Name)
		}
	}

	// Comandos sem args ainda precisam de um objeto de parâmetros válido.
	if len(props) == 0 {
		props["_noop"] = toolProp{Type: "string", Description: "ignorar"}
	}

	return toolSchema{
		Type: "function",
		Function: toolFunction{
			Name:        meta.Name,
			Description: meta.Description,
			Parameters: toolParameters{
				Type:       "object",
				Properties: props,
				Required:   required,
			},
		},
	}
}

// parseToolArgs converte o JSON de args retornado pelo modelo em []string.
func parseToolArgs(meta commands.CommandMeta, argsJSON string) ([]string, error) {
	var raw map[string]string
	if err := json.Unmarshal([]byte(argsJSON), &raw); err != nil {
		return nil, err
	}

	args := make([]string, 0, len(meta.Args))
	for _, arg := range meta.Args {
		if v, ok := raw[arg.Name]; ok && v != "" {
			args = append(args, v)
		}
	}
	return args, nil
}
