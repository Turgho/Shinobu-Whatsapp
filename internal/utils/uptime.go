package utils

import "time"

// start armazena o momento em que o bot foi iniciado
var start time.Time

// Start registra o início do bot (chame no main)
func StartUptime() {
	start = time.Now()
}

// Get retorna o tempo de execução desde o início
func GetUptime() time.Duration {
	if start.IsZero() {
		// Caso não tenha sido inicializado ainda
		return 0
	}
	return time.Since(start)
}

// Since retorna o horário de início
func SinceUptime() time.Time {
	return start
}
