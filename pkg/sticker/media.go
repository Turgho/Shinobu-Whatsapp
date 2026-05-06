package sticker

import (
	"context"
	"fmt"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// Media agrupa os bytes baixados e o tipo da mídia encontrada.
type Media struct {
	Data     []byte
	Ext      string // extensão do arquivo de entrada, ex: ".jpg", ".mp4"
	Animated bool   // true = sticker animado (vídeo/gif)
}

// DownloadFromEvent encontra e baixa a mídia de um evento de mensagem.
// Tenta em ordem: imagem atual → vídeo atual → imagem citada → vídeo citado.
func DownloadFromEvent(ctx context.Context, client *whatsmeow.Client, evt *events.Message) (*Media, error) {
	info, err := extractDownloadable(evt)
	if err != nil {
		return nil, err
	}

	data, err := client.Download(ctx, info.source)
	if err != nil {
		return nil, fmt.Errorf("sticker: falha ao baixar mídia: %w", err)
	}

	return &Media{
		Data:     data,
		Ext:      info.ext,
		Animated: info.animated,
	}, nil
}

// downloadable agrupa os campos necessários para baixar uma mídia.
type downloadable struct {
	source   whatsmeow.DownloadableMessage
	ext      string
	animated bool
}

// extractDownloadable localiza a mídia dentro do evento, com ou sem citação.
func extractDownloadable(evt *events.Message) (*downloadable, error) {
	msg := evt.Message

	if img := msg.GetImageMessage(); img != nil {
		return &downloadable{source: img, ext: ".jpg", animated: false}, nil
	}

	if vid := msg.GetVideoMessage(); vid != nil {
		return &downloadable{source: vid, ext: ".mp4", animated: true}, nil
	}

	// Mensagem de texto citando uma mídia (usuário respondeu com !sticker)
	quoted := msg.GetExtendedTextMessage().GetContextInfo().GetQuotedMessage()
	if quoted == nil {
		return nil, fmt.Errorf("nenhuma mídia encontrada na mensagem ou na citação")
	}

	if img := quoted.GetImageMessage(); img != nil {
		return &downloadable{source: img, ext: ".jpg", animated: false}, nil
	}

	if vid := quoted.GetVideoMessage(); vid != nil {
		return &downloadable{source: vid, ext: ".mp4", animated: true}, nil
	}

	return nil, fmt.Errorf("tipo de mídia não suportado (use imagem ou vídeo)")
}
