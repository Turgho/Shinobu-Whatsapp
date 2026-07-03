package weather

import (
	"bytes"
	"fmt"
	"image/color"
	"sync"
	"time"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
)

const (
	cardW         = 800
	heroH         = 220
	rowH          = 90
	cardPadding   = 20
	iconSizeRow   = 50.0
	iconSizeHero  = 100.0
)

var (
	fontInit   sync.Once
	fontErr    error
	faceReg22  font.Face
	faceReg20  font.Face
	faceReg18  font.Face
	faceReg14  font.Face
	faceBold72 font.Face
	faceBold26 font.Face
	faceBold22 font.Face
)

func ensureFonts() {
	fontInit.Do(func() {
		regular, err := truetype.Parse(goregular.TTF)
		if err != nil {
			fontErr = fmt.Errorf("weather card: parse fonte regular: %w", err)
			return
		}
		bold, err := truetype.Parse(gobold.TTF)
		if err != nil {
			fontErr = fmt.Errorf("weather card: parse fonte bold: %w", err)
			return
		}

		faceReg22 = truetype.NewFace(regular, &truetype.Options{Size: 22})
		faceReg20 = truetype.NewFace(regular, &truetype.Options{Size: 20})
		faceReg18 = truetype.NewFace(regular, &truetype.Options{Size: 18})
		faceReg14 = truetype.NewFace(regular, &truetype.Options{Size: 14})
		faceBold72 = truetype.NewFace(bold, &truetype.Options{Size: 72})
		faceBold26 = truetype.NewFace(bold, &truetype.Options{Size: 26})
		faceBold22 = truetype.NewFace(bold, &truetype.Options{Size: 22})
	})
}

// GenerateForecastCard desenha um card com seção hero (condição de hoje) e
// lista dos próximos dias (amanhã em diante). forecasts[0] deve ser hoje.
// current é opcional — se nil, usa forecasts[0] como dado do hero.
// Retorna bytes PNG ou erro (caller deve fazer fallback para texto).
func GenerateForecastCard(forecasts []DailyForecast, current *WeatherResult, location, country string) ([]byte, error) {
	ensureFonts()
	if fontErr != nil {
		return nil, fmt.Errorf("weather card: %w", fontErr)
	}

	numList := len(forecasts) - 1
	cardH := heroH + numList*rowH + cardPadding*2
	dc := gg.NewContext(cardW, cardH)

	// hero background — gradiente baseado no clima de hoje
	drawHeroBackground(dc, forecasts[0].WeatherCode)

	// fundo sólido da lista
	divY := float64(heroH)
	dc.SetRGBA(0.16, 0.18, 0.27, 1)
	dc.DrawRectangle(0, divY, cardW, float64(cardH-heroH))
	dc.Fill()

	// decoração de fundo sobre hero + lista, antes do conteúdo textual
	drawCardDecoration(dc, forecasts[0].WeatherCode, cardW, heroH, cardH)

	// divisor entre hero e lista
	dc.SetRGBA(1, 1, 1, 0.2)
	dc.SetLineWidth(2)
	dc.DrawLine(float64(cardPadding), divY, float64(cardW-cardPadding), divY)
	dc.Stroke()

	drawHeroSection(dc, forecasts[0], current, location, country)

	if numList > 0 {
		drawDailyList(dc, forecasts[1:], heroH)
	}

	var buf bytes.Buffer
	if err := dc.EncodePNG(&buf); err != nil {
		return nil, fmt.Errorf("weather card: encode PNG: %w", err)
	}
	return buf.Bytes(), nil
}

// ────────────────────────────── Hero ───────────────────────────────────

