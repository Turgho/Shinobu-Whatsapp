package commands

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/ignore"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// newTestRouter cria um router mínimo (sem client) com um comando de teste.
func newRouterWithCommand(t *testing.T) (*Router, *bool) {
	t.Helper()
	r := NewRouter("!", nil, zap.NewNop(), nil, ignore.NewStore())
	chamou := false
	r.RegisterCommand(CommandMeta{
		Name:        "clima",
		Description: "clima",
		Type:        CommandTypeUtility,
	}, func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		chamou = true
		return nil
	})
	return r, &chamou
}

// mensagem de teste "!clima"
func testMessage() *events.Message {
	return &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{
				Chat:   types.JID{User: "5511999999999", Server: "s.whatsapp.net"},
				Sender: types.JID{User: "5511888888888", Server: "s.whatsapp.net"},
			},
			Timestamp: time.Now(),
		},
		Message: &waE2E.Message{Conversation: proto.String("!clima")},
	}
}

// Comando habilitado: o handler roda.
func TestComandoHabilitadoExecuta(t *testing.T) {
	r, chamou := newRouterWithCommand(t)
	r.SetCommandState(NewCommandStateAt(filepath.Join(t.TempDir(), "state.json")))

	r.HandleMessage(testMessage())

	if !*chamou {
		t.Fatal("comando habilitado deveria executar")
	}
}

// Comando desabilitado: o handler NÃO roda (o bot continua vivo).
func TestComandoDesabilitadoNaoExecuta(t *testing.T) {
	r, chamou := newRouterWithCommand(t)
	r.SetCommandState(NewCommandStateAt(filepath.Join(t.TempDir(), "state.json")))

	if err := r.SetCommandDisabled("clima", true); err != nil {
		t.Fatalf("SetCommandDisabled falhou: %v", err)
	}

	r.HandleMessage(testMessage())

	if *chamou {
		t.Fatal("comando desabilitado não deveria executar")
	}
}
