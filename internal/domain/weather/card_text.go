package weather

import (
	"github.com/fogleman/gg"
)

// truncateToWidth reduz o texto adicionando "..." no final se ele exceder
// maxWidth com a fonte atualmente configurada no dc. Usa busca binária
// para encontrar o prefixo que cabe.
func truncateToWidth(dc *gg.Context, text string, maxWidth float64) string {
	w, _ := dc.MeasureString(text)
	if w <= maxWidth {
		return text
	}

	runes := []rune(text)
	if len(runes) <= 3 {
		return text
	}

	// reserva espaço para "..."
	dotsW, _ := dc.MeasureString("...")
	avail := maxWidth - dotsW

	lo, hi := 0, len(runes)-1
	best := 0
	for lo <= hi {
		mid := (lo + hi) / 2
		prefix := string(runes[:mid])
		pw, _ := dc.MeasureString(prefix)
		if pw <= avail {
			best = mid
			lo = mid + 1
		} else {
			hi = mid - 1
		}
	}

	return string(runes[:best]) + "..."
}

// measureWidth retorna a largura em pixels do texto com a fonte atual do dc.
func measureWidth(dc *gg.Context, text string) float64 {
	w, _ := dc.MeasureString(text)
	return w
}

// fitsWidth retorna true se o texto couber na largura máxima com a fonte atual.
func fitsWidth(dc *gg.Context, text string, maxWidth float64) bool {
	return measureWidth(dc, text) <= maxWidth
}
