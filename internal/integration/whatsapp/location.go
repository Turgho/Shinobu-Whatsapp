package whatsapp

import (
	"context"
	"fmt"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

// SendLocation envia pin de mapa com coordenadas e nome/endereço opcionais.
func SendLocation(
	ctx context.Context,
	client *whatsmeow.Client,
	evt *events.Message,
	lat float64,
	lon float64,
	name string,
	address string,
	reply bool,
) error {
	return withTyping(ctx, client, evt, types.ChatPresenceMediaText, func() error {
		msg := &waE2E.Message{
			LocationMessage: &waE2E.LocationMessage{
				DegreesLatitude:  proto.Float64(lat),
				DegreesLongitude: proto.Float64(lon),
				Name:             proto.String(name),
				Address:          proto.String(address),
				ContextInfo:      buildContext(evt, reply, nil),
			},
		}
		_, err := client.SendMessage(ctx, evt.Info.Chat, msg)
		if err != nil {
			return fmt.Errorf("whatsapp: send location: %w", err)
		}
		return nil
	})
}
