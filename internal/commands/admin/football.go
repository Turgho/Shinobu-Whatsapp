package admin

import (
	"context"
	"fmt"

	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/configs"
	"github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// FootballTestCommand sends a test goal notification for the football watcher.
func FootballTestCommand(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
	// Load configuration.
	cfg := configs.Load()
	if cfg == nil {
		return whatsapp.Reply(ctx, client, evt, "Erro ao carregar configuração.")
	}

	// Check if football is enabled.
	if !cfg.Football.Enabled {
		return whatsapp.Reply(ctx, client, evt, "O módulo de futebol está desativado.")
	}

	// Determine notify JID: use config's notify_jid if set, else fallback to owner.
	notifyJID := cfg.Football.NotifyJID
	if notifyJID == "" {
		notifyJID = cfg.UsersJID.Owner
	}
	if notifyJID == "" {
		return whatsapp.Reply(ctx, client, evt, "Nenhum JID de notificação configurado.")
	}

	// Determine a team to show in the message: first watched team if any, else generic.
	teamName := "Brasil"
	teamFlag := "🇧🇷"
	if len(cfg.Football.WatchedTeams) > 0 {
		team := cfg.Football.WatchedTeams[0]
		teamName = team.Name
		teamFlag = team.Flag
		if teamFlag == "" {
			teamFlag = "⚽"
		}
	}

	// Build a test message.
	message := fmt.Sprintf("%s%s TESTE DE GOL! %s %d x %d %s — %s aos %d'",
		teamFlag, teamFlag, teamName, 1, 0, "Adversário", "Jogador de Teste", 45)

	// Send the WhatsApp notification.
	jid, err := types.ParseJID(notifyJID)
	if err != nil {
		return whatsapp.Reply(ctx, client, evt, fmt.Sprintf("JID inválido: %v", err))
	}
	if err := whatsapp.SendTextToJID(ctx, client, jid, message, nil); err != nil {
		return whatsapp.Reply(ctx, client, evt, fmt.Sprintf("Falha ao enviar notificação de teste: %v", err))
	}

	// Confirm to the user.
	return whatsapp.Reply(ctx, client, evt, "Notificação de teste enviada com sucesso.")
}

// FootballStatusCommand shows the football watcher status and config.
func FootballStatusCommand(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
	cfg := configs.Load()
	if cfg == nil {
		return whatsapp.Reply(ctx, client, evt, "Erro ao carregar configuração.")
	}

	var sb string
	sb += "⚽ *Status do Módulo Futebol (Copa 2026)*\n\n"

	if !cfg.Football.Enabled {
		sb += "❌ *Status:* Desativado\n"
		sb += "\nUse `football.enabled: true` no config.yaml para ativar."
		return whatsapp.Reply(ctx, client, evt, sb)
	}

	sb += "✅ *Status:* Ativado\n"
	sb += fmt.Sprintf("📋 *Times monitorados:* %d\n", len(cfg.Football.WatchedTeams))

	for i, team := range cfg.Football.WatchedTeams {
		idDisplay := team.APITeamID
		if idDisplay == 0 {
			idDisplay = -1 // será auto-detectado
		}
		idStr := fmt.Sprintf("%d", idDisplay)
		if idDisplay == -1 {
			idStr = "auto-detect"
		}
		sb += fmt.Sprintf("  %d. %s %s (API Team ID: %s)\n", i+1, team.Flag, team.Name, idStr)
	}

	sb += fmt.Sprintf("📍 *Notificar em:* %s\n", cfg.Football.NotifyJID)
	sb += fmt.Sprintf("⏱️ *Intervalo idle:* %s\n", cfg.Football.PollInterval.IdleInterval)
	sb += fmt.Sprintf("⚡ *Intervalo live:* %s\n", cfg.Football.PollInterval.LiveInterval)

	if cfg.Football.APIKey == "" {
		sb += "\n⚠️ *API Key:* Não configurada (usará FOOTBALL_API_KEY env)"
	} else {
		sb += "\n🔑 *API Key:* Configurada"
	}

	sb += "\n\n💡 *Comandos:*\n"
	sb += "• `!footballtest` — Envia notificação de teste\n"
	sb += "• `!footballstatus` — Mostra este status"

	return whatsapp.Reply(ctx, client, evt, sb)
}
