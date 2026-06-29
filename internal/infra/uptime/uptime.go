// Package uptime regista o instante de arranque do processo (para !stats e filtro de mensagens antigas).
package uptime

import "time"

var start time.Time

// Start marca o início do bot (chamar uma vez no arranque, ex. em app.Run).
func Start() {
	start = time.Now()
}

// ProcessStartTime devolve o time.Time gravado em Start (zero se não inicializado).
func ProcessStartTime() time.Time {
	return start
}
