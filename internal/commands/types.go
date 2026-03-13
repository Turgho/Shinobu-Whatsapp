package commands

import (
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// Command é uma função que recebe um evento de mensagem e uma lista de argumentos
type Command func(client *whatsmeow.Client, evt *events.Message, args []string) error

type Middleware func(cmd string, evt *events.Message) bool
