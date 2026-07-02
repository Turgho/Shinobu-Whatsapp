package whatsapp

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/image/draw"
	"google.golang.org/protobuf/proto"
)

const thumbSize = 72

// SendImage envia imagem com legenda opcional e miniatura borrada (JPEGThumbnail).
func SendImage(
	ctx context.Context,
	client *whatsmeow.Client,
	evt *events.Message,
	uploaded *whatsmeow.UploadResponse,
	rawData []byte,
	caption string,
	reply bool,
) error {
	thumb := generateThumbnail(rawData)

	return withTyping(ctx, client, evt, types.ChatPresenceMediaText, func() error {
		msg := &waE2E.Message{
			ImageMessage: &waE2E.ImageMessage{
				URL:           proto.String(uploaded.URL),
				DirectPath:    proto.String(uploaded.DirectPath),
				MediaKey:      uploaded.MediaKey,
				Mimetype:      proto.String("image/jpeg"),
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    proto.Uint64(uploaded.FileLength),
				Caption:       proto.String(caption),
				JPEGThumbnail: thumb,
				ContextInfo:   buildContext(evt, reply, nil),
			},
		}
		_, err := client.SendMessage(ctx, evt.Info.Chat, msg)
		if err != nil {
			return fmt.Errorf("whatsapp: send image: %w", err)
		}
		return nil
	})
}

// SendImageBytes faz upload e envia imagem PNG a partir de bytes, com legenda.
// Conveniência para quando a imagem já está em memória (ex: cards gerados).
func SendImageBytes(ctx context.Context, client *whatsmeow.Client, evt *events.Message, data []byte, caption string) error {
	uploaded, err := client.Upload(ctx, data, whatsmeow.MediaImage)
	if err != nil {
		return fmt.Errorf("whatsapp: upload image: %w", err)
	}
	return SendImage(ctx, client, evt, &uploaded, data, caption, true)
}

func generateThumbnail(data []byte) []byte {
	if len(data) == 0 {
		return nil
	}

	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil
	}

	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w > h {
		h = h * thumbSize / w
		w = thumbSize
	} else {
		w = w * thumbSize / h
		h = thumbSize
	}
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.BiLinear.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 50}); err != nil {
		return nil
	}
	return buf.Bytes()
}
