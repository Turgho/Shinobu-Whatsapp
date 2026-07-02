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
	cardW       = 800
	headerH     = 130
	rowH        = 100
	cardPadding = 20
	iconSizeRow = 50.0
)

var (
	fontInit   sync.Once
	fontErr    error
	faceReg22  font.Face
	faceReg20  font.Face
	faceReg18  font.Face
	faceReg14  font.Face
	faceBold26 font.Face
	faceBold22 font.Face
)

// ensureFonts carrega as fontes embutidas no binário (golang.org/x/image/font/gofont).
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
		faceBold26 = truetype.NewFace(bold, &truetype.Options{Size: 26})
		faceBold22 = truetype.NewFace(bold, &truetype.Options{Size: 22})
	})
}

// GenerateForecastCard desenha um card vertical com a previsão dos próximos
// dias em PNG. Cada entrada em forecasts vira uma linha. Retorna os bytes
// da imagem ou erro (caller deve fazer fallback para texto).
func GenerateForecastCard(forecasts []DailyForecast, location, country string) ([]byte, error) {
	ensureFonts()
	if fontErr != nil {
		return nil, fmt.Errorf("weather card: %w", fontErr)
	}

	cardH := headerH + len(forecasts)*rowH + cardPadding*2
	dc := gg.NewContext(cardW, cardH)

	drawCardBackground(dc, cardH)

	drawHeader(dc, location, country)

	for i, f := range forecasts {
		y := headerH + i*rowH + cardPadding
		drawRowBackground(dc, i, y)
		drawDayLabel(dc, i, f, y)
		drawDayIcon(dc, f, y)
		drawDayDescription(dc, f, y)
		drawDayTemp(dc, f, y)
		drawDayPrecip(dc, f, y)

		dc.SetRGBA(1, 1, 1, 0.12)
		dc.SetLineWidth(1)
		dc.DrawLine(float64(cardPadding), float64(y+rowH), float64(cardW-cardPadding), float64(y+rowH))
		dc.Stroke()
	}

	var buf bytes.Buffer
	if err := dc.EncodePNG(&buf); err != nil {
		return nil, fmt.Errorf("weather card: encode PNG: %w", err)
	}
	return buf.Bytes(), nil
}

func drawCardBackground(dc *gg.Context, cardH int) {
	g := gg.NewLinearGradient(0, 0, cardW, float64(cardH))
	g.AddColorStop(0, rgba(40, 50, 80, 255))
	g.AddColorStop(1, rgba(60, 40, 70, 255))
	dc.SetFillStyle(g)
	dc.DrawRectangle(0, 0, cardW, float64(cardH))
	dc.Fill()
}

func drawHeader(dc *gg.Context, location, country string) {
	label := location
	if country != "" {
		label = location + ", " + country
	}

	dc.SetFontFace(faceBold26)
	dc.SetRGB(1, 1, 1)
	dc.DrawStringAnchored(label, 40, 45, 0, 0.5)

	dc.SetFontFace(faceReg20)
	dc.SetRGBA(1, 1, 1, 0.7)
	dc.DrawStringAnchored("Previsão para os próximos dias", 40, 80, 0, 0.5)
}

func drawRowBackground(dc *gg.Context, index int, y int) {
	if index%2 == 0 {
		return
	}

	dc.SetRGBA(1, 1, 1, 0.05)
	dc.DrawRectangle(float64(cardPadding), float64(y), float64(cardW-cardPadding*2), float64(rowH))
	dc.Fill()
}

func drawDayLabel(dc *gg.Context, index int, f DailyForecast, y int) {
	var label string
	switch index {
	case 0:
		label = "Hoje"
	case 1:
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

	if index == 0 {
		dc.SetRGBA(1, 0.85, 0.3, 1)
	}

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

	// max temp — bold, right-aligned
	xMax := 600.0
	dc.SetFontFace(faceBold22)
	dc.SetRGB(1, 1, 1)
	dc.DrawStringAnchored(textMax, xMax-maxW, midY, 0, 0.5)

	// separator
	dc.SetFontFace(faceReg18)
	dc.SetRGBA(1, 1, 1, 0.5)
	dc.DrawStringAnchored("/", xMax-5, midY, 1, 0.5)

	// min temp — regular, lighter
	dc.SetFontFace(faceReg20)
	dc.SetRGBA(1, 1, 1, 0.65)
	dc.DrawStringAnchored(textMin, xMax+5, midY, 0, 0.5)
}

func drawDayPrecip(dc *gg.Context, f DailyForecast, y int) {
	if f.PrecipitationProb <= 30 {
		return
	}

	midY := float64(y) + float64(rowH)/2

	// small drop icon (just a blue circle)
	dc.SetRGBA(0.3, 0.6, 0.9, 0.8)
	dc.DrawCircle(720, midY-6, 5)
	dc.Fill()

	dc.SetFontFace(faceReg14)
	dc.SetRGBA(1, 1, 1, 0.8)
	dc.DrawStringAnchored(fmt.Sprintf("%.0f%%", f.PrecipitationProb), 730, midY, 0, 0.5)
}

// WeekdayPT retorna a abreviação em português do dia da semana (3 letras).
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
