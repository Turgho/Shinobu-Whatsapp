package admin

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/noteapi"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
	"go.uber.org/zap"
)

// ── Comandos administrativos do notebook ──────────────────────
// Todos falam com o servidor de música (MUSIC_SERVER_URL) pelos
// endpoints /stats e /admin/*, autenticados com MUSIC_API_TOKEN.

// NoteStatusCommand exibe o estado da máquina do notebook.
func NoteStatusCommand(c *noteapi.Client, logger *zap.Logger) func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		st, err := c.Status(ctx)
		if err != nil {
			logger.Error("note status falhou", zap.Error(err))
			return whatsapp.Reply(ctx, client, evt, "😴 Não consegui falar com o notebook... o túnel pode ter caído.")
		}

		msg := fmt.Sprintf(
			"🖥️ *Notebook*\n"+
				"• Uptime: %s\n"+
				"• CPU: %.1f%% (x%d) — %s\n"+
				"• RAM: %.1f/%.1f GB\n"+
				"• Disco: %.1f/%.1f GB (livre)\n"+
				"• Wi-Fi: %s (%s)\n"+
				"• Cache: %d músicas, %.2f MB",
			st.Uptime,
			st.CPUUsage, st.CPUCores, st.CPUTemp,
			st.RAMUsedMB/1024, st.RAMTotalMB/1024,
			st.DiskFreeGB, st.DiskTotalGB,
			st.WifiState, st.WifiSignal,
			st.CacheEntries, st.CacheSizeMB,
		)
		return whatsapp.Reply(ctx, client, evt, msg)
	}
}

// RestartAPICommand reinicia o servidor de música (systemd religa sozinho).
func RestartAPICommand(c *noteapi.Client, logger *zap.Logger) func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		if err := whatsapp.Reply(ctx, client, evt, "🔁 Reiniciando a API do notebook..."); err != nil {
			return fmt.Errorf("falha ao avisar antes de reiniciar a API: %w", err)
		}

		_, code, callErr := c.Action(ctx, "/admin/restart", 15*time.Second)
		if callErr != nil && code != 0 {
			logger.Error("restart da API falhou", zap.Error(callErr))
			return whatsapp.Reply(ctx, client, evt, "❌ Não consegui reiniciar a API, olha os logs do notebook.")
		}
		return whatsapp.Reply(ctx, client, evt, "✅ API reiniciando — ela volta sozinha em segundos.")
	}
}

// RestartTunnelCommand reinicia o túnel cloudflared (a URL muda; o sync
// automático propaga pro bot).
func RestartTunnelCommand(c *noteapi.Client, logger *zap.Logger) func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		if err := whatsapp.Reply(ctx, client, evt, "🔁 Reiniciando o túnel... (a URL nova é propagada em até 2 min)"); err != nil {
			return fmt.Errorf("falha ao avisar antes de reiniciar o túnel: %w", err)
		}

		_, code, callErr := c.Action(ctx, "/admin/restart-tunnel", 30*time.Second)
		if callErr != nil && code != 0 {
			logger.Error("restart do túnel falhou", zap.Error(callErr))
			return whatsapp.Reply(ctx, client, evt, "❌ Falhou o restart do túnel, olha o log do cloudflared no notebook.")
		}
		return whatsapp.Reply(ctx, client, evt, "✅ Túnel reiniciando — o sync atualiza o bot em até 2 min.")
	}
}

// ResetWifiCommand força o reset do driver Wi-Fi (mesmo ação do watchdog).
func ResetWifiCommand(c *noteapi.Client, logger *zap.Logger) func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		if err := whatsapp.Reply(ctx, client, evt, "📶 Resetando o driver Wi-Fi do notebook..."); err != nil {
			return fmt.Errorf("falha ao avisar antes do reset de Wi-Fi: %w", err)
		}

		body, code, callErr := c.Action(ctx, "/admin/wifi-recover", 30*time.Second)
		if callErr != nil && code != 0 {
			logger.Error("reset de wifi falhou", zap.Error(callErr))
			if strings.Contains(body, "sudo") {
				return whatsapp.Reply(ctx, client, evt, "⚠️ O notebook ainda não tem a regra sudo pra isso — configura o sudoers e tenta de novo.")
			}
			return whatsapp.Reply(ctx, client, evt, "❌ Reset do Wi-Fi falhou. Olha o journal do notebook.")
		}
		return whatsapp.Reply(ctx, client, evt, "✅ Driver resetado — a rede deve voltar em ~30s.")
	}
}

// RebootNotebookCommand reinicia a máquina toda.
func RebootNotebookCommand(c *noteapi.Client, logger *zap.Logger) func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		if err := whatsapp.Reply(ctx, client, evt, "🔄 Reiniciando o notebook... volta em ~1 min."); err != nil {
			return fmt.Errorf("falha ao avisar antes de reiniciar o notebook: %w", err)
		}

		body, code, callErr := c.Action(ctx, "/admin/reboot", 15*time.Second)
		// Conexão morta aqui é esperado: o reboot derruba o túnel.
		if callErr != nil && code != 0 && !strings.Contains(body, "sudo") {
			logger.Error("reboot do notebook falhou", zap.Error(callErr))
			return whatsapp.Reply(ctx, client, evt, "❌ Não consegui reiniciar o notebook (a API respondeu?).")
		}
		if strings.Contains(body, "sudo") {
			return whatsapp.Reply(ctx, client, evt, "⚠️ Falta a regra sudo no notebook pra reboot via API.")
		}
		return whatsapp.Reply(ctx, client, evt, "✅ Reboot disparado.")
	}
}

// ClearCacheCommand limpa o cache de músicas no notebook.
func ClearCacheCommand(c *noteapi.Client, logger *zap.Logger) func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
	return func(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
		if err := whatsapp.Reply(ctx, client, evt, "🧹 Limpando o cache de músicas..."); err != nil {
			return fmt.Errorf("falha ao avisar antes de limpar o cache: %w", err)
		}

		body, err := c.ClearCache(ctx)
		if err != nil {
			logger.Error("limpeza de cache falhou", zap.Error(err), zap.String("body", body))
			return whatsapp.Reply(ctx, client, evt, "❌ Falhou a limpeza do cache.")
		}
		return whatsapp.Reply(ctx, client, evt, "✅ Cache removido — próximas músicas baixam de novo.")
	}
}
