package sticker

import (
	"context"
	"fmt"

	"github.com/Turgho/YuukoWhatsapp/internal/utils"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// Send envia um sticker salvo pelo nome.
// reply=true → cita a mensagem original.
func Send(
	ctx context.Context,
	client *whatsmeow.Client,
	evt *events.Message,
	name string,
	reply bool,
) error {
	d, ok := Get(name)
	if !ok {
		return fmt.Errorf("sticker %q não encontrado", name)
	}

	uploaded := dataToUpload(d)
	return utils.SendSticker(ctx, client, evt, uploaded, d.IsAnimated, reply)
}

// dataToUpload converte um Data salvo em whatsmeow.UploadResponse
// para ser passado ao utils.SendSticker sem precisar de novo upload.
func dataToUpload(d Data) *whatsmeow.UploadResponse {
	return &whatsmeow.UploadResponse{
		URL:           d.URL,
		DirectPath:    d.DirectPath,
		MediaKey:      d.MediaKey,
		FileEncSHA256: d.FileEncSHA256,
		FileSHA256:    d.FileSHA256,
		FileLength:    d.FileLength,
	}
}
