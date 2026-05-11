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

// WeatherCommand resolve o nome da cidade, busca clima atual e responde com um resumo formatado.
func WeatherCommand(
	ctx context.Context,
	client *whatsmeow.Client,
	evt *events.Message,
	args []string,
	geo *geocoding.GeoCoding,
	weatherClient *weather.WeatherClient,
) error {
	if len(args) == 0 {
		return utils.SendText(ctx, client, evt, "Por favor, informe o nome da cidade.", true)
	}

	query := strings.Join(args, " ")

	results, err := geo.Lookup(ctx, query, 1)
	if err != nil || len(results) == 0 {
		return replyOrUnderlying(ctx, client, evt, "Não consegui encontrar a cidade.", err)
	}

	loc := results[0]
	weatherData, err := weatherClient.GetCurrentWeather(ctx, loc.Latitude, loc.Longitude)
	if err != nil {
		return replyOrUnderlying(ctx, client, evt, "Não consegui pegar o clima.", err)
	}

	msg := buildWeatherMessage(loc, weatherData)
	return utils.SendText(ctx, client, evt, msg, true)
}

// replyOrUnderlying notifica o usuário com reply; se o envio falhar devolve esse erro, senão o erro original (pode ser nil).
func replyOrUnderlying(
	ctx context.Context,
	client *whatsmeow.Client,
	evt *events.Message,
	userMsg string,
	underlying error,
) error {
	if replyErr := utils.Reply(ctx, client, evt, userMsg); replyErr != nil {
		return replyErr
	}
	return underlying
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

// splitLocation interpreta DisplayName no estilo Nominatim ("cidade, estado, país, ...").
// Com três ou mais segmentos, o país é o último token (regiões intermediárias variam).
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
