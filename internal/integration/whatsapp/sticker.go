package whatsapp

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

// audioStickerOpus é o mimetype do som dentro de um audio sticker.
const audioStickerOpus = "audio/ogg; codecs=opus"

// stickerSide é o lado do WebP gerado pelo sticker.ConvertToWebP (512×512).
const stickerSide = 512

// SendSticker envia figurinha WebP estática ou animada já uploaded.
func SendSticker(
	ctx context.Context,
	client *whatsmeow.Client,
	evt *events.Message,
	uploaded *whatsmeow.UploadResponse,
	animated bool,
	reply bool,
) error {
	return withTyping(ctx, client, evt, types.ChatPresenceMediaText, func() error {
		msg := &waE2E.Message{
			StickerMessage: &waE2E.StickerMessage{
				URL:           proto.String(uploaded.URL),
				DirectPath:    proto.String(uploaded.DirectPath),
				MediaKey:      uploaded.MediaKey,
				Mimetype:      proto.String("image/webp"),
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    proto.Uint64(uploaded.FileLength),
				IsAnimated:    proto.Bool(animated),
				ContextInfo:   buildContext(evt, reply, nil),
			},
		}
		_, err := client.SendMessage(ctx, evt.Info.Chat, msg)
		if err != nil {
			return fmt.Errorf("whatsapp: send sticker: %w", err)
		}
		return nil
	})
}

// SendAudioSticker envia uma figurinha com som (audio sticker) já uploaded.
//
// Formato do WhatsApp: um StickerMessage comum (imagem WebP) que carrega,
// dentro do oneof "audio", um AudioMessage com som próprio (chave de mídia
// própria) — e o conjunto vai embrulhado em AudioStickerMessage (campo 134
// da mensagem). O MessageContextInfo fica FORA do embrulho: ele pertence à
// mensagem, não à figurinha.
func SendAudioSticker(
	ctx context.Context,
	client *whatsmeow.Client,
	evt *events.Message,
	image, sound *whatsmeow.UploadResponse,
	animated bool,
	seconds uint32,
) error {
	return withTyping(ctx, client, evt, types.ChatPresenceMediaText, func() error {
		now := time.Now()
		msg := &waE2E.Message{
			AudioStickerMessage: &waE2E.FutureProofMessage{
				Message: &waE2E.Message{
					StickerMessage: &waE2E.StickerMessage{
						URL:               proto.String(image.URL),
						DirectPath:        proto.String(image.DirectPath),
						MediaKey:          image.MediaKey,
						Mimetype:          proto.String("image/webp"),
						FileEncSHA256:     image.FileEncSHA256,
						FileSHA256:        image.FileSHA256,
						FileLength:        proto.Uint64(image.FileLength),
						MediaKeyTimestamp: proto.Int64(now.Unix()),
						Width:             proto.Uint32(stickerSide),
						Height:            proto.Uint32(stickerSide),
						IsAnimated:        proto.Bool(animated),
						StickerSentTS:     proto.Int64(now.UnixMilli()),
						Audio: &waE2E.StickerMessage_AudioMessage{
							AudioMessage: &waE2E.AudioMessage{
								URL:               proto.String(sound.URL),
								DirectPath:        proto.String(sound.DirectPath),
								MediaKey:          sound.MediaKey,
								Mimetype:          proto.String(audioStickerOpus),
								FileEncSHA256:     sound.FileEncSHA256,
								FileSHA256:        sound.FileSHA256,
								FileLength:        proto.Uint64(sound.FileLength),
								MediaKeyTimestamp: proto.Int64(now.Unix()),
								Seconds:           proto.Uint32(seconds),
								ContextInfo:       &waE2E.ContextInfo{},
							},
						},
						ContextInfo: buildContext(evt, true, nil),
					},
				},
			},
			MessageContextInfo: &waE2E.MessageContextInfo{MessageSecret: messageSecret()},
		}

		_, err := client.SendMessage(ctx, evt.Info.Chat, msg)
		if err != nil {
			return fmt.Errorf("whatsapp: send audio sticker: %w", err)
		}
		return nil
	})
}

// messageSecret gera o segredo de 32 bytes exigido pelo MessageContextInfo.
// Campo opcional: se o entropy falhar, fica nil e o envio continua.
func messageSecret() []byte {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil
	}
	return secret
}

// SendStickerToJID envia figurinha salva para um JID sem evento.
func SendStickerToJID(ctx context.Context, client *whatsmeow.Client, jid types.JID, uploaded *whatsmeow.UploadResponse, animated bool) error {
	_ = client.SendChatPresence(ctx, jid, types.ChatPresenceComposing, types.ChatPresenceMediaText)
	defer client.SendChatPresence(ctx, jid, types.ChatPresencePaused, types.ChatPresenceMediaText)

	msg := &waE2E.Message{
		StickerMessage: &waE2E.StickerMessage{
			URL:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			Mimetype:      proto.String("image/webp"),
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uploaded.FileLength),
			IsAnimated:    proto.Bool(animated),
		},
	}

	_, err := client.SendMessage(ctx, jid, msg)
	if err != nil {
		return fmt.Errorf("whatsapp: send sticker to jid: %w", err)
	}
	return nil
}
