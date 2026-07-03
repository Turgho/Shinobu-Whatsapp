package weather

import (
	"math"

	"github.com/fogleman/gg"
)

// ────────────────────────────── Rain ───────────────────────────────────

func drawRainDecoration(dc *gg.Context, cardW, heroH, cardH int) {
	type streak struct{ x1, y1, x2, y2 float64 }

	heroStreaks := make([]streak, 0)
	for x := 10; x < cardW-10; x += 18 {
		for y := 10; y < heroH-10; y += 22 {
			if x > 30 && x < 500 && y > 20 && y < 200 {
				continue
			}
			heroStreaks = append(heroStreaks, streak{
				float64(x), float64(y),
				float64(x + 12), float64(y + 18),
			})
		}
	}

	listStreaks := make([]streak, 0)
	for x := 30; x < cardW-10; x += 35 {
		for y := heroH + 10; y < cardH-10; y += 40 {
			if x > 30 && x < 180 && y < heroH+rowH*4 {
				continue
			}
			listStreaks = append(listStreaks, streak{
				float64(x), float64(y),
				float64(x + 10), float64(y + 15),
			})
		}
	}

	dc.SetLineWidth(1.5)
	for _, s := range heroStreaks {
		dc.SetRGBA(0.45, 0.6, 0.9, heroAlpha(0.12))
		dc.DrawLine(s.x1, s.y1, s.x2, s.y2)
		dc.Stroke()
	}
	for _, s := range listStreaks {
		dc.SetRGBA(0.45, 0.6, 0.9, listAlpha(0.12))
		dc.DrawLine(s.x1, s.y1, s.x2, s.y2)
		dc.Stroke()
	}
}

// ────────────────────────────── Snow ───────────────────────────────────

func drawSnowDecoration(dc *gg.Context, cardW, heroH, cardH int) {
	type flake struct{ x, y, r float64 }

	heroFlakes := make([]flake, 0)
	for x := 20; x < cardW; x += 28 {
		for y := 15; y < heroH; y += 30 {
			offsetX := math.Sin(float64(x+y)*0.3) * 8
			offsetY := math.Cos(float64(x+y)*0.2) * 6
			heroFlakes = append(heroFlakes, flake{
				float64(x) + offsetX,
				float64(y) + offsetY,
				3.5,
			})
		}
	}

	listFlakes := make([]flake, 0)
	for x := 30; x < cardW; x += 50 {
		for y := heroH + 15; y < cardH; y += 50 {
			if x > 30 && x < 180 {
				continue
			}
			offsetX := math.Sin(float64(x+y)*0.3) * 6
			listFlakes = append(listFlakes, flake{
				float64(x) + offsetX,
				float64(y),
				2.5,
			})
		}
	}

	for _, f := range heroFlakes {
		dc.SetRGBA(0.9, 0.92, 1, heroAlpha(0.15))
		dc.DrawCircle(f.x, f.y, f.r)
		dc.Fill()
	}
	for _, f := range listFlakes {
		dc.SetRGBA(0.9, 0.92, 1, listAlpha(0.15))
		dc.DrawCircle(f.x, f.y, f.r)
		dc.Fill()
	}
}

// ────────────────────────────── Storm ──────────────────────────────────

func drawStormDecoration(dc *gg.Context, cardW, heroH, cardH int) {
	bolt := func(x, y, s float64, a float64) {
		dc.SetLineWidth(3)
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
		dc.SetRGBA(1, 0.9, 0.4, a*0.5)
		dc.DrawCircle(x+s*0.15, y-s*0.1, s*0.4)
		dc.Fill()
	}

	bolt(580, 100, 60, heroAlpha(0.08))
	bolt(640, 60, 35, heroAlpha(0.05))

	type line struct{ x1, y1, x2, y2 float64 }

	streaks := make([]line, 0)
	for x := 15; x < cardW; x += 25 {
		for y := 10; y < cardH; y += 30 {
			if x > 30 && x < 500 && y > 20 && y < 130 {
				continue
			}
			streaks = append(streaks, line{
				float64(x), float64(y),
				float64(x + 10), float64(y + 15),
			})
		}
	}

	dc.SetLineWidth(1.5)
	for _, s := range streaks {
		a := heroAlpha(0.10)
		if int(s.y1) > heroH {
			a = listAlpha(0.10)
		}
		dc.SetRGBA(0.45, 0.55, 0.85, a)
		dc.DrawLine(s.x1, s.y1, s.x2, s.y2)
		dc.Stroke()
	}
}
