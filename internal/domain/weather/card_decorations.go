package weather

import "github.com/fogleman/gg"

// drawCardDecoration desenha elementos decorativos de fundo no card
// climatico inteiro, baseado na categoria de hoje. A opacidade é maior
// no hero (topo) e reduzida na lista de dias.
func drawCardDecoration(dc *gg.Context, code int, cardW, heroH, cardH int) {
	switch iconCategory(code) {
	case iconSun:
		drawSunDecoration(dc, cardW, heroH, cardH)
	case iconCloud:
		drawCloudDecoration(dc, cardW, heroH, cardH)
	case iconFog:
		drawFogDecoration(dc, cardW, heroH, cardH)
	case iconRain:
		drawRainDecoration(dc, cardW, heroH, cardH)
	case iconSnow:
		drawSnowDecoration(dc, cardW, heroH, cardH)
	case iconStorm:
		drawStormDecoration(dc, cardW, heroH, cardH)
	}
}

func heroAlpha(alpha float64) float64 { return alpha }
func listAlpha(alpha float64) float64 { return alpha * 0.3 }

// ────────────────────────────── Sun ────────────────────────────────────

func drawSunDecoration(dc *gg.Context, cardW, heroH, cardH int) {
	positions := []struct {
		x, y, r    float64
		r_, g_, b_ float64
	}{
		{560, 80, 220, 1, 0.85, 0.3},
		{700, 140, 280, 1, 0.75, 0.2},
		{480, 180, 160, 1, 0.9, 0.4},
		{650, 50, 200, 1, 0.8, 0.25},
		{580, 260, 240, 1, 0.85, 0.3},
		{720, 300, 200, 1, 0.75, 0.2},
		{440, 350, 150, 1, 0.9, 0.4},
		{660, 420, 180, 1, 0.8, 0.25},
	}

	for _, p := range positions {
		a := heroAlpha(0.10)
		if int(p.y) > heroH {
			a = listAlpha(0.10)
		}
		dc.SetRGBA(p.r_, p.g_, p.b_, a)
		dc.DrawCircle(p.x, p.y, p.r)
		dc.Fill()
	}
}

// ────────────────────────────── Cloud ──────────────────────────────────

func drawCloudDecoration(dc *gg.Context, cardW, heroH, cardH int) {
	heroClouds := []struct {
		x, y, rx, ry float64
	}{
		{150, 60, 120, 50},
		{400, 100, 180, 60},
		{650, 50, 140, 55},
		{300, 170, 100, 40},
		{550, 180, 160, 50},
	}

	listClouds := []struct {
		x, y, rx, ry float64
	}{
		{650, 280, 130, 40},
		{500, 350, 110, 35},
		{700, 390, 90, 30},
		{620, 460, 120, 35},
		{550, 520, 100, 30},
	}

	for _, c := range heroClouds {
		dc.SetRGBA(0.85, 0.87, 0.92, heroAlpha(0.10))
		dc.DrawEllipse(c.x, c.y, c.rx, c.ry)
		dc.Fill()
	}
	for _, c := range listClouds {
		dc.SetRGBA(0.85, 0.87, 0.92, listAlpha(0.10))
		dc.DrawEllipse(c.x, c.y, c.rx, c.ry)
		dc.Fill()
	}
}

// ────────────────────────────── Fog ────────────────────────────────────

func drawFogDecoration(dc *gg.Context, cardW, heroH, cardH int) {
	heroBands := []float64{40, 90, 140, 190}
	listBands := make([]float64, 0)
	for y := heroH + 30; y < cardH; y += 45 {
		listBands = append(listBands, float64(y))
	}

	for _, yPos := range heroBands {
		dc.SetRGBA(0.75, 0.78, 0.85, heroAlpha(0.07))
		dc.DrawRectangle(0, yPos-6, float64(cardW), 12)
		dc.Fill()
	}
	for _, yPos := range listBands {
		dc.SetRGBA(0.75, 0.78, 0.85, listAlpha(0.07))
		dc.DrawRectangle(0, yPos-4, float64(cardW), 8)
		dc.Fill()
	}
}
