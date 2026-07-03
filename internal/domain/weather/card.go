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
	heroH         = 290
	rowH          = 90
	cardPadding   = 20
	iconSizeRow   = 36.0

	// row X layout — measured: "Amanhã"=86px at 22pt bold
	rowLabelX   = 30.0  // label anchor (left)
	rowLabelW   = 120.0 // max label width (86 + 34 safety)
	rowIconX    = 200.0 // icon center, after label end + gap
	rowDescX    = 230.0 // description anchor (left)
	rowDescMaxW = 260.0 // max description width
	rowTempX    = 660.0 // temp right-aligned to this x
	rowSlashX   = 655.0 // "/" right-aligned
	rowMinX     = 665.0 // min temp anchor (left)
	rowPrecipCX = 710.0 // precip circle center
	rowPrecipTX = 720.0 // precip text anchor (left)
)

var (
	fontInit   sync.Once
	fontErr    error
	faceReg26  font.Face
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

		faceReg26 = truetype.NewFace(regular, &truetype.Options{Size: 26})
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

	// hero background — fundo sólido (sem gradiente)
	drawHeroBackground(dc, forecasts[0].WeatherCode)

	// fundo sólido da lista
	divY := float64(heroH)
	dc.SetRGBA(0.14, 0.16, 0.24, 1)
	dc.DrawRectangle(0, divY, cardW, float64(cardH-heroH))
	dc.Fill()

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
	// tom fixo azul-noite — sem gradiente, sem variação por condição
	dc.SetRGBA(0.094, 0.125, 0.227, 1) // #18203A
	dc.DrawRectangle(0, 0, cardW, float64(heroH))
	dc.Fill()
}

func drawHeroSection(dc *gg.Context, today DailyForecast, current *WeatherResult, location, country string) {
	// pin + localização
	drawPinIcon(dc, 30, 35, 14)
	label := location
	if country != "" {
		label = location + ", " + country
	}
	dc.SetFontFace(faceBold26)
	dc.SetRGB(1, 1, 1)
	dc.DrawStringAnchored(label, 55, 35, 0, 0.5)

	// temperatura grande — current.Temperature se disponível, senão daily max
	var temp, apparent float64
	if current != nil {
		temp = current.Temperature
		apparent = current.ApparentTemperature
	} else {
		temp = today.TempMax
		apparent = today.ApparentTempMax
	}
	dc.SetFontFace(faceBold72)
	dc.SetRGB(1, 1, 1)
	dc.DrawStringAnchored(fmt.Sprintf("%.0f°", temp), 40, 120, 0, 0.5)

	// condição
	weatherCode := today.WeatherCode
	info := Lookup(weatherCode)
	dc.SetFontFace(faceReg26)
	dc.SetRGB(1, 1, 1)
	dc.DrawStringAnchored(info.Description, 40, 185, 0, 0.5)

	// linha de máx/mín com setas
	y1 := 225.0
	drawUpArrow(dc, 40, y1, 10)
	dc.SetFontFace(faceReg20)
	dc.DrawStringAnchored(fmt.Sprintf("%.0f°", today.TempMax), 60, y1, 0, 0.5)
	drawDownArrow(dc, 135, y1, 10)
	dc.DrawStringAnchored(fmt.Sprintf("%.0f°", today.TempMin), 155, y1, 0, 0.5)

	// sensação térmica — linha separada abaixo das setas
	dc.SetFontFace(faceReg18)
	dc.SetRGBA(1, 1, 1, 0.8)
	dc.DrawStringAnchored(fmt.Sprintf("Sensação térmica de %.0f°", apparent), 40, 255, 0, 0.5)
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
	dc.DrawStringAnchored(label, rowLabelX, midY, 0, 0.5)
}

func drawDayIcon(dc *gg.Context, f DailyForecast, y int) {
	x, yPos := rowIconX, float64(y)+float64(rowH)/2
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
	dc.DrawStringAnchored(info.Description, rowDescX, midY, 0, 0.5)
}

func drawDayTemp(dc *gg.Context, f DailyForecast, y int) {
	midY := float64(y) + float64(rowH)/2
	textMax := fmt.Sprintf("%.0f°", f.TempMax)
	textMin := fmt.Sprintf("%.0f°", f.TempMin)

	maxW, _ := dc.MeasureString(textMax)

	dc.SetFontFace(faceBold22)
	dc.SetRGB(1, 1, 1)
	dc.DrawStringAnchored(textMax, rowTempX-maxW, midY, 0, 0.5)

	dc.SetFontFace(faceReg18)
	dc.SetRGBA(1, 1, 1, 0.5)
	dc.DrawStringAnchored("/", rowSlashX, midY, 1, 0.5)

	dc.SetFontFace(faceReg20)
	dc.SetRGBA(1, 1, 1, 0.65)
	dc.DrawStringAnchored(textMin, rowMinX, midY, 0, 0.5)
}

func drawDayPrecip(dc *gg.Context, f DailyForecast, y int) {
	if f.PrecipitationProb <= 30 {
		return
	}

	midY := float64(y) + float64(rowH)/2

	dc.SetRGBA(0.3, 0.6, 0.9, 0.8)
	dc.DrawCircle(rowPrecipCX, midY-6, 5)
	dc.Fill()

	dc.SetFontFace(faceReg14)
	dc.SetRGBA(1, 1, 1, 0.8)
	dc.DrawStringAnchored(fmt.Sprintf("%.0f%%", f.PrecipitationProb), rowPrecipTX, midY, 0, 0.5)
}

// ────────────────────────────── Icon helpers ───────────────────────────

func drawPinIcon(dc *gg.Context, x, y, size float64) {
	dc.SetRGBA(1, 1, 1, 0.9)
	r := size * 0.4
	// círculo do marcador
	dc.DrawCircle(x, y-size*0.1, r)
	dc.Fill()
	// ponta triangular apontando para baixo
	dc.MoveTo(x-r*1.2, y+size*0.1)
	dc.LineTo(x+r*1.2, y+size*0.1)
	dc.LineTo(x, y+size*0.5)
	dc.ClosePath()
	dc.Fill()
}

func drawUpArrow(dc *gg.Context, x, y, size float64) {
	dc.SetRGBA(1, 1, 1, 0.8)
	s := size * 0.5
	dc.MoveTo(x, y-s)
	dc.LineTo(x-s, y+s)
	dc.LineTo(x+s, y+s)
	dc.ClosePath()
	dc.Fill()
}

func drawDownArrow(dc *gg.Context, x, y, size float64) {
	dc.SetRGBA(1, 1, 1, 0.8)
	s := size * 0.5
	dc.MoveTo(x, y+s)
	dc.LineTo(x-s, y-s)
	dc.LineTo(x+s, y-s)
	dc.ClosePath()
	dc.Fill()
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
