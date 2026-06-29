package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/music"
	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/uptime"
	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/version"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/host"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

type remoteStats struct {
	Uptime     string  `json:"uptime"`
	CPUUsage   float64 `json:"cpu_usage"`
	CPUTemp    string  `json:"cpu_temp"`
	RAMUsed    float64 `json:"ram_used_mb"`
	RAMTotal   float64 `json:"ram_total_mb"`
	RAMPercent float64 `json:"ram_percent"`
	DiskUsed   float64 `json:"disk_used_gb"`
	DiskTotal  float64 `json:"disk_total_gb"`
	DiskFree   float64 `json:"disk_free_gb"`
	Goroutines int     `json:"goroutines"`
	CPUCores   int     `json:"cpu_cores"`
	Version    string  `json:"version"`
}

func fetchNotebookStats(musicCfg *music.Config) (*remoteStats, error) {
	client := &http.Client{Timeout: 3 * time.Second}

	req, err := http.NewRequest("GET", musicCfg.ServerURL+"/stats", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+musicCfg.APIToken)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var stats remoteStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, err
	}

	return &stats, nil
}

// StatsCommand exibe stats do sistema (CPU, RAM, temperatura, uptime, notebook remoto).
// Dados do notebook são buscados em goroutine concorrente com waitgroup.
func StatsCommand(musicCfg *music.Config) func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		var (
			notebook *remoteStats
			nbErr    error
			wg       sync.WaitGroup
		)

		wg.Add(1)
		go func() {
			defer wg.Done()
			notebook, nbErr = fetchNotebookStats(musicCfg)
		}()

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
			notebookBlock = fmt.Sprintf(
				"🏷 *Versão:* `%s`\n"+
					"⏱ *Uptime:* %s\n"+
					"🧵 *Goroutines:* %d\n"+
					"⚙ *CPU cores:* %d\n"+
					"🖥 *CPU uso:* %.1f%%\n"+
					"🌡 *CPU temp:* %s\n"+
					"💾 *RAM:* %.0f MB / %.0f MB (%.1f%%)\n"+
					"💿 *Disco:* %.1f GB livres de %.1f GB",
				notebookVersion,
				notebook.Uptime,
				notebook.Goroutines,
				notebook.CPUCores,
				notebook.CPUUsage,
				notebook.CPUTemp,
				notebook.RAMUsed, notebook.RAMTotal, notebook.RAMPercent,
				notebook.DiskFree, notebook.DiskTotal,
			)
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
