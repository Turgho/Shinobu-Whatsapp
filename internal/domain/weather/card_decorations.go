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
func listAlpha(alpha float64) float64 { return alpha * 0.15 }

// ────────────────────────────── Sun ────────────────────────────────────

func drawSunDecoration(dc *gg.Context, cardW, heroH, cardH int) {
	// hero: círculos contidos no hero, sem sangrar na lista
	dc.SetRGBA(1, 0.85, 0.3, heroAlpha(0.08))
	dc.DrawCircle(560, 110, 100)
	dc.Fill()
	dc.SetRGBA(1, 0.75, 0.2, heroAlpha(0.06))
	dc.DrawCircle(660, 80, 80)
	dc.Fill()
	dc.SetRGBA(1, 0.9, 0.4, heroAlpha(0.04))
	dc.DrawCircle(480, 170, 70)
	dc.Fill()

	// lista: círculos pequenos e quase invisíveis
	dc.SetRGBA(1, 0.85, 0.3, listAlpha(0.08))
	dc.DrawCircle(600, 320, 50)
	dc.Fill()
	dc.SetRGBA(1, 0.75, 0.2, listAlpha(0.08))
	dc.DrawCircle(680, 460, 40)
	dc.Fill()
}

// ────────────────────────────── Cloud ──────────────────────────────────

func drawCloudDecoration(dc *gg.Context, cardW, heroH, cardH int) {
	heroClouds := []struct {
		x, y, rx, ry float64
	}{
		{400, 120, 140, 45},
		{650, 80, 110, 40},
		{200, 160, 100, 35},
	}

	listClouds := []struct {
		x, y, rx, ry float64
	}{
		{600, 300, 80, 25},
		{500, 420, 70, 20},
		{680, 500, 60, 18},
	}

	for _, c := range heroClouds {
		dc.SetRGBA(0.85, 0.87, 0.92, heroAlpha(0.08))
		dc.DrawEllipse(c.x, c.y, c.rx, c.ry)
		dc.Fill()
	}
	for _, c := range listClouds {
		dc.SetRGBA(0.85, 0.87, 0.92, listAlpha(0.08))
		dc.DrawEllipse(c.x, c.y, c.rx, c.ry)
		dc.Fill()
	}
}

// ────────────────────────────── Fog ────────────────────────────────────

func drawFogDecoration(dc *gg.Context, cardW, heroH, cardH int) {
	heroBands := []float64{50, 100, 150, 200}
	listBands := make([]float64, 0)
	for y := heroH + 40; y < cardH; y += 55 {
		listBands = append(listBands, float64(y))
	}

	for _, yPos := range heroBands {
		dc.SetRGBA(0.75, 0.78, 0.85, heroAlpha(0.05))
		dc.DrawRectangle(0, yPos-5, float64(cardW), 10)
		dc.Fill()
	}
	for _, yPos := range listBands {
		dc.SetRGBA(0.75, 0.78, 0.85, listAlpha(0.05))
		dc.DrawRectangle(0, yPos-2, float64(cardW), 4)
		dc.Fill()
	}
}
