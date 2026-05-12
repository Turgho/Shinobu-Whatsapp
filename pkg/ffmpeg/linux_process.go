package ffmpeg

import "syscall"

// lowPriorityProc retorna SysProcAttr com nice=10 no Linux,
// deixando o ffmpeg rodar em segundo plano sem competir com o bot.
func LowPriorityProc() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		// Nice 10: prioridade abaixo do normal, CPU disponível para o bot
		Pdeathsig: syscall.SIGTERM, // mata o ffmpeg se o processo pai morrer
	}
}
