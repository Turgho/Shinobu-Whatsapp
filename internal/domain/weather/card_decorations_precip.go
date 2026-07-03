package weather

import (
	"math"

	"github.com/fogleman/gg"
)

// ────────────────────────────── Rain ───────────────────────────────────

func drawRainDecoration(dc *gg.Context, cardW, heroH, cardH int) {
	type streak struct{ x1, y1, x2, y2 float64 }

	heroStreaks := make([]streak, 0)
	for x := 20; x < cardW-20; x += 24 {
		for y := 15; y < heroH-10; y += 28 {
			if x > 40 && x < 500 && y > 20 && y < 200 {
				continue
			}
			heroStreaks = append(heroStreaks, streak{
				float64(x), float64(y),
				float64(x + 10), float64(y + 15),
			})
		}
	}

	listStreaks := make([]streak, 0)
	for x := 50; x < cardW-20; x += 60 {
		for y := heroH + 20; y < cardH-10; y += 60 {
			listStreaks = append(listStreaks, streak{
				float64(x), float64(y),
				float64(x + 8), float64(y + 12),
			})
		}
	}

	dc.SetLineWidth(1.2)
	for _, s := range heroStreaks {
		dc.SetRGBA(0.45, 0.6, 0.9, heroAlpha(0.10))
		dc.DrawLine(s.x1, s.y1, s.x2, s.y2)
		dc.Stroke()
	}
	for _, s := range listStreaks {
		dc.SetRGBA(0.45, 0.6, 0.9, listAlpha(0.10))
		dc.DrawLine(s.x1, s.y1, s.x2, s.y2)
		dc.Stroke()
	}
}

// ────────────────────────────── Snow ───────────────────────────────────

func drawSnowDecoration(dc *gg.Context, cardW, heroH, cardH int) {
	type flake struct{ x, y, r float64 }

	heroFlakes := make([]flake, 0)
	for x := 20; x < cardW; x += 35 {
		for y := 15; y < heroH; y += 38 {
			offsetX := math.Sin(float64(x+y)*0.3) * 6
			heroFlakes = append(heroFlakes, flake{
				float64(x) + offsetX,
				float64(y) + math.Cos(float64(x+y)*0.2)*4,
				3.0,
			})
		}
	}

	listFlakes := make([]flake, 0)
	for x := 50; x < cardW; x += 80 {
		for y := heroH + 25; y < cardH; y += 80 {
			listFlakes = append(listFlakes, flake{
				float64(x),
				float64(y),
				1.5,
			})
		}
	}

	for _, f := range heroFlakes {
		dc.SetRGBA(0.9, 0.92, 1, heroAlpha(0.12))
		dc.DrawCircle(f.x, f.y, f.r)
		dc.Fill()
	}
	for _, f := range listFlakes {
		dc.SetRGBA(0.9, 0.92, 1, listAlpha(0.12))
		dc.DrawCircle(f.x, f.y, f.r)
		dc.Fill()
	}
}

// ────────────────────────────── Storm ──────────────────────────────────

func drawStormDecoration(dc *gg.Context, cardW, heroH, cardH int) {
	// relâmpago sutil só no hero
	bolt := func(x, y, s float64, a float64) {
		dc.SetLineWidth(2.5)
		dc.MoveTo(x, y)
		dc.LineTo(x+s*0.4, y-s*0.3)
		dc.LineTo(x-s*0.1, y-s*0.3)
		dc.LineTo(x+s*0.5, y+s*0.6)
		dc.LineTo(x+s*0.15, y+s*0.6)
		dc.LineTo(x+s*0.3, y+s*1.0)
		dc.LineTo(x-s*0.15, y+s*0.4)
		dc.LineTo(x+s*0.1, y+s*0.4)
		dc.ClosePath()
		dc.SetRGBA(1, 0.85, 0.2, a)
		dc.Fill()
		dc.SetRGBA(1, 0.9, 0.4, a*0.4)
		dc.DrawCircle(x+s*0.15, y-s*0.1, s*0.35)
		dc.Fill()
	}

	bolt(580, 100, 55, heroAlpha(0.06))
	bolt(640, 60, 30, heroAlpha(0.04))

	// chuva esparsa: hero moderada, lista quase imperceptível
	type line struct{ x1, y1, x2, y2 float64 }

	heroStreaks := make([]line, 0)
	for x := 20; x < cardW; x += 30 {
		for y := 15; y < heroH; y += 35 {
			if x > 40 && x < 500 && y > 20 && y < 130 {
				continue
			}
			heroStreaks = append(heroStreaks, line{
				float64(x), float64(y),
				float64(x + 8), float64(y + 12),
			})
		}
	}

	listStreaks := make([]line, 0)
	for x := 60; x < cardW-20; x += 70 {
		for y := heroH + 30; y < cardH-10; y += 70 {
			listStreaks = append(listStreaks, line{
				float64(x), float64(y),
				float64(x + 6), float64(y + 10),
			})
		}
	}

	dc.SetLineWidth(1.2)
	for _, s := range heroStreaks {
		dc.SetRGBA(0.45, 0.55, 0.85, heroAlpha(0.08))
		dc.DrawLine(s.x1, s.y1, s.x2, s.y2)
		dc.Stroke()
	}
	for _, s := range listStreaks {
		dc.SetRGBA(0.45, 0.55, 0.85, listAlpha(0.08))
		dc.DrawLine(s.x1, s.y1, s.x2, s.y2)
		dc.Stroke()
	}
}
