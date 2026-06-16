package public

import (
	"context"
	"fmt"
	"strings"

	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/geocoding"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/weather"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
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
		return whatsapp.Reply(ctx, client, evt, "Por favor, informe o nome da cidade.")
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
	return whatsapp.Reply(ctx, client, evt, msg)
}

// replyOrUnderlying notifica o usuário com reply; se o envio falhar devolve esse erro, senão o erro original (pode ser nil).
func replyOrUnderlying(
	ctx context.Context,
	client *whatsmeow.Client,
	evt *events.Message,
	userMsg string,
	underlying error,
) error {
	if replyErr := whatsapp.Reply(ctx, client, evt, userMsg); replyErr != nil {
		return replyErr
	}
	return underlying
}

func buildWeatherMessage(loc geocoding.GeoResult, w *weather.WeatherResult) string {
	info := weather.WeatherCodeMap[w.WeatherCode]
	return fmt.Sprintf(
		"%s *%s*\n"+
			"📍 %s, %s\n\n"+
			"🌡 `%.1f°C` — sensação `%.1f°C`\n"+
			"💧 Umidade `%.0f%%`\n"+
			"🌧 Chuva `%.1f mm` — chance `%.0f%%`\n"+
			"💨 Vento `%.1f km/h` — direção `%.0f°`",
		info.Emoji,
		info.Description,
		loc.DisplayName,
		loc.Country,
		w.Temperature,
		w.ApparentTemperature,
		w.RelativeHumidity,
		w.Precipitation,
		w.PrecipitationProb,
		w.WindSpeed,
		w.WindDirection,
	)
}
