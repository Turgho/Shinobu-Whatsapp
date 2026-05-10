package utils

import (
	"context"
	"fmt"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

// SendLocation sends a map pin with coordinates and an optional name.
//
//   - lat, lon:  GPS coordinates (e.g. -23.5505, -46.6333)
//   - name:      place name shown above the pin (e.g. "Escritório")
//   - address:   full address shown below the name (optional)
//   - reply:     true = quotes the original message
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
			return fmt.Errorf("utils/location: %w", err)
		}
		return nil
	})
}
