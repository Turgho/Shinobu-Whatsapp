package commands

import (
	"sync"
	"time"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
	"go.uber.org/zap"
)

const (
	albumBufferTimeout = 10 * time.Second
	albumMaxItems      = 30
)

// pendingAlbum rastreia os itens de um album sendo coletados.
type pendingAlbum struct {
	cmdName    string
	args       []string
	items      []*events.Message
	expected   int
	timer      *time.Timer
	dispatched bool
}

// AlbumCoordinator bufferiza itens de album (múltiplas imagens/vídeos enviados
// juntos no WhatsApp) e despacha o comando batch quando todos chegam.
//
// O fluxo: a primeira mensagem do album tem AlbumMessage (com contagens).
// As subsequentes têm MessageAssociation apontando pro pai.
// O coordinator aguarda até todos chegarem ou timeout, e despacha pro BatchHandlerFunc.
type AlbumCoordinator struct {
	mu           sync.Mutex
	pending      map[string]*pendingAlbum // parentMsgID → album
	log          *zap.Logger
	batchHandler func(cmdName string, args []string, items []*events.Message)
}

// NewAlbumCoordinator cria um coordinator para buffering de albums.
func NewAlbumCoordinator(log *zap.Logger) *AlbumCoordinator {
	return &AlbumCoordinator{
		pending: make(map[string]*pendingAlbum),
		log:     log.Named("album"),
	}
}

// IsAlbumMessage retorna true se a mensagem é o primeiro item de um album
// (contém AlbumMessage no protobuf com contagens > 0).
func IsAlbumMessage(evt *events.Message) bool {
	if evt == nil || evt.Message == nil {
		return false
	}
	album := evt.Message.GetAlbumMessage()
	if album == nil {
		return false
	}
	imgCount := int(album.GetExpectedImageCount())
	vidCount := int(album.GetExpectedVideoCount())
	return imgCount+vidCount > 0
}

// AlbumExpectedCount retorna o total de itens esperados no album.
func AlbumExpectedCount(evt *events.Message) int {
	if evt == nil || evt.Message == nil {
		return 0
	}
	album := evt.Message.GetAlbumMessage()
	if album == nil {
		return 0
	}
	return int(album.GetExpectedImageCount()) + int(album.GetExpectedVideoCount())
}

// HasMediaAlbumAssociation retorna true se a mensagem é um item filho de album
// (contém MessageAssociation do tipo MEDIA_ALBUM).
func HasMediaAlbumAssociation(evt *events.Message) bool {
	if evt == nil || evt.Message == nil {
		return false
	}
	mci := evt.Message.GetMessageContextInfo()
	if mci == nil {
		return false
	}
	assoc := mci.GetMessageAssociation()
	if assoc == nil {
		return false
	}
	return assoc.GetAssociationType() == waE2E.MessageAssociation_MEDIA_ALBUM
}

// ParentMessageID retorna o ID da mensagem pai a partir da associação do album.
func ParentMessageID(evt *events.Message) string {
	if evt == nil || evt.Message == nil {
		return ""
	}
	mci := evt.Message.GetMessageContextInfo()
	if mci == nil {
		return ""
	}
	assoc := mci.GetMessageAssociation()
	if assoc == nil {
		return ""
	}
	key := assoc.GetParentMessageKey()
	if key == nil {
		return ""
	}
	return key.GetID()
}

// Bufferize adiciona um item ao album pendente.
// Retorna true quando todos os itens esperados foram coletados.
func (ac *AlbumCoordinator) Bufferize(parentID string, evt *events.Message, cmdName string, args []string, expected int) bool {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	pa, exists := ac.pending[parentID]
	if !exists {
		if expected > albumMaxItems {
			ac.log.Warn("album: itens excedem limite",
				zap.String("parentID", parentID),
				zap.Int("expected", expected),
			)
			return false
		}
		pa = &pendingAlbum{
			cmdName:  cmdName,
			args:     args,
			items:    make([]*events.Message, 0, expected),
			expected: expected,
		}
		ac.pending[parentID] = pa

		pa.timer = time.AfterFunc(albumBufferTimeout, func() {
			ac.dispatchTimeout(parentID)
		})

		ac.log.Debug("album: buffering iniciado",
			zap.String("parentID", parentID),
			zap.String("cmd", cmdName),
			zap.Int("expected", expected),
		)
	}

	pa.items = append(pa.items, evt)

	ac.log.Debug("album: item adicionado",
		zap.String("parentID", parentID),
		zap.Int("current", len(pa.items)),
		zap.Int("expected", pa.expected),
	)

	return len(pa.items) >= pa.expected
}

// dispatchTimeout é chamado pelo timer quando o album não completou a tempo.
func (ac *AlbumCoordinator) dispatchTimeout(parentID string) {
	ac.mu.Lock()

	pa, exists := ac.pending[parentID]
	if !exists || pa.dispatched {
		ac.mu.Unlock()
		return
	}

	ac.log.Warn("album: timeout — despachando itens parciais",
		zap.String("parentID", parentID),
		zap.Int("received", len(pa.items)),
		zap.Int("expected", pa.expected),
	)

	pa.dispatched = true
	pa.timer.Stop()
	items := pa.items
	cmdName := pa.cmdName
	args := pa.args
	handler := ac.batchHandler
	delete(ac.pending, parentID)
	ac.mu.Unlock()

	if handler != nil {
		handler(cmdName, args, items)
	}
}

// Dispatch dispara o handler batch imediatamente com os itens coletados.
// Chamado pelo Router quando todos os itens chegaram.
func (ac *AlbumCoordinator) Dispatch(parentID string, handler func(cmdName string, args []string, items []*events.Message)) {
	ac.mu.Lock()

	pa, exists := ac.pending[parentID]
	if !exists || pa.dispatched {
		ac.mu.Unlock()
		return
	}

	pa.dispatched = true
	pa.timer.Stop()
	items := pa.items
	cmdName := pa.cmdName
	args := pa.args
	delete(ac.pending, parentID)
	ac.mu.Unlock()

	handler(cmdName, args, items)
}

// SetBatchHandler define a função de callback para despachar albums prontos.
func (ac *AlbumCoordinator) SetBatchHandler(handler func(cmdName string, args []string, items []*events.Message)) {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	ac.batchHandler = handler
}
