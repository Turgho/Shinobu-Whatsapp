// Testes do AlbumCoordinator e dos helpers de detecção de album.
// Sem rede: eventos são montados com protobufs sintéticos do whatsmeow.
package commands

import (
	"sync"
	"testing"
	"time"

	"go.mau.fi/whatsmeow/proto/waCommon"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// albumParent monta um evento de primeiro item de album (3 itens, 2 img + 1 vid).
func albumParent(id string, imgs, vids uint32) *events.Message {
	return &events.Message{
		Info: types.MessageInfo{ID: id},
		Message: &waE2E.Message{AlbumMessage: &waE2E.AlbumMessage{
			ExpectedImageCount: proto.Uint32(imgs),
			ExpectedVideoCount: proto.Uint32(vids),
		}},
	}
}

// albumChild monta um evento de item filho apontando pro pai.
func albumChild(parentID string) *events.Message {
	return &events.Message{Message: &waE2E.Message{MessageContextInfo: &waE2E.MessageContextInfo{
		MessageAssociation: &waE2E.MessageAssociation{
			AssociationType:  waE2E.MessageAssociation_MEDIA_ALBUM.Enum(),
			ParentMessageKey: &waCommon.MessageKey{ID: proto.String(parentID)},
		},
	}}}
}

func emptyEvt() *events.Message { return &events.Message{Message: &waE2E.Message{}} }

func TestIsAlbumMessage(t *testing.T) {
	if !IsAlbumMessage(albumParent("p1", 2, 1)) {
		t.Error("album com contagens deveria ser detectado")
	}
	// contagens zeradas não são album
	if IsAlbumMessage(albumParent("p2", 0, 0)) {
		t.Error("album com 0+0 não deveria contar como album")
	}
	if IsAlbumMessage(emptyEvt()) || IsAlbumMessage(nil) {
		t.Error("mensagem vazia/nil não é album")
	}
}

func TestAlbumExpectedCount(t *testing.T) {
	if got := AlbumExpectedCount(albumParent("p1", 2, 1)); got != 3 {
		t.Errorf("expected count = %d, want 3", got)
	}
	if AlbumExpectedCount(emptyEvt()) != 0 || AlbumExpectedCount(nil) != 0 {
		t.Error("sem album count deve ser 0")
	}
}

func TestHasMediaAlbumAssociationAndParent(t *testing.T) {
	child := albumChild("parent-99")
	if !HasMediaAlbumAssociation(child) {
		t.Error("item filho deveria ser detectado")
	}
	if got := ParentMessageID(child); got != "parent-99" {
		t.Errorf("ParentMessageID = %q, want parent-99", got)
	}
	// associação de tipo errado
	wrong := &events.Message{Message: &waE2E.Message{MessageContextInfo: &waE2E.MessageContextInfo{
		MessageAssociation: &waE2E.MessageAssociation{
			AssociationType: waE2E.MessageAssociation_UNKNOWN.Enum(),
		},
	}}}
	if HasMediaAlbumAssociation(wrong) {
		t.Error("associação não-MEDIA_ALBUM não deveria contar")
	}
	if HasMediaAlbumAssociation(emptyEvt()) || ParentMessageID(emptyEvt()) != "" {
		t.Error("mensagem vazia não tem associação")
	}
}

func TestBufferizeCompletesWhenAllArrive(t *testing.T) {
	ac := NewAlbumCoordinator(zap.NewNop())

	if ac.Bufferize("p1", albumParent("p1", 0, 2), "sticker", []string{}, 2) {
		t.Fatal("1/2 não deveria completar")
	}
	if !ac.Bufferize("p1", albumChild("p1"), "", nil, 2) {
		t.Fatal("2/2 deveria sinalizar completo")
	}

	// O Router chama Dispatch quando Bufferize sinaliza true
	var (
		gotCmd  string
		gotArgs []string
		gotLen  int
	)
	ac.Dispatch("p1", func(cmdName string, args []string, items []*events.Message) {
		gotCmd, gotArgs, gotLen = cmdName, args, len(items)
	})
	if gotCmd != "sticker" || gotLen != 2 {
		t.Fatalf("dispatch: cmd=%q items=%d", gotCmd, gotLen)
	}
	_ = gotArgs
}

func TestBufferizeChildBeforeParent(t *testing.T) {
	ac := NewAlbumCoordinator(zap.NewNop())

	// filhos chegam primeiro (sem cmd) e ainda incompletos…
	if ac.Bufferize("p2", albumChild("p2"), "", nil, 2) {
		t.Fatal("1/2 não deveria completar")
	}
	// …depois o pai chega e completa (2/2)
	if !ac.Bufferize("p2", albumParent("p2", 0, 2), "sticker", []string{"arg"}, 2) {
		t.Fatal("pai completando deveria retornar true")
	}

	var (
		gotCmd  string
		gotArgs []string
	)
	ac.Dispatch("p2", func(cmdName string, args []string, _ []*events.Message) {
		gotCmd, gotArgs = cmdName, args
	})
	if gotCmd != "sticker" {
		t.Fatalf("cmd herdado do pai = %q, want sticker", gotCmd)
	}
	if len(gotArgs) != 1 || gotArgs[0] != "arg" {
		t.Fatalf("args herdados = %v", gotArgs)
	}
}

func TestBufferizeOverMaxItems(t *testing.T) {
	ac := NewAlbumCoordinator(zap.NewNop())

	// 31 itens excedem albumMaxItems → não armazena, não completa
	if ac.Bufferize("pmax", albumParent("pmax", 31, 0), "sticker", nil, 31) {
		t.Fatal("acima do limite não deveria completar")
	}
	if _, exists := ac.pending["pmax"]; exists {
		t.Fatal("album acima do limite não deveria ficar pendente")
	}
}

func TestDispatchRemovesPending(t *testing.T) {
	ac := NewAlbumCoordinator(zap.NewNop())

	ac.Bufferize("p3", albumParent("p3", 0, 1), "sticker", nil, 1)

	var calls int
	ac.Dispatch("p3", func(_ string, _ []string, _ []*events.Message) { calls++ })
	if calls != 1 {
		t.Fatalf("dispatch chamado %d vez(es), want 1", calls)
	}

	// segundo Dispatch no mesmo parent é no-op (já removido)
	ac.Dispatch("p3", func(_ string, _ []string, _ []*events.Message) { calls++ })
	if calls != 1 {
		t.Fatalf("double dispatch: %d, want 1", calls)
	}
	if _, exists := ac.pending["p3"]; exists {
		t.Fatal("pending não removido após Dispatch")
	}
}

func TestDispatchTimeoutPartial(t *testing.T) {
	ac := NewAlbumCoordinator(zap.NewNop())

	var (
		mu     sync.Mutex
		gotLen int
	)
	ac.SetBatchHandler(func(_ string, _ []string, items []*events.Message) {
		mu.Lock()
		gotLen = len(items)
		mu.Unlock()
	})

	// 1/2 itens recebidos; timeout dispara dispatch parcial
	ac.Bufferize("p4", albumParent("p4", 0, 2), "sticker", nil, 2)
	ac.dispatchTimeout("p4")

	mu.Lock()
	defer mu.Unlock()
	if gotLen != 1 {
		t.Fatalf("despacho parcial = %d itens, want 1", gotLen)
	}
	if _, exists := ac.pending["p4"]; exists {
		t.Fatal("pending não removido após timeout")
	}

	// segundo timeout é no-op
	ac.dispatchTimeout("p4")
	if gotLen != 1 {
		t.Fatal("double timeout dispatch")
	}
}

func TestDispatchTimeoutDiscardsWithoutCommand(t *testing.T) {
	ac := NewAlbumCoordinator(zap.NewNop())

	// album sem cmdName (filhos soltos) → descartado no timeout
	ac.Bufferize("p5", albumChild("p5"), "", nil, 2)
	ac.dispatchTimeout("p5")

	// batchHandler não deveria disparar (cmd vazio → descarte)
	var calls int
	ac.SetBatchHandler(func(_ string, _ []string, _ []*events.Message) { calls++ })
	if calls != 0 {
		t.Fatal("album sem comando não deveria ser despachado")
	}
	if _, exists := ac.pending["p5"]; exists {
		t.Fatal("album descartado ainda está pendente")
	}
}

func TestBufferizeUnknownExpectedFiresTimerFast(t *testing.T) {
	ac := NewAlbumCoordinator(zap.NewNop())

	// O handler roda em outra goroutine (o timer do album): o contador
	// precisa de sincronização — sem isso o -race acusa data race.
	var mu sync.Mutex
	gotLen := 0
	ac.SetBatchHandler(func(_ string, _ []string, items []*events.Message) {
		mu.Lock()
		gotLen = len(items)
		mu.Unlock()
	})

	// expected=0 (desconhecido) → timer de 3s dispara despacho
	ac.Bufferize("p6", albumChild("p6"), "sticker", nil, 0)

	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		n := gotLen
		mu.Unlock()
		if n != 0 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	mu.Lock()
	final := gotLen
	mu.Unlock()
	if final != 1 {
		t.Fatalf("timeout de album desconhecido não despachou (gotLen=%d)", final)
	}
}
