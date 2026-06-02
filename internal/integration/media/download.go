package media

import (
	"context"
	"fmt"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

const MaxDownloadSize = 50 * 1024 * 1024

// Type representa o tipo de mídia encontrada.
type Type int

const (
	TypeImage Type = iota
	TypeVideo
	TypeAudio
)

// Media agrupa os bytes baixados e metadados da mídia encontrada.
type Media struct {
	Data     []byte
	Ext      string // ex: ".jpg", ".mp4", ".m4a"
	Animated bool   // true = vídeo/gif (para stickers animados)
	Type     Type
}

// Filter define quais tipos de mídia aceitar no download.
type Filter struct {
	Image    bool
	Video    bool
	Audio    bool
	Document bool // mp3/arquivo enviado como documento
}

var (
	FilterAll       = Filter{Image: true, Video: true, Audio: true, Document: true}
	FilterVisual    = Filter{Image: true, Video: true}
	FilterAudioOnly = Filter{Audio: true, Document: true}
)

// DownloadFromEvent encontra e baixa mídia no evento ou na mensagem citada.
func DownloadFromEvent(ctx context.Context, client *whatsmeow.Client, evt *events.Message, f Filter) (*Media, error) {
	info, err := extractDownloadable(evt, f)
	if err != nil {
		return nil, err
	}

	data, err := client.Download(ctx, info.source)
	if err != nil {
		return nil, fmt.Errorf("media: falha ao baixar mídia: %w", err)
	}

	// Checagem pós-download: a API do WhatsApp não expõe o tamanho do arquivo
	// de forma confiável antes de baixar (alguns tipos de mídia não têm o campo).
	if len(data) > MaxDownloadSize {
		return nil, fmt.Errorf("media: mídia muito grande (%d bytes, máximo %d)", len(data), MaxDownloadSize)
	}

	return &Media{
		Data:     data,
		Ext:      info.ext,
		Animated: info.animated,
		Type:     info.mediaType,
	}, nil
}

type downloadable struct {
	source    whatsmeow.DownloadableMessage
	ext       string
	animated  bool
	mediaType Type
}

func extractDownloadable(evt *events.Message, f Filter) (*downloadable, error) {
	msg := evt.Message

	if d := fromDirect(
		msg.GetImageMessage(),
		msg.GetVideoMessage(),
		msg.GetAudioMessage(),
		msg.GetDocumentMessage(),
		f,
	); d != nil {
		return d, nil
	}

	quoted := msg.GetExtendedTextMessage().GetContextInfo().GetQuotedMessage()
	if quoted == nil {
		return nil, fmt.Errorf("nenhuma mídia encontrada na mensagem ou na citação")
	}

	if d := fromDirect(
		quoted.GetImageMessage(),
		quoted.GetVideoMessage(),
		quoted.GetAudioMessage(),
		quoted.GetDocumentMessage(),
		f,
	); d != nil {
		return d, nil
	}

	return nil, fmt.Errorf("tipo de mídia não suportado pelo comando")
}

func fromDirect(
	img whatsmeow.DownloadableMessage,
	vid whatsmeow.DownloadableMessage,
	aud whatsmeow.DownloadableMessage,
	doc whatsmeow.DownloadableMessage,
	f Filter,
) *downloadable {
	if f.Image && !isNil(img) {
		return &downloadable{source: img, ext: ".jpg", animated: false, mediaType: TypeImage}
	}
	if f.Video && !isNil(vid) {
		return &downloadable{source: vid, ext: ".mp4", animated: true, mediaType: TypeVideo}
	}
	if f.Audio && !isNil(aud) {
		return &downloadable{source: aud, ext: ".m4a", animated: false, mediaType: TypeAudio}
	}
	if f.Document && !isNil(doc) {
		return &downloadable{source: doc, ext: ".mp3", animated: false, mediaType: TypeAudio}
	}
	return nil
}

func isNil(d whatsmeow.DownloadableMessage) bool {
	if d == nil {
		return true
	}
	switch v := d.(type) {
	case interface{ GetDirectPath() string }:
		return v.GetDirectPath() == ""
	}
	return false
}
