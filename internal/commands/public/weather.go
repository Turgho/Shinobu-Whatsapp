package public

import (
	"context"
	"fmt"
	"strings"

	"github.com/Turgho/YuukoWhatsapp/internal/utils"
	"github.com/Turgho/YuukoWhatsapp/pkg/geocoding"
	"github.com/Turgho/YuukoWhatsapp/pkg/weather"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

func WeatherCommand(
	ctx context.Context,
	client *whatsmeow.Client,
	evt *events.Message,
	args []string,
	geo *geocoding.GeoCoding,
	weatherClient *weather.WeatherClient,
) error {
	if len(args) == 0 {
		return utils.Reply(ctx, client, evt, "Por favor, informe o nome da cidade.")
	}

	query := strings.Join(args, " ")

	results, err := geo.Lookup(query, 1)
	if err != nil || len(results) == 0 {
		if replyErr := utils.Reply(ctx, client, evt, "Não consegui encontrar a cidade."); replyErr != nil {
			return replyErr
		}
		return err
	}

	loc := results[0]

	// Buscar clima
	weatherData, err := weatherClient.GetCurrentWeather(loc.Latitude, loc.Longitude)
	if err != nil {
		if replyErr := utils.Reply(ctx, client, evt, "Não consegui pegar o clima."); replyErr != nil {
			return replyErr
		}
		return err
	}

	msg := buildWeatherMessage(loc, weatherData)

	return utils.Reply(ctx, client, evt, msg)
}

func buildWeatherMessage(loc geocoding.GeoResult, w *weather.WeatherResult) string {
	cidade, _, pais := splitLocation(loc.DisplayName)
	info := weather.WeatherCodeMap[w.WeatherCode]

	return fmt.Sprintf(
		"*🌍 Local:* %s, %s\n"+
			"*🌡️ Temperatura atual:* `%.1f°C`\n"+
			"*🤗 Sensação térmica:* `%.1f°C`\n"+
			"*💨 Vento:* `%.1f km/h` _%.0f°_\n"+
			"%s *%s*",
		cidade,
		pais,
		w.Temperature,
		w.ApparentTemperature,
		w.WindSpeed,
		w.WindDirection,
		info.Emoji,
		info.Description,
	)
}

func splitLocation(displayName string) (cidade, estado, pais string) {
	parts := strings.Split(displayName, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}

	if len(parts) > 0 {
		cidade = parts[0]
	}
	if len(parts) > 1 {
		estado = parts[1]
	}
	if len(parts) > 2 {
		pais = parts[len(parts)-1]
	}
	return
}
