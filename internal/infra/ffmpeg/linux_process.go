package ffmpeg

import (
	"os/exec"
	"syscall"
)

// LowPriorityProc retorna SysProcAttr com Pdeathsig=SIGTERM no Linux.
// O nice+10 é aplicado em RunLowPriority após o fork.
func LowPriorityProc() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Pdeathsig: syscall.SIGTERM,
	}
}

// RunLowPriority inicia cmd com prioridade baixa (nice+10) e espera terminar.
// Equivalente a cmd.Run() mas ajusta o nice do processo filho após o start.
func RunLowPriority(cmd *exec.Cmd) error {
	// Não sobrescreve o SysProcAttr do caller: quem define Setpgid (para
	// matar o grupo de processos) perde o Pdeathsig se fosse sobrescrito.
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = LowPriorityProc()
	} else {
		cmd.SysProcAttr.Pdeathsig = syscall.SIGTERM
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	// Ajusta nice depois do fork mas antes do Wait.
	// syscall.SysProcAttr não expõe Nice neste Go, então fazemos manual.
	syscall.Setpriority(syscall.PRIO_PROCESS, cmd.Process.Pid, 10)
	return cmd.Wait()
}
