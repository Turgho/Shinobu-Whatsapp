package app

import (
	"net/http"
	"time"

	"github.com/Turgho/Shinobu-Whatsapp/internal/commands"
	"github.com/Turgho/Shinobu-Whatsapp/internal/commands/admin"
	"github.com/Turgho/Shinobu-Whatsapp/internal/commands/public"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/cotacao"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/feriado"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/geocoding"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/history"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/ia"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/ignore"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/mikael"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/music"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/scheduler"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/sticker"
	"github.com/Turgho/Shinobu-Whatsapp/internal/domain/weather"
	"github.com/Turgho/Shinobu-Whatsapp/internal/infra/configs"
	"go.mau.fi/whatsmeow"
	"go.uber.org/zap"
)

func registerPublicCommands(r *commands.Router, cfg *configs.Config, logger *zap.Logger, store *history.Store, sched *scheduler.Scheduler, dynStore *scheduler.DynamicStore, stickerStore *sticker.Store, mikaelStore *mikael.Store) {
	geoClient := geocoding.NewGeoCoding(cfg.ApiURLs.Geocoding, cfg.ApiURLs.OpenMeteoGeo, logger.Named("GEOCODING"))
	weatherClient := weather.NewWeatherClient(cfg.ApiURLs.Weather, logger.Named("WEATHER"))

	r.RegisterCommand(commands.CommandMeta{
		Name:        "menu",
		Description: "Lista todos os comandos disponíveis",
		Type:        commands.CommandTypeUtility,
	}, public.MenuCommand(r))

	r.RegisterCommand(commands.CommandMeta{
		Name:        "ping",
		Description: "Verifica se o bot está online",
		Type:        commands.CommandTypeUtility,
	}, public.PingCommand)

	r.RegisterCommand(commands.CommandMeta{
		Name:        "clima",
		Description: "Mostra o clima atual de uma cidade",
		Type:        commands.CommandTypeUtility,
		Args:        []commands.ArgMeta{{Name: "cidade", Required: true}},
	}, public.WeatherHandler(geoClient, weatherClient, logger))

	r.RegisterBatchCommand(commands.CommandMeta{
		Name:        "sticker",
		Description: "Gera uma figurinha com base em uma imagem ou vídeo (suporta albums)",
		Type:        commands.CommandTypeUtility,
	}, public.StickerCommand, public.StickerAlbumCommand)

	musicCfg := &music.Config{
		ServerURL: cfg.Music.ServerURL,
		APIToken:  cfg.Music.APIToken,
	}

	r.RegisterCommand(commands.CommandMeta{
		Name:        "play",
		Description: "Busca por uma música via nome ou URL",
		Type:        commands.CommandTypeDownload,
		Args:        []commands.ArgMeta{{Name: "nome da música ou URL", Required: true}},
	}, public.PlayCommand(musicCfg, logger))

	r.RegisterCommand(commands.CommandMeta{
		Name:        "mambo",
		Description: "M A M B O 🏇",
		Type:        commands.CommandTypeFun,
	}, public.FixedBundledAudioCommand("assets/audios/mambo.ogg", "", stickerStore))

	r.RegisterCommand(commands.CommandMeta{
		Name:        "dio",
		Description: "Talvez o tempo pare...",
		Type:        commands.CommandTypeFun,
	}, public.FixedBundledAudioCommand("assets/audios/zawarudo.ogg", "zawarudo", stickerStore))

	r.RegisterCommand(commands.CommandMeta{
		Name:        "cafe",
		Description: "Não importa a hora!",
		Type:        commands.CommandTypeFun,
	}, public.FixedBundledAudioCommand("assets/audios/hora_cafe.ogg", "hora_cafe", stickerStore))

	r.RegisterCommand(commands.CommandMeta{
		Name:        "shinobu",
		Description: "converse com shinobu",
		Type:        commands.CommandTypeAI,
		Args:        []commands.ArgMeta{{Name: "escreva algo", Required: false}},
	}, public.ShinobuCommand(store, cfg, stickerStore))

	r.RegisterCommand(commands.CommandMeta{
		Name:        "aniversário",
		Description: "Gerencia aniversário de grupos",
		Type:        commands.CommandTypeGroup,
	}, public.BirthdayCommand(cfg.UsersJID.Owner, cfg.UsersJID.Admins))

	r.RegisterCommand(commands.CommandMeta{
		Name:        "efeito",
		Description: "Aplica efeitos em um áudio. Use !efeito para ver os disponíveis.",
		Type:        commands.CommandTypeMedia,
		Args: []commands.ArgMeta{
			{Name: "efeito", Required: false},
			{Name: "intensidade", Required: false},
		},
	}, public.AudioEffectsCommand)

	cotacaoClient := cotacao.NewCotacaoClient(cfg.ApiURLs.Cotacao, logger.Named("COTACAO"))

	feriadoClient := feriado.NewFeriadosClient(cfg.ApiURLs.Feriado, cfg.Feriados.APIKey, logger.Named("FERIADO"))

	r.RegisterCommand(commands.CommandMeta{
		Name:        "cotacao",
		Description: "Cotação do dólar e euro em reais",
		Type:        commands.CommandTypeUtility,
		Private:     false,
	}, public.CotacaoHandler(cotacaoClient))

	r.RegisterCommand(commands.CommandMeta{
		Name:        "feriado",
		Description: "Próximos feriados nacionais ou estaduais (!feriado SP)",
		Type:        commands.CommandTypeUtility,
		Private:     false,
	}, public.FeriadoHandler(feriadoClient))

	aiCfg := &ia.Config{
		GroqURL:    cfg.Groq.URL,
		GroqKey:    cfg.Groq.APIKey,
		TavilyKey:  cfg.Tavily.APIKey,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}

	loc, errLoc := time.LoadLocation(cfg.Bot.Timezone)
	if errLoc != nil {
		logger.Warn("timezone inválida, usando Local", zap.String("timezone", cfg.Bot.Timezone), zap.Error(errLoc))
		loc = time.Local
	}

	r.RegisterCommand(commands.CommandMeta{
		Name:        "noticia",
		Description: "Principais notícias do dia",
		Type:        commands.CommandTypeUtility,
		Private:     false,
	}, public.NoticiaHandler(aiCfg, loc))

	r.RegisterCommand(commands.CommandMeta{
		Name:        "receita",
		Description: "Busca uma receita",
		Type:        commands.CommandTypeUtility,
		Args:        []commands.ArgMeta{{Name: "prato", Required: true}},
		Private:     false,
	}, public.ReceitaHandler(aiCfg))

	r.RegisterCommand(commands.CommandMeta{
		Name:        "piada",
		Description: "Conta uma piada",
		Type:        commands.CommandTypeFun,
		Private:     false,
	}, public.PiadaHandler(aiCfg))

	r.RegisterCommand(commands.CommandMeta{
		Name:        "fato",
		Description: "Compartilha um fato curioso",
		Type:        commands.CommandTypeFun,
		Private:     false,
	}, public.FatoHandler(aiCfg))

	r.RegisterCommand(commands.CommandMeta{
		Name:        "filme",
		Description: "Recomenda um filme",
		Type:        commands.CommandTypeFun,
		Args:        []commands.ArgMeta{{Name: "gênero", Required: false}},
		Private:     false,
	}, public.FilmeHandler(aiCfg))

	r.RegisterCommand(commands.CommandMeta{
		Name:        "contagem",
		Description: "Conta os dias para uma data",
		Type:        commands.CommandTypeUtility,
		Args: []commands.ArgMeta{
			{Name: "evento", Required: true},
			{Name: "data", Required: true},
		},
		Private: false,
	}, public.ContagemHandler(loc))

	r.RegisterCommand(commands.CommandMeta{
		Name:        "unsticker",
		Description: "Converte figurinha de volta em imagem",
		Type:        commands.CommandTypeMedia,
		Private:     false,
	}, public.UnstickerHandler())

	r.RegisterCommand(commands.CommandMeta{
		Name:        "traduz",
		Description: "Traduz texto para português",
		Type:        commands.CommandTypeUtility,
		Args:        []commands.ArgMeta{{Name: "texto", Required: false}},
		Private:     false,
	}, public.TraduzHandler(aiCfg))

	r.RegisterCommand(commands.CommandMeta{
		Name:        "agenda",
		Description: "Agenda um lembrete. Ex: agenda 2026-06-28T09:00 tomar remédio",
		Type:        commands.CommandTypeUtility,
		Args: []commands.ArgMeta{
			{Name: "DD/MM", Required: true},
			{Name: "mensagem", Required: true},
		},
	}, public.AgendaHandler(sched, dynStore, logger, loc))

	r.RegisterCommand(commands.CommandMeta{
		Name:        "mikael",
		Description: "Conta quantas vezes o Mikael escreveu 'pix' no grupo",
		Type:        commands.CommandTypeFun,
	}, public.MikaelHandler(mikaelStore, logger))
}

