package admin

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/noteapi"
	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/gosafe"
	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/uptime"
	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/version"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/host"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
	"go.uber.org/zap"
)

// StatsCommand exibe stats do sistema (CPU, RAM, temperatura, uptime, notebook remoto).
// Dados do notebook vêm do noteapi.Client (mesma fonte usada pelos comandos !note/*).
// Buscados em goroutine concorrente com waitgroup.
func StatsCommand(note *noteapi.Client, log *zap.Logger) func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		var (
			notebook *noteapi.StatusRef
			nbErr    error
			wg       sync.WaitGroup
		)

		wg.Add(1)
		gosafe.Go(log, func() {
			defer wg.Done()
			notebook, nbErr = note.Status(ctx)
		})

		uptimeStr := time.Since(uptime.ProcessStartTime()).Round(time.Second)
		cpuPercent, _ := cpu.Percent(time.Second, false)
		cpuUsage := 0.0
		if len(cpuPercent) > 0 {
			cpuUsage = cpuPercent[0]
		}

		cpuTemp := "N/A"
		if temps, err := host.SensorsTemperatures(); err == nil {
			for _, t := range temps {
				if t.Temperature > 0 {
					cpuTemp = fmt.Sprintf("%.1f°C", t.Temperature)
					break
				}
			}
		}

		var botMem runtime.MemStats
		runtime.ReadMemStats(&botMem)

		wg.Wait()

		notebookBlock := ""
		if nbErr != nil {
			notebookBlock = "❌ *Offline / sem resposta*"
		} else {
			notebookVersion := notebook.Version
			if notebookVersion == "" {
				notebookVersion = "unknown"
			}
			stat := fmt.Sprintf(
				"🏷 *Versão:* `%s`\n"+
					"⏱ *Uptime:* %s\n"+
					"🧵 *Goroutines:* %d\n"+
					"⚙ *CPU cores:* %d\n"+
					"🖥 *CPU uso:* %.1f%%\n"+
					"🌡 *CPU temp:* %s\n"+
					"💾 *RAM:* %.0f MB / %.0f MB\n"+
					"💿 *Disco:* %.1f GB livres de %.1f GB",
				notebookVersion,
				notebook.Uptime,
				notebook.Goroutines,
				notebook.CPUCores,
				notebook.CPUUsage,
				notebook.CPUTemp,
				notebook.RAMUsedMB, notebook.RAMTotalMB,
				notebook.DiskFreeGB, notebook.DiskTotalGB,
			)
			notebookBlock = stat
		}

		msg := fmt.Sprintf(
			"📊 *Bot Status*\n\n"+
				"☁️ *Square Cloud*\n"+
				"🏷 *Versão:* `%s`\n"+
				"⏱ *Uptime:* %s\n"+
				"🧵 *Goroutines:* %d\n"+
				"⚙ *CPU cores:* %d\n"+
				"🖥 *CPU uso:* %.1f%%\n"+
				"🌡 *CPU temp:* %s\n"+
				"📦 *RAM bot:* %.2f MB\n\n"+
				"——————————————\n"+
				"🖥 *Notebook (yt-dlp)*\n"+
				"%s",
			version.Version,
			uptimeStr,
			runtime.NumGoroutine(),
			runtime.NumCPU(),
			cpuUsage,
			cpuTemp,
			float64(botMem.Alloc)/1024/1024,
			notebookBlock,
		)

		return whatsapp.Reply(ctx, client, evt, msg)
	}
}
