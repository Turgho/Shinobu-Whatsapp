// Package uptime regista o instante de arranque do processo (para !stats e filtro de mensagens antigas).
package uptime

import "time"

var start time.Time

// Start marca o início do bot (chamar uma vez no arranque, ex. em app.Run).
func Start() {
	start = time.Now()
}

// Duration devolve o tempo desde Start(); zero se Start ainda não foi chamado.
func Duration() time.Duration {
	if start.IsZero() {
		return 0
	}
	return time.Since(start)
}

// ProcessStartTime devolve o time.Time gravado em Start (zero se não inicializado).
func ProcessStartTime() time.Time {
	return start
}
