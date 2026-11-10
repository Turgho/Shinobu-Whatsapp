package media

import (
	"testing"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"
)

// A checagem de tamanho antes do download evita que uma mídia grande
// ocupe a RAM inteira do container (limite de 512MB) só para ser
// descartada depois.
func TestFileLength(t *testing.T) {
	img := &waE2E.ImageMessage{
		FileLength: proto.Uint64(1234),
	}
	size, ok := fileLength(img)
	if !ok {
		t.Fatal("ImageMessage deveria expor o tamanho")
	}
	if size != 1234 {
		t.Errorf("tamanho errado: %d", size)
	}

	// StickerMessage também tem FileLength no protobuf (o valor 0 só
	// significa "tamanho não informado").
	size, ok = fileLength(&waE2E.StickerMessage{})
	if !ok {
		t.Error("StickerMessage também expõe FileLength")
	}
	if size != 0 {
		t.Errorf("tamanho inesperado: %d", size)
	}
}
