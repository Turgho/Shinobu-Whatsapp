package admin

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/Turgho/YuukoWhatsapp/internal/utils"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

func StatsCommand(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
	upTime := utils.SinceUptime()

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	uptime := time.Since(upTime).Round(time.Second)

	msg := fmt.Sprintf(`
📊 *Bot Status*

⏱ Uptime: %s
🧠 Memória usada: %.2f MB
📦 Memória total: %.2f MB
🧵 Goroutines: %d
⚙ CPU cores: %d
`,
		uptime,
		float64(mem.Alloc)/1024/1024,
		float64(mem.Sys)/1024/1024,
		runtime.NumGoroutine(),
		runtime.NumCPU(),
	)

	return utils.Reply(ctx, client, evt, msg)
}