func registerAdminCommands(r *commands.Router, cfg *configs.Config, store *history.Store, logger *zap.Logger, ignoreStore *ignore.Store, stickerStore *sticker.Store, waClient *whatsmeow.Client, sched *scheduler.Scheduler) {
	musicCfg := &music.Config{
		ServerURL: cfg.Music.ServerURL,
		APIToken:  cfg.Music.APIToken,
	}

	r.RegisterCommand(commands.CommandMeta{
		Name:        "stats",
		Description: "Exibe estatísticas de runtime do bot",
		Type:        commands.CommandTypeOwner,
		Private:     true,
	}, admin.StatsCommand(musicCfg, logger))

	r.RegisterCommand(commands.CommandMeta{
		Name:        "shutdown",
		Description: "Desliga o bot",
		Type:        commands.CommandTypeOwner,
		Private:     true,
	}, admin.ShutdownCommand)

	r.RegisterCommand(commands.CommandMeta{
		Name:        "restart",
		Description: "Reinicia o bot",
		Type:        commands.CommandTypeOwner,
		Private:     true,
	}, admin.RestartCommand)

	r.RegisterCommand(commands.CommandMeta{
		Name:        "fig",
		Description: "Gerencia stickers salvos. Uso em DM: !fig salvar <nome>, !fig remover <nome>, !fig lista. Uso normal: !fig <nome>",
		Type:        commands.CommandTypeAdmin,
		Args: []commands.ArgMeta{
			{Name: "nome", Required: false},
		},
		Private: true,
	}, admin.SaveStickerCommand(cfg.UsersJID.Owner, stickerStore))

	r.RegisterCommand(commands.CommandMeta{
		Name:        "ignorar",
		Description: "Ignorar mensagens de un número",
		Type:        commands.CommandTypeAdmin,
		Private:     true,
	}, admin.IgnoreCommand(ignoreStore))

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
	}, admin.ChatInfoHandler(waClient))

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
