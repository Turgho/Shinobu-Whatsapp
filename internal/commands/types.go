package commands

import (
	"context"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// HandlerFunc é a assinatura de um handler de comando.
// Recebe contexto para permitir cancelamento e timeouts.
type HandlerFunc func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error

// Middleware decide se um comando deve ser executado.
// Retorna true para continuar, false para bloquear.
type Middleware func(cmd string, evt *events.Message) bool

// ArgMeta descreve um argumento de um comando para exibição no menu.
type ArgMeta struct {
	Name     string // nome do argumento, ex: "cidade"
	Required bool   // true = obrigatório (<cidade>), false = opcional ([cidade])
}

// CommandMeta contém os metadados de um comando.
// É usado para registro, validação de permissões e geração automática do menu.
type CommandMeta struct {
	Name        string    // chave do comando, ex: "weather"
	Description string    // descrição exibida no menu
	Args        []ArgMeta // argumentos esperados, em ordem
	Private     bool      // se true, apenas owner/admins podem executar
}

// command é o tipo interno que junta meta + handler.
type command struct {
	Meta    CommandMeta
	Handler HandlerFunc
}
