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

// FootballTestCommand envia uma notificação de gol de teste para o JID configurado.
// Uso: !footballtest
func FootballTestCommand(ctx context.Context, client *whatsmeow.Client, evt *events.Message, _ []string) error {
	cfg := configs.Load()
	if cfg == nil {
		return whatsapp.Reply(ctx, client, evt, "Erro ao carregar configuração.")
	}

	if !cfg.Football.Enabled {
		return whatsapp.Reply(ctx, client, evt, "O módulo de futebol está desativado.")
	}

	// Usa notify_jid do config; fallback para o owner se não configurado.
	notifyJID := cfg.Football.NotifyJID
	if notifyJID == "" {
		notifyJID = cfg.UsersJID.Owner
	}
	if notifyJID == "" {
		return whatsapp.Reply(ctx, client, evt, "Nenhum JID de notificação configurado.")
	}

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

	message := fmt.Sprintf("%s%s TESTE DE GOL! %s %d x %d %s — %s aos %d'",
		teamFlag, teamFlag, teamName, 1, 0, "Adversário", "Jogador de Teste", 45)

	jid, err := types.ParseJID(notifyJID)
	if err != nil {
		return whatsapp.Reply(ctx, client, evt, fmt.Sprintf("JID inválido: %v", err))
	}

	if err := whatsapp.SendTextToJID(ctx, client, jid, message, nil); err != nil {
		return whatsapp.Reply(ctx, client, evt, fmt.Sprintf("Falha ao enviar notificação de teste: %v", err))
	}

	return whatsapp.Reply(ctx, client, evt, fmt.Sprintf("✅ Notificação de teste enviada para %s.", notifyJID))
}

// FootballStatusCommand exibe o status e configuração atual do watcher de futebol.
// Uso: !footballstatus
func FootballStatusCommand(ctx context.Context, client *whatsmeow.Client, evt *events.Message, _ []string) error {
	cfg := configs.Load()
	if cfg == nil {
		return whatsapp.Reply(ctx, client, evt, "Erro ao carregar configuração.")
	}

	if !cfg.Football.Enabled {
		return whatsapp.Reply(ctx, client, evt,
			"⚽ *Status do Módulo Futebol (Copa 2026)*\n\n❌ *Status:* Desativado\n\nUse `football.enabled: true` no config.yaml para ativar.")
	}

	apiKeyStatus := "🔑 *API Key:* Configurada"
	if cfg.Football.APIKey == "" {
		apiKeyStatus = "⚠️ *API Key:* Não configurada"
	}

	teamsStr := ""
	for i, team := range cfg.Football.WatchedTeams {
		idStr := fmt.Sprintf("%d", team.APITeamID)
		if team.APITeamID == 0 {
			idStr = "auto-detect"
		}
		flag := team.Flag
		if flag == "" {
			flag = "⚽"
		}
		teamsStr += fmt.Sprintf("  %d. %s %s (ID: %s)\n", i+1, flag, team.Name, idStr)
	}

	msg := fmt.Sprintf(
		"⚽ *Status do Módulo Futebol (Copa 2026)*\n\n"+
			"✅ *Status:* Ativado\n"+
			"📋 *Times monitorados:* %d\n%s"+
			"📍 *Notificar em:* %s\n"+
			"⏱️ *Intervalo idle:* %s\n"+
			"⚡ *Intervalo live:* %s\n"+
			"%s\n\n"+
			"💡 *Comandos:*\n"+
			"• `!footballtest` — Envia notificação de teste\n"+
			"• `!footballstatus` — Mostra este status",
		len(cfg.Football.WatchedTeams),
		teamsStr,
		cfg.Football.NotifyJID,
		cfg.Football.PollInterval.IdleInterval,
		cfg.Football.PollInterval.LiveInterval,
		apiKeyStatus,
	)

	return whatsapp.Reply(ctx, client, evt, msg)
}