func drawHeroBackground(dc *gg.Context, code int) {
	g := gg.NewLinearGradient(0, 0, cardW, float64(heroH))

	switch iconCategory(code) {
	case iconSun:
		g.AddColorStop(0, rgba(100, 180, 255, 255))
		g.AddColorStop(1, rgba(255, 200, 100, 255))
	case iconCloud:
		g.AddColorStop(0, rgba(180, 190, 200, 255))
		g.AddColorStop(1, rgba(140, 150, 160, 255))
	case iconRain, iconFog:
		g.AddColorStop(0, rgba(70, 90, 120, 255))
		g.AddColorStop(1, rgba(100, 110, 130, 255))
	case iconSnow:
		g.AddColorStop(0, rgba(180, 210, 240, 255))
		g.AddColorStop(1, rgba(220, 230, 245, 255))
	case iconStorm:
		g.AddColorStop(0, rgba(40, 30, 60, 255))
		g.AddColorStop(1, rgba(60, 50, 70, 255))
	default:
		g.AddColorStop(0, rgba(180, 190, 200, 255))
		g.AddColorStop(1, rgba(140, 150, 160, 255))
	}

	dc.SetFillStyle(g)
	dc.DrawRectangle(0, 0, cardW, float64(heroH))
	dc.Fill()
}

func drawHeroSection(dc *gg.Context, today DailyForecast, current *WeatherResult, location, country string) {
	label := location
	if country != "" {
		label = location + ", " + country
	}

	dc.SetFontFace(faceBold26)
	dc.SetRGB(1, 1, 1)
	dc.DrawStringAnchored(label, 40, 35, 0, 0.5)

	// temperatura grande — current.Temperature se disponível, senão daily max
	var temp float64
	var apparent float64
	if current != nil {
		temp = current.Temperature
		apparent = current.ApparentTemperature
	} else {
		temp = today.TempMax
		apparent = today.ApparentTempMax
	}

	dc.SetFontFace(faceBold72)
	dc.SetRGB(1, 1, 1)
	dc.DrawStringAnchored(fmt.Sprintf("%.0f°", temp), 40, 115, 0, 0.5)

	// descrição — usa weather code do current se disponível
	weatherCode := today.WeatherCode
	info := Lookup(weatherCode)
	dc.SetFontFace(faceReg22)
	dc.SetRGB(1, 1, 1)
	dc.DrawStringAnchored(info.Description, 40, 155, 0, 0.5)

	// linha: sensação · máx / mín
	sub := fmt.Sprintf("Sensação %.0f° · Máx %.0f° / Mín %.0f°",
		apparent, today.TempMax, today.TempMin)
	dc.SetFontFace(faceReg18)
	dc.SetRGBA(1, 1, 1, 0.75)
	dc.DrawStringAnchored(sub, 40, 190, 0, 0.5)

	drawBigIcon(dc, weatherCode)
}

func drawBigIcon(dc *gg.Context, code int) {
	x, yPos := 620.0, 115.0
	s := iconSizeHero

	switch iconCategory(code) {
	case iconSun:
		drawSunIcon(dc, x, yPos, s)
	case iconCloud:
		drawCloudIcon(dc, x, yPos, s)
	case iconRain:
		drawRainIcon(dc, x, yPos, s)
	case iconSnow:
		drawSnowIcon(dc, x, yPos, s)
	case iconStorm:
		drawStormIcon(dc, x, yPos, s)
	case iconFog:
		drawFogIcon(dc, x, yPos, s)
	}
}

// ────────────────────────────── Daily list ─────────────────────────────

func drawDailyList(dc *gg.Context, forecasts []DailyForecast, startY int) {
	for i, f := range forecasts {
		y := startY + i*rowH + cardPadding
		drawRowBackground(dc, i, y)
		drawDayLabel(dc, i, f, y)
		drawDayIcon(dc, f, y)
		drawDayDescription(dc, f, y)
		drawDayTemp(dc, f, y)
		drawDayPrecip(dc, f, y)

		dc.SetRGBA(1, 1, 1, 0.08)
		dc.SetLineWidth(1)
		dc.DrawLine(float64(cardPadding), float64(y+rowH), float64(cardW-cardPadding), float64(y+rowH))
		dc.Stroke()
	}
}

