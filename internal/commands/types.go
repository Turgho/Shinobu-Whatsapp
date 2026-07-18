package commands

import (
	"context"
	"errors"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// ErrNotACommand indica que o handler identificou que não é o comando certo
// para a mensagem. O pipeline de NLU deve cair de volta para IA geral.
var ErrNotACommand = errors.New("handler: not the right command for this input")

// HandlerFunc é a assinatura de um handler de comando.
// O contexto vem do Router (timeouts); args exclui o nome do comando.
type HandlerFunc func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error

// BatchHandlerFunc é a assinatura de um handler que processa múltiplos eventos
// de album (imagens/vídeos agrupados). Usado pelo AlbumCoordinator quando o
// comando suporta processamento em lote (ex: !sticker com album).
type BatchHandlerFunc func(ctx context.Context, client *whatsmeow.Client, items []*events.Message, args []string) error

// Middleware decide se o fluxo segue para o handler.
// O parâmetro cmd é o nome lógico do comando ("shinobu" no atalho por menção, ou o token após o prefixo).
// Retorna true para continuar, false para bloquear sem executar o handler.
type Middleware func(cmd string, evt *events.Message) bool

// ArgMeta descreve um argumento de um comando para exibição no menu.
type ArgMeta struct {
	Name     string // nome do argumento, ex: "cidade"
	Required bool   // true = obrigatório (<cidade>), false = opcional ([cidade])
}

// CommandType agrupa comandos para exibição e filtros no menu.
type CommandType string

const (
	CommandTypeUtility  CommandType = "utility"
	CommandTypeFun      CommandType = "fun"
	CommandTypeAI       CommandType = "ai"
	CommandTypeMedia    CommandType = "media"
	CommandTypeDownload CommandType = "download"
	CommandTypeAdmin    CommandType = "admin"
	CommandTypeOwner    CommandType = "owner"
	CommandTypeGroup    CommandType = "group"
	CommandTypeNSFW     CommandType = "nsfw"
)

// CommandMeta contém os metadados de um comando.
// É usado para registro, validação de permissões e geração automática do menu.
type CommandMeta struct {
	Name        string      // chave do comando, ex: "clima"
	Description string      // descrição exibida no menu
	Type        CommandType // categoria do comando
	Args        []ArgMeta   // argumentos esperados, em ordem
	Private     bool        // se true, apenas owner/admins podem executar
}

// command acopla metadados públicos ao HandlerFunc registrado no Router.
type command struct {
	Meta         CommandMeta
	Handler      HandlerFunc
	BatchHandler BatchHandlerFunc // opcional: handler para albums (múltiplos itens)
}
