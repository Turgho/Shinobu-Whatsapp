package ia

// toolCall representa uma chamada de ferramenta retornada pelo modelo.
type toolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"` // sempre "function"
	Function toolCallFunc `json:"function"`
}

// toolCallFunc contém o nome da ferramenta e os argumentos em JSON.
type toolCallFunc struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string com os args
}