func drawRowBackground(dc *gg.Context, index int, y int) {
	if index%2 == 0 {
		return
	}

	dc.SetRGBA(1, 1, 1, 0.04)
	dc.DrawRectangle(float64(cardPadding), float64(y), float64(cardW-cardPadding*2), float64(rowH))
	dc.Fill()
}

func drawDayLabel(dc *gg.Context, index int, f DailyForecast, y int) {
	var label string
	switch index {
	case 0:
		label = "Amanhã"
	default:
		t, err := time.Parse("2006-01-02", f.Date)
		if err == nil {
			label = WeekdayPT(t)
		} else {
			label = f.Date
		}
	}

	midY := float64(y) + float64(rowH)/2
	dc.SetFontFace(faceBold22)
	dc.SetRGB(1, 1, 1)
	dc.DrawStringAnchored(label, 30, midY, 0, 0.5)
}

func drawDayIcon(dc *gg.Context, f DailyForecast, y int) {
	x, yPos := 105.0, float64(y)+float64(rowH)/2
	s := iconSizeRow

	switch iconCategory(f.WeatherCode) {
	case iconSun:
		drawSunIcon(dc, x, yPos, s)
	case iconCloud:
		drawCloudIcon(dc, x, yPos, s)
	case iconRain:
		drawRainIcon(dc, x, yPos, s)
	case iconSnow:
		drawSnowIcon(dc, x, yPos, s)
	case iconStorm:
		drawStormIcon(dc, x, yPos, s)
	case iconFog:
		drawFogIcon(dc, x, yPos, s)
	}
}

func drawDayDescription(dc *gg.Context, f DailyForecast, y int) {
	info := Lookup(f.WeatherCode)
	midY := float64(y) + float64(rowH)/2

	dc.SetFontFace(faceReg20)
	dc.SetRGB(1, 1, 1)
	dc.DrawStringAnchored(info.Description, 165, midY, 0, 0.5)
}

func drawDayTemp(dc *gg.Context, f DailyForecast, y int) {
	midY := float64(y) + float64(rowH)/2
	textMax := fmt.Sprintf("%.0f°", f.TempMax)
	textMin := fmt.Sprintf("%.0f°", f.TempMin)

	maxW, _ := dc.MeasureString(textMax)

	xMax := 600.0
	dc.SetFontFace(faceBold22)
	dc.SetRGB(1, 1, 1)
	dc.DrawStringAnchored(textMax, xMax-maxW, midY, 0, 0.5)

	dc.SetFontFace(faceReg18)
	dc.SetRGBA(1, 1, 1, 0.5)
	dc.DrawStringAnchored("/", xMax-5, midY, 1, 0.5)

	dc.SetFontFace(faceReg20)
	dc.SetRGBA(1, 1, 1, 0.65)
	dc.DrawStringAnchored(textMin, xMax+5, midY, 0, 0.5)
}

func drawDayPrecip(dc *gg.Context, f DailyForecast, y int) {
	if f.PrecipitationProb <= 30 {
		return
	}

	midY := float64(y) + float64(rowH)/2

	dc.SetRGBA(0.3, 0.6, 0.9, 0.8)
	dc.DrawCircle(720, midY-6, 5)
	dc.Fill()

	dc.SetFontFace(faceReg14)
	dc.SetRGBA(1, 1, 1, 0.8)
	dc.DrawStringAnchored(fmt.Sprintf("%.0f%%", f.PrecipitationProb), 730, midY, 0, 0.5)
}

// ────────────────────────────── Helpers ────────────────────────────────

func WeekdayPT(t time.Time) string {
	names := map[time.Weekday]string{
		time.Sunday:    "Dom",
		time.Monday:    "Seg",
		time.Tuesday:   "Ter",
		time.Wednesday: "Qua",
		time.Thursday:  "Qui",
		time.Friday:    "Sex",
		time.Saturday:  "Sáb",
	}
	return names[t.Weekday()]
}

func rgba(r, g, b, a int) color.Color {
	return color.NRGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: uint8(a)}
}
