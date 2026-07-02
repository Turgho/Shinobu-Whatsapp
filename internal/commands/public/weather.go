package public

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Turgho/Shinobu-Whatsapp/internal/commands"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/geocoding"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/weather"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
	"go.uber.org/zap"
)

var datePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

func WeatherCommand(
	ctx context.Context,
	client *whatsmeow.Client,
	evt *events.Message,
	args []string,
	geo *geocoding.GeoCoding,
	weatherClient *weather.WeatherClient,
	logger *zap.Logger,
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

	query := strings.TrimSpace(strings.Join(args, " "))
	if utf8.RuneCountInString(query) < 2 {
		return whatsapp.Reply(ctx, client, evt, "De qual cidade você quer saber o clima?")
	}

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

	// tenta gerar card visual com fallback para texto
	cardBytes, cardErr := weather.GenerateCard(weatherData, loc.DisplayName, loc.Country)
	if cardErr != nil {
		logger.Warn("falha ao gerar card do clima, enviando texto", zap.Error(cardErr))
		msg := buildWeatherMessage(loc, weatherData, hasDate)
		return whatsapp.Reply(ctx, client, evt, msg)
	}

	info := weather.Lookup(weatherData.WeatherCode)
	caption := fmt.Sprintf("%s %s — *%s, %s*",
		info.Emoji, info.Description, loc.DisplayName, loc.Country)

	if err := whatsapp.SendImageBytes(ctx, client, evt, cardBytes, caption); err != nil {
		logger.Warn("falha ao enviar card do clima, enviando texto", zap.Error(err))
		msg := buildWeatherMessage(loc, weatherData, hasDate)
		return whatsapp.Reply(ctx, client, evt, msg)
	}
	return nil
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

// WeatherHandler wraps WeatherCommand with dependencies.
func WeatherHandler(geo *geocoding.GeoCoding, wc *weather.WeatherClient, logger *zap.Logger) commands.HandlerFunc {
	l := logger.Named("WEATHER")
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		return WeatherCommand(ctx, client, evt, args, geo, wc, l)
	}
}
