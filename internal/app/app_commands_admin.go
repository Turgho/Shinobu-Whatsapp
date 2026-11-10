// Registro dos comandos administrativos/owner do bot (!stats, !note, !fig, ...).
//
// Os handlers vivem em internal/commands/admin/; aqui só monta dependências
// (noteapi para o notebook) e registra no Router.
package app

import (
	"github.com/Turgho/Shinobu-Whatsapp/internal/commands"
	"github.com/Turgho/Shinobu-Whatsapp/internal/commands/admin"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/history"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/ignore"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/noteapi"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/scheduler"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/sticker"
	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/configs"
	"go.mau.fi/whatsmeow"
	"go.uber.org/zap"
)

func registerAdminCommands(r *commands.Router, cfg *configs.Config, store *history.Store, logger *zap.Logger, ignoreStore *ignore.Store, stickerStore *sticker.Store, waClient *whatsmeow.Client, sched *scheduler.Scheduler) {
	// Cliente dos endpoints administrativos do notebook (mesma base e token
	// do fallback de download: MUSIC_SERVER_URL / API_AUTH_TOKEN).
	noteClient := noteapi.NewClient(cfg.Music.ServerURL, cfg.Music.APIToken, logger)

	r.RegisterCommand(commands.CommandMeta{
		Name:        "stats",
		Description: "Exibe estatísticas de runtime do bot",
		Type:        commands.CommandTypeOwner,
		Private:     true,
	}, admin.StatsCommand(noteClient, r, logger))

	r.RegisterCommand(commands.CommandMeta{
		Name:        "shutdown",
		Description: "Desliga o bot",
		Type:        commands.CommandTypeOwner,
		Private:     true,
	}, admin.ShutdownCommand(r.RequestShutdown))

	r.RegisterCommand(commands.CommandMeta{
		Name:        "restart",
		Description: "Reinicia o bot",
		Type:        commands.CommandTypeOwner,
		Private:     true,
	}, admin.RestartCommand)

	r.RegisterCommand(commands.CommandMeta{
		Name:        "note",
		Description: "Status da máquina do notebook (RAM, disco, Wi-Fi, cache)",
		Type:        commands.CommandTypeOwner,
		Private:     true,
	}, admin.NoteStatusCommand(noteClient, logger))

	r.RegisterCommand(commands.CommandMeta{
		Name:        "reiniciar_api",
		Description: "Reinicia o servidor de música no notebook",
		Type:        commands.CommandTypeOwner,
		Private:     true,
	}, admin.RestartAPICommand(noteClient, logger))

	r.RegisterCommand(commands.CommandMeta{
		Name:        "reiniciar_tunel",
		Description: "Reinicia o túnel Cloudflare do notebook",
		Type:        commands.CommandTypeOwner,
		Private:     true,
	}, admin.RestartTunnelCommand(noteClient, logger))

	r.RegisterCommand(commands.CommandMeta{
		Name:        "resetar_wifi",
		Description: "Força reset do driver Wi-Fi no notebook",
		Type:        commands.CommandTypeOwner,
		Private:     true,
	}, admin.ResetWifiCommand(noteClient, logger))

	r.RegisterCommand(commands.CommandMeta{
		Name:        "reiniciar_note",
		Description: "Reinicia a máquina do notebook",
		Type:        commands.CommandTypeOwner,
		Private:     true,
	}, admin.RebootNotebookCommand(noteClient, logger))

	r.RegisterCommand(commands.CommandMeta{
		Name:        "limpar_cache",
		Description: "Limpa o cache de músicas no notebook",
		Type:        commands.CommandTypeOwner,
		Private:     true,
	}, admin.ClearCacheCommand(noteClient, logger))

	r.RegisterCommand(commands.CommandMeta{
		Name:        "fig",
		Description: "Gerencia stickers salvos. Uso em DM: !fig salvar <nome>, !fig remover <nome>, !fig lista. Uso normal: !fig <nome>",
		Type:        commands.CommandTypeAdmin,
		Args: []commands.ArgMeta{
			{Name: "nome", Required: false},
		},
		Private: true,
	}, admin.SaveStickerCommand(cfg.Owner, stickerStore))

	r.RegisterCommand(commands.CommandMeta{
		Name:        "ignorar",
		Description: "Ignorar mensagens de un número",
		Type:        commands.CommandTypeAdmin,
		Private:     true,
	}, admin.IgnoreCommand(ignoreStore, logger))

	r.RegisterCommand(commands.CommandMeta{
		Name:        "testjob",
		Description: "Testa o job semanal (audio+@all+sticker) no chat atual",
		Type:        commands.CommandTypeOwner,
		Private:     true,
	}, admin.TestJobCommand(stickerStore))

	r.RegisterCommand(commands.CommandMeta{
		Name:        "manutencao",
		Description: "Ativa/desativa modo manutenção (comandos bloqueados)",
		Type:        commands.CommandTypeOwner,
		Private:     true,
	}, admin.ManutencaoCommand(r))

	r.RegisterCommand(commands.CommandMeta{
		Name:        "memoria",
		Description: "Gerencia a memória da IA no chat. Subcomandos: ver, limpar [@user], resumo",
		Type:        commands.CommandTypeAdmin,
		Private:     true,
	}, admin.MemoriaHandler(store, logger))

	// --- Comandos de debug ---

	r.RegisterCommand(commands.CommandMeta{
		Name:        "whois",
		Description: "Informações de JID/LID de um contato",
		Type:        commands.CommandTypeAdmin,
		Private:     true,
	}, admin.WhoisHandler(waClient))

	r.RegisterCommand(commands.CommandMeta{
		Name:        "rawmsg",
		Description: "Dump da mensagem crua do whatsmeow (debug)",
		Type:        commands.CommandTypeAdmin,
		Private:     true,
	}, admin.RawMsgHandler)

	r.RegisterCommand(commands.CommandMeta{
		Name:        "chatinfo",
		Description: "Informações do chat atual",
		Type:        commands.CommandTypeAdmin,
		Private:     true,
	}, admin.ChatInfoHandler(waClient, logger))

	r.RegisterCommand(commands.CommandMeta{
		Name:        "rawevent",
		Description: "Dump do evento completo do whatsmeow (debug)",
		Type:        commands.CommandTypeAdmin,
		Private:     true,
	}, admin.RawEventHandler)

	r.RegisterCommand(commands.CommandMeta{
		Name:        "groupjid",
		Description: "JID do grupo atual",
		Type:        commands.CommandTypeAdmin,
		Private:     true,
	}, admin.GroupJIDHandler)

	r.RegisterCommand(commands.CommandMeta{
		Name:        "jobs",
		Description: "Gerencia jobs do scheduler. Subcomandos: (lista), forcar <nome>",
		Type:        commands.CommandTypeOwner,
		Private:     true,
	}, admin.JobsHandler(sched, logger))
}
