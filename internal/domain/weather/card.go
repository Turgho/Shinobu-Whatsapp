package weather

import (
	"bytes"
	"fmt"
	"image/color"
	"sync"

	"github.com/fogleman/gg"
	"golang.org/x/image/font"
)

const (
	cardW = 800
	cardH = 450

	iconSun = iota
	iconCloud
	iconRain
	iconSnow
	iconStorm
	iconFog
)

var (
	fontInit   sync.Once
	fontErr    error
	faceReg22  font.Face
	faceReg20  font.Face
	faceBold26 font.Face
	faceBold72 font.Face
	faceBold22 font.Face
)

func ensureFonts() {
	fontInit.Do(func() {
		base := "assets/fonts/Inter"
		faceReg22, fontErr = gg.LoadFontFace(base+"-Regular.ttf", 22)
		if fontErr != nil {
			return
		}
		faceReg20, fontErr = gg.LoadFontFace(base+"-Regular.ttf", 20)
		if fontErr != nil {
			return
		}
		faceBold26, fontErr = gg.LoadFontFace(base+"-Bold.ttf", 26)
		if fontErr != nil {
			return
		}
		faceBold72, fontErr = gg.LoadFontFace(base+"-Bold.ttf", 72)
		if fontErr != nil {
			return
		}
		faceBold22, fontErr = gg.LoadFontFace(base+"-Bold.ttf", 22)
	})
}

// GenerateCard desenha um card visual do clima em PNG e retorna os bytes.
// Em caso de erro de renderização, retorna nil e o erro — o caller deve
// fazer fallback para texto.
func GenerateCard(result *WeatherResult, location, country string) ([]byte, error) {
	ensureFonts()
	if fontErr != nil {
		return nil, fmt.Errorf("weather card: carregar fonte: %w", fontErr)
	}

	dc := gg.NewContext(cardW, cardH)

	drawBackground(dc, result.WeatherCode)
	drawLocation(dc, location, country)
	drawTemperature(dc, result.Temperature)
	drawDescription(dc, result.WeatherCode)
	drawIcon(dc, result.WeatherCode)
	drawStatsRow(dc, result)

	var buf bytes.Buffer
	if err := dc.EncodePNG(&buf); err != nil {
		return nil, fmt.Errorf("weather card: encode PNG: %w", err)
	}
	return buf.Bytes(), nil
}

func drawBackground(dc *gg.Context, code int) {
	g := gg.NewLinearGradient(0, 0, cardW, cardH)

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
	dc.DrawRectangle(0, 0, cardW, cardH)
	dc.Fill()
}

func drawLocation(dc *gg.Context, location, country string) {
	label := location
	if country != "" {
		label = location + ", " + country
	}

	dc.SetFontFace(faceBold26)
	dc.SetRGB(1, 1, 1)
	dc.DrawStringAnchored(label, 40, 50, 0, 0.5)
}

func drawTemperature(dc *gg.Context, temp float64) {
	dc.SetFontFace(faceBold72)
	dc.SetRGB(1, 1, 1)
	dc.DrawStringAnchored(fmt.Sprintf("%.0f°", temp), 40, 200, 0, 0.5)
}

func drawDescription(dc *gg.Context, code int) {
	info := Lookup(code)

	dc.SetFontFace(faceReg22)
	dc.SetRGB(1, 1, 1)
	dc.DrawStringAnchored(info.Description, 40, 250, 0, 0.5)
}

func drawIcon(dc *gg.Context, code int) {
	x, y := 600.0, 180.0
	size := 90.0

	switch iconCategory(code) {
	case iconSun:
		drawSunIcon(dc, x, y, size)
	case iconCloud:
		drawCloudIcon(dc, x, y, size)
	case iconRain:
		drawRainIcon(dc, x, y, size)
	case iconSnow:
		drawSnowIcon(dc, x, y, size)
	case iconStorm:
		drawStormIcon(dc, x, y, size)
	case iconFog:
		drawFogIcon(dc, x, y, size)
	}
}

func drawStatsRow(dc *gg.Context, result *WeatherResult) {
	y := 370.0

	stats := []struct {
		label string
		value string
	}{
		{"Sensação", fmt.Sprintf("%.0f°", result.ApparentTemperature)},
		{"Umidade", fmt.Sprintf("%.0f%%", result.RelativeHumidity)},
		{"Vento", fmt.Sprintf("%.1f km/h", result.WindSpeed)},
	}

	totalW := float64(len(stats)) * 150.0
	startX := (float64(cardW) - totalW) / 2.0

	for i, s := range stats {
		x := startX + float64(i)*150.0

		dc.SetFontFace(faceReg20)
		dc.SetRGB(1, 1, 1)
		dc.DrawStringAnchored(s.label, x+75, y-20, 0.5, 0.5)

		dc.SetFontFace(faceBold22)
		dc.SetRGB(1, 1, 1)
		dc.DrawStringAnchored(s.value, x+75, y+15, 0.5, 0.5)
	}
}

func iconCategory(code int) int {
	switch {
	case code <= 1:
		return iconSun
	case code <= 3:
		return iconCloud
	case code == 45 || code == 48:
		return iconFog
	case code >= 51 && code <= 67:
		return iconRain
	case code >= 71 && code <= 77:
		return iconSnow
	case code >= 80 && code <= 82:
		return iconRain
	case code >= 85 && code <= 86:
		return iconSnow
	case code >= 95:
		return iconStorm
	default:
		return iconCloud
	}
}

func rgba(r, g, b, a int) color.Color {
	return color.NRGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: uint8(a)}
}
