package public

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/geocoding"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/weather"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

var datePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

func WeatherCommand(
	ctx context.Context,
	client *whatsmeow.Client,
	evt *events.Message,
	args []string,
	geo *geocoding.GeoCoding,
	weatherClient *weather.WeatherClient,
) error {
	if len(args) == 0 {
		return whatsapp.Reply(ctx, client, evt, msgNeedCityName)
	}

	var targetDate time.Time
	hasDate := false

	if len(args) >= 2 && datePattern.MatchString(args[len(args)-1]) {
		parsed, err := time.Parse("2006-01-02", args[len(args)-1])
		if err == nil {
			targetDate = parsed
			hasDate = true
			args = args[:len(args)-1]
		}
	}

	query := strings.Join(args, " ")

	results, err := geo.Lookup(ctx, query, 1)
	if err != nil || len(results) == 0 {
		return replyOrUnderlying(ctx, client, evt, msgCityNotFound, err)
	}

	loc := results[0]

	var weatherData *weather.WeatherResult
	if hasDate {
		weatherData, err = weatherClient.GetForecastForDate(ctx, loc.Latitude, loc.Longitude, targetDate)
	} else {
		weatherData, err = weatherClient.GetCurrentWeather(ctx, loc.Latitude, loc.Longitude)
	}
	if err != nil {
		return replyOrUnderlying(ctx, client, evt, msgWeatherFailed, err)
	}

	msg := buildWeatherMessage(loc, weatherData, hasDate)
	return whatsapp.Reply(ctx, client, evt, msg)
}

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

func buildWeatherMessage(loc geocoding.GeoResult, w *weather.WeatherResult, isForecast bool) string {
	info := weather.WeatherCodeMap[w.WeatherCode]

	header := fmt.Sprintf("%s *%s*", info.Emoji, info.Description)
	if isForecast {
		header += " — *PREVISÃO*"
	}

	var dateLine string
	if isForecast {
		if parsed, err := time.Parse("2006-01-02T15:00", w.Time); err == nil {
			dateLine = fmt.Sprintf("📅 %s\n", parsed.Format("02/01/2006 15:04"))
		}
	}

	return fmt.Sprintf(
		"%s\n%s📍 %s, %s\n\n"+
			"🌡 `%.1f°C` — sensação `%.1f°C`\n"+
			"💧 Umidade `%.0f%%`\n"+
			"🌧 Chuva `%.1f mm` — chance `%.0f%%`\n"+
			"💨 Vento `%.1f km/h` — direção `%.0f°`",
		header,
		dateLine,
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
