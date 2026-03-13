package public

import (
	"fmt"
	"strings"

	"github.com/Turgho/YuukoWhatsapp/internal/utils"
	"github.com/Turgho/YuukoWhatsapp/pkg/geocoding"
	"github.com/Turgho/YuukoWhatsapp/pkg/weather"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

func WeatherCommand(
	client *whatsmeow.Client,
	evt *events.Message,
	args []string,
	geo *geocoding.GeoCoding,
	weatherClient *weather.WeatherClient,
) error {
	if len(args) == 0 {
		if err := utils.Reply(client, evt, "Por favor, informe o nome da cidade."); err != nil {
			return err
		}
		return nil
	}

	query := strings.Join(args, " ") // junta ["São", "Paulo"] → "São Paulo"

	// Busca coordenadas
	results, err := geo.Lookup(query, 1)
	if err != nil || len(results) == 0 {
		err = utils.Reply(client, evt, "Não consegui encontrar a cidade.")
		if err != nil {
			return err
		}
		return err
	}
	loc := results[0]

	// Busca clima horário
	weatherResults, err := weatherClient.GetHourlyWeather(loc.Latitude, loc.Longitude)
	if err != nil || len(weatherResults) == 0 {
		err = utils.Reply(client, evt, "Não consegui pegar o clima.")
		if err != nil {
			return err
		}
		return err
	}
	w := weatherResults[0]

	// Monta mensagem
	msg := buildWeatherMessage(loc, w)

	return utils.Reply(client, evt, msg)
}

func buildWeatherMessage(loc geocoding.GeoResult, w weather.WeatherResult) string {
	// Cidade, estado, país
	cidade, _, pais := splitLocation(loc.DisplayName)

	// Descrição e emoji
	info := weather.WeatherCodeMap[w.WeatherCode]

	// Monta a mensagem formatada
	msg := fmt.Sprintf(
		"*🌍 Local:* %s, %s\n"+
			"*🌡️ Temperatura:* `%.1f°C`\n"+
			"*🤗 Sensação térmica:* `%.1f°C`\n"+
			"*☔ Chuva:* `%.1fmm` (_%.0f%% chance_)\n"+
			"*💨 Vento:* `%.1f km/h` _%.0f°_\n"+
			"%s *%s*",
		cidade,
		pais,
		w.Temperature,
		w.ApparentTemperature,
		w.Precipitation,
		w.PrecipitationProb,
		w.WindSpeed,
		w.WindDirection,
		info.Emoji,
		info.Description,
	)

	return msg
}

// splitLocation separa cidade, estado e país do displayName
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
