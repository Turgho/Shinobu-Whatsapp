package weather

import (
	"math"

	"github.com/fogleman/gg"
)

const (
	iconSun = iota
	iconCloud
	iconRain
	iconSnow
	iconStorm
	iconFog
)

// iconCategory mapeia weather code WMO para categoria de ícone.
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

func drawSunIcon(dc *gg.Context, x, y, size float64) {
	r := size * 0.4

	// corpo do sol
	dc.SetRGBA(1, 0.9, 0.2, 1)
	dc.DrawCircle(x, y, r)
	dc.Fill()

	// raios
	dc.SetRGBA(1, 0.9, 0.2, 0.7)
	dc.SetLineWidth(size * 0.035)
	for angle := 0.0; angle < 360; angle += 45 {
		rad := gg.Radians(angle)
		inner := r * 1.3
		outer := r * 1.7
		x1 := x + inner*math.Cos(rad)
		y1 := y + inner*math.Sin(rad)
		x2 := x + outer*math.Cos(rad)
		y2 := y + outer*math.Sin(rad)
		dc.DrawLine(x1, y1, x2, y2)
	}
	dc.Stroke()

	// brilho suave ao redor
	dc.SetRGBA(1, 0.95, 0.4, 0.15)
	dc.DrawCircle(x, y, r*1.8)
	dc.Fill()
}

func drawCloudIcon(dc *gg.Context, x, y, size float64) {
	s := size * 0.5

	dc.SetRGBA(0.85, 0.85, 0.9, 1)
	dc.DrawEllipse(x-s*0.6, y+s*0.2, s*0.7, s*0.5)
	dc.Fill()
	dc.DrawEllipse(x+s*0.5, y+s*0.2, s*0.6, s*0.45)
	dc.Fill()
	dc.DrawEllipse(x+s*0.1, y-s*0.2, s*0.8, s*0.55)
	dc.Fill()
	dc.DrawEllipse(x-s*0.2, y+s*0.4, s*0.9, s*0.35)
	dc.Fill()

	// destaque leve
	dc.SetRGBA(1, 1, 1, 0.3)
	dc.DrawEllipse(x-s*0.3, y-s*0.1, s*0.5, s*0.3)
	dc.Fill()
}

func drawRainIcon(dc *gg.Context, x, y, size float64) {
	s := size * 0.5

	// nuvem cinza escura
	dc.SetRGBA(0.5, 0.5, 0.6, 1)
	dc.DrawEllipse(x-s*0.6, y-s*0.1, s*0.7, s*0.5)
	dc.Fill()
	dc.DrawEllipse(x+s*0.5, y-s*0.1, s*0.6, s*0.45)
	dc.Fill()
	dc.DrawEllipse(x+s*0.1, y-s*0.5, s*0.8, s*0.55)
	dc.Fill()
	dc.DrawEllipse(x-s*0.2, y+s*0.1, s*0.9, s*0.35)
	dc.Fill()

	// gotas
	dc.SetRGBA(0.5, 0.6, 0.9, 0.8)
	dc.SetLineWidth(size * 0.03)
	for i := -1; i <= 1; i++ {
		dx := float64(i) * s * 0.45
		dc.DrawLine(x+dx-s*0.05, y+s*0.45, x+dx+s*0.05, y+s*0.85)
	}
	dc.Stroke()
}

func drawSnowIcon(dc *gg.Context, x, y, size float64) {
	s := size * 0.5

	// nuvem clara
	dc.SetRGBA(0.8, 0.85, 0.95, 1)
	dc.DrawEllipse(x-s*0.6, y-s*0.1, s*0.7, s*0.5)
	dc.Fill()
	dc.DrawEllipse(x+s*0.5, y-s*0.1, s*0.6, s*0.45)
	dc.Fill()
	dc.DrawEllipse(x+s*0.1, y-s*0.5, s*0.8, s*0.55)
	dc.Fill()
	dc.DrawEllipse(x-s*0.2, y+s*0.1, s*0.9, s*0.35)
	dc.Fill()

	// flocos
	dc.SetRGBA(0.9, 0.92, 1, 0.9)
	dc.SetLineWidth(size * 0.025)
	for i := -1; i <= 1; i++ {
		dx := float64(i) * s * 0.45
		cx := x + dx
		cy := y + s*0.6
		fs := s * 0.08
		dc.DrawCircle(cx, cy, fs)
		dc.Fill()
		dc.DrawLine(cx-fs*1.5, cy, cx+fs*1.5, cy)
		dc.DrawLine(cx, cy-fs*1.5, cx, cy+fs*1.5)
		dc.Stroke()
	}
}

func drawStormIcon(dc *gg.Context, x, y, size float64) {
	s := size * 0.5

	// nuvem escura
	dc.SetRGBA(0.35, 0.35, 0.45, 1)
	dc.DrawEllipse(x-s*0.6, y-s*0.1, s*0.7, s*0.5)
	dc.Fill()
	dc.DrawEllipse(x+s*0.5, y-s*0.1, s*0.6, s*0.45)
	dc.Fill()
	dc.DrawEllipse(x+s*0.1, y-s*0.5, s*0.8, s*0.55)
	dc.Fill()
	dc.DrawEllipse(x-s*0.2, y+s*0.1, s*0.9, s*0.35)
	dc.Fill()

	// raio
	dc.SetRGBA(1, 0.85, 0.2, 0.9)
	dc.SetLineWidth(size * 0.035)
	lx, ly := x, y+s*0.35
	dc.MoveTo(lx-s*0.1, ly)
	dc.LineTo(lx+s*0.1, ly+s*0.2)
	dc.LineTo(lx-s*0.05, ly+s*0.2)
	dc.LineTo(lx+s*0.15, ly+s*0.55)
	dc.Stroke()
}

func drawFogIcon(dc *gg.Context, x, y, size float64) {
	s := size * 0.5

	dc.SetRGBA(0.7, 0.72, 0.78, 0.7)
	dc.SetLineWidth(size * 0.045)
	for i := -1; i <= 1; i++ {
		dy := float64(i) * s * 0.3
		dc.DrawLine(x-s*0.8, y+dy, x+s*0.8, y+dy)
	}
	dc.Stroke()
}
