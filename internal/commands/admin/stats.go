package admin

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/Turgho/YuukoWhatsapp/internal/utils"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/host"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

func StatsCommand(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
	uptime := time.Since(utils.SinceUptime()).Round(time.Second)

	cpuPercent, _ := cpu.Percent(time.Second, false)
	cpuUsage := 0.0
	if len(cpuPercent) > 0 {
		cpuUsage = cpuPercent[0]
	}

	// Temperatura da CPU
	cpuTemp := "N/A"
	temps, err := host.SensorsTemperatures()
	if err == nil {
		for _, t := range temps {
			if t.Temperature > 0 {
				cpuTemp = fmt.Sprintf("%.1f°C", t.Temperature)
				break
			}
		}
	}

	var botMem runtime.MemStats
	runtime.ReadMemStats(&botMem)

	msg := fmt.Sprintf(`
📊 *Bot Status*
⏱ *Uptime:* %s
🧵 *Goroutines:* %d
⚙ *CPU cores:* %d
🖥 *CPU uso:* %.1f%%
🌡 *CPU temp:* %s
📦 *RAM bot:* %.2f MB
`,
		uptime,
		runtime.NumGoroutine(),
		runtime.NumCPU(),
		cpuUsage,
		cpuTemp,
		float64(botMem.Alloc)/1024/1024,
	)

	return utils.Reply(ctx, client, evt, msg)
}
