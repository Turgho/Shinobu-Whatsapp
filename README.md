# Shinobu

[![Status](https://img.shields.io/badge/status-em%20desenvolvimento-orange?style=flat-square)](https://github.com/Turgho/Shinobu-Whatsapp)
[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-green?style=flat-square)](./LICENSE)
[![Último commit](https://img.shields.io/github/last-commit/Turgho/Shinobu-Whatsapp?style=flat-square)](https://github.com/Turgho/Shinobu-Whatsapp/commits/main)

Bot de WhatsApp escrito em Go, construído sobre **whatsmeow**. Possui router de comandos com middlewares, IA com personalidade via **Groq**, histórico de conversa por usuário, busca web via **Tavily**, gerenciamento de aniversários em grupo com scheduler diário, scheduler genérico para jobs semanais/configuráveis (áudio + @all + sticker), efeitos de áudio com **ffmpeg**, reprodução de música via servidor remoto com **yt-dlp** e sistema de figurinhas salvas.

> **Módulo Go:** `github.com/Turgho/Shinobu-Whatsapp`
> O repositório pode ser clonado como `Shinobu-Whatsapp` — use sempre o caminho do módulo nos imports.

---

## Sumário

- [Requisitos](#requisitos)
- [Instalação](#instalação)
- [Configuração](#configuração)
- [Execução](#execução)
- [Comandos](#comandos)
- [IA — Oshino Shinobu](#ia--oshino-shinobu)
- [Aniversários](#aniversários)
- [Estrutura do projeto](#estrutura-do-projeto)
- [Criando um novo comando](#criando-um-novo-comando)

---

## Requisitos

| Dependência | Detalhe |
|-------------|---------|
| **Go 1.25+** | Ver `go.mod` |
| **ffmpeg** | Necessário para `!sticker` e `!efeito`. O binário deve estar em `./bin/ffmpeg` relativo à raiz do projeto |
| **webpmux** | Necessário para injetar metadados nos stickers. Binário em `./bin/webpmux` |
| **Servidor de música** | `!play` e `!stats` dependem de `MUSIC_SERVER_URL` (servidor com yt-dlp) |
| **Groq** | Obrigatório para `!shinobu` e menções à IA |
| **Tavily** | Opcional — habilita busca web na IA |

### Instalando dependências (Linux x86-64)

```bash
./scripts/setup.sh
```

O script baixa binários estáticos de **ffmpeg**, **webpmux** e **yt-dlp** para `./bin/`. Alternativamente, instale manualmente via gerenciador de pacotes e garanta que os binários estejam em `./bin/` ou no `PATH`.

---

## Instalação

```bash
git clone https://github.com/Turgho/Shinobu-Whatsapp.git
cd Shinobu-Whatsapp
go mod tidy
```

---

## Configuração

### config.yaml

```bash
cp config.example.yaml config.yaml
```

O arquivo cobre `bot`, `database`, `log`, `usersJID`, `apiUrls` e `scheduledJobs`. Os campos podem ser sobrescritos por variáveis de ambiente via **Viper** (ex: `OWNER_JID`, `COMMAND_PREFIX`, `DB_DSN`).

### .env

```env
# Groq — obrigatório para a IA
GROQ_URL=https://api.groq.com/openai/v1/chat/completions
GROQ_API_KEY=gsk_sua_chave_aqui

# Tavily — opcional, habilita busca web
TAVILY_API_KEY=tvly-sua-chave-aqui

# Número do dono (somente dígitos)
OWNER_NUMBER=5511999999999

# Servidor de música (yt-dlp)
# POST /play  → baixa e retorna áudio
# GET  /stats → métricas do servidor remoto
MUSIC_SERVER_URL=http://seu-servidor:porta
```

> **JID do dono:** inicie o bot, envie uma mensagem e copie o JID que aparece nos logs. Cole em `usersJID.owner` no `config.yaml`.
>
> **Geocoding:** a API Nominatim do OpenStreetMap é usada como primária com `countrycodes=BR` e `addressdetails=1`. Open-Meteo Geocoding API é o fallback.
>
> **Groq:** obtenha uma API key gratuita em [console.groq.com](https://console.groq.com).
>
> **Tavily:** plano free disponível em [tavily.com](https://tavily.com).

---

## Execução

```bash
go run cmd/bot/main.go
```

Na primeira execução, escaneie o QR Code exibido no terminal. A sessão WhatsApp é persistida em SQLite conforme `database.dsn` (padrão: `storage/storage.db`). O histórico da IA usa `storage/message_history.db`.

---

## Comandos

### Públicos

| Comando | Descrição |
|---------|-----------|
| `!menu` (atalho: `!m`) | Lista todos os comandos registrados |
| `!ping` | Verifica latência e disponibilidade |
| `!clima <cidade>` (atalho: `!c`, `!tempo`) | Clima atual via Nominatim + Open-Meteo |
| `!sticker` (atalho: `!s`, `!figurinha`) | Converte imagem ou vídeo em figurinha |
| `!play <nome ou URL>` (atalho: `!p`, `!plau`) | Reproduz música via servidor remoto (yt-dlp) |
| `!efeito [nome] [intensidade]` (atalho: `!e`) | Aplica efeito em um áudio. Sem args, lista os disponíveis. Intensidades: `leve`, `medio`, `forte` |
| `!shinobu <texto>` | Conversa com a IA |
| Menção **"shinobu"** | Atalho para o mesmo handler do `!shinobu` |
| `!aniversário` (atalho: `!a`, `!aniver`) | Gerencia aniversários do grupo (ver abaixo) |
| `!mambo`, `!dio`, `!cafe` | Reproduz áudios OGG de `assets/audios/` |

> **Aliases:** erros de digitação comuns como `!plau`, `!stiker`, `!clim`, `!figurinha` e `!aniversario` (sem acento) também funcionam.

### `!aniversário` — detalhamento

| Uso | Quem pode |
|-----|-----------|
| `!aniversário DD/MM` | Qualquer membro — salva o próprio |
| `!aniversário lista` | Qualquer membro |
| `!aniversário remover` | Qualquer membro — remove o próprio |
| `!aniversário salvar @pessoa DD/MM` | Dono / admin |
| `!aniversário remover @pessoa` | Dono / admin |

### Administrativos

| Comando | Descrição |
|---------|-----------|
| `!stats` | Métricas de runtime do bot + servidor remoto |
| `!shutdown` | Encerra o processo |
| `!restart` | Reinicia o processo (mesmo PID via `syscall.Exec`) |
| `!ignorar <número>` | Ignorar/designorar mensagens de um número |
| `!ignorar lista` | Lista números ignorados |
| `!fig <nome>` | Envia figurinha salva |
| `!fig salvar <nome>` | Salva figurinha (enviar ou citar) |
| `!fig remover <nome>` | Remove figurinha salva |
| `!fig lista` | Lista figurinhas salvas |
| `!testjob [audioPath] [stickerName]` | Testa envio de áudio+@all+sticker no chat atual |

> Comandos administrativos exigem que o remetente seja owner ou admin configurado em `usersJID`.

### Efeitos de áudio disponíveis

| Efeito | Descrição |
|--------|-----------|
| `reverb` | Slowed + reverb |
| `deep` | Mais lento e grave |
| `echo` | Eco pronunciado |
| `nightcore` | Mais rápido e agudo |
| `bass` | Boost de graves |
| `lofi` | Lofi com filtro e reverb leve |

---

## IA — Oshino Shinobu

- Personalidade definida via system prompt.
- Histórico por usuário armazenado em SQLite com limpeza periódica.
- Resumo de conversa persistido entre sessões.
- Busca web via Tavily acionada automaticamente quando a pergunta exige dados atuais.
- Tom diferenciado para o owner.

### Modelos

| Uso | Modelo |
|-----|--------|
| Conversa e resumo | `meta-llama/llama-4-scout-17b-16e-instruct` |
| Resposta com contexto web | `llama-3.3-70b-versatile` |
| Classificação de busca | Scout com `MaxTokens` reduzido |

---

## Scheduler de Jobs

O pacote `internal/domain/scheduler` implementa um scheduler genérico com interface `Job`:

```go
type Job interface {
    Name() string
    Next(now time.Time) time.Time
    Run(ctx context.Context) error
}
```

O scheduler roda em uma goroutine segura (via `gosafe.Go`) com ticker de 1 minuto, executando jobs cujo `Next(now)` seja igual ou anterior ao horário atual. Cada job tem seu próprio contexto com timeout de 5 minutos e recuperação individual de panics.

### Aniversários

O `BirthdayJob` (substituiu o scheduler específico) roda todos os dias às **08:00** (horário de Brasília), notificando grupos com aniversariantes e mencionando @all.

### Jobs de Dia da Semana

O pacote `internal/domain/weekday` implementa `WeekdayJob`, configurável via `config.yaml`:

```yaml
scheduledJobs:
  - name: "sextou"
    day: "friday"          # sunday, monday, ..., saturday
    enabled: true
    hour: 10
    minute: 0
    audioPath: "assets/audios/play_tv.ogg"
    stickerName: "play_tv"  # opcional — nome salvo em !fig salvar
    targetGroups:
      - "grupo_jid@g.us"
```

O job envia sequencialmente para cada grupo:
1. Áudio OGG como nota de voz (PTT)
2. Mensagem de texto com `@all` nativo (`NonJIDMentions`)
3. Sticker salvo (se configurado e existente no store)

---

## Estrutura do projeto

```text
.
├── cmd/bot/
│   └── main.go                              # Entry point — chama app.Run()
├── config.example.yaml                      # Modelo de configuração
├── scripts
│   └── setup.sh                             # Instala ffmpeg e webpmux em ./bin/
├── go.mod                                   # Módulo: github.com/Turgho/Shinobu-Whatsapp
│
├── assets/
│   ├── audios/                              # OGGs estáticos (!mambo, !dio, !cafe, !testjob)
│   │   ├── hora_cafe.ogg
│   │   ├── mambo.ogg
│   │   ├── play_tv.ogg
│   │   └── zawarudo.ogg
│   ├── images/                              # Imagens estáticas (ex: banner do !menu)
│   │   └── shinobu_banner.png
│   ├── stickers/                            # JSON do store de figurinhas salvas
│   └── videos/                              # Vídeos estáticos (uso futuro)
│
├── storage/                                 # Gerado em runtime — não commitar
│   └── message_history.db                   # Histórico SQLite da IA por JID
│
└── internal/
    ├── app/
    │   └── app.go                           # Inicialização de deps, router, handlers, scheduler
    │
    ├── bot/
    │   ├── client.go                        # Sessão whatsmeow (QR, reconexão)
    │   └── handler.go                       # Dispatcher de eventos do WhatsApp
    │
    ├── commands/                            # Camada de comandos
    │   ├── router.go                        # Roteamento por prefixo + middlewares
    │   ├── middleware.go                    # IgnoreOld, NotFound, PrivateCommands
    │   ├── types.go                         # CommandMeta, HandlerFunc, ArgMeta
    │   ├── admin/
    │   │   ├── ignore.go                    # !ignorar — bloqueio de números
    │   │   ├── restart.go                   # !restart — syscall.Exec
    │   │   ├── save_sticker.go              # !fig — gerencia figurinhas salvas
    │   │   ├── shutdown.go                  # !shutdown
    │   │   ├── stats.go                     # !stats — runtime + servidor remoto
    │   │   └── testjob.go                   # !testjob — debug de jobs agendados
    │   └── public/
    │       ├── audio_effects.go             # !efeito — reverb, lofi, nightcore, etc.
    │       ├── birthday.go                  # !aniversário — wrapper do domain
    │       ├── bundled_audio.go             # !mambo, !dio, !cafe
    │       ├── menu.go                      # !menu — banner + lista de comandos
    │       ├── ping.go                      # !ping
    │       ├── play.go                      # !play — encaminha para servidor yt-dlp
    │       ├── shinobu.go                   # !shinobu / menção — IA com personalidade
    │       ├── sticker.go                   # !sticker — imagem/vídeo → figurinha
    │       └── weather.go                   # !clima
    │
    ├── domain/                             # Regras de negócio
    │   ├── birthday/
    │   │   ├── handler.go                   # Subcomandos do grupo (salvar, remover, lista)
    │   │   ├── job.go                       # BirthdayJob — implementa scheduler.Job (diário 08:00)
    │   │   └── store.go                     # Persistência JSON + helpers (parseDate, etc.)
    │   ├── geocoding/
    │   │   └── geocode.go                   # Nominatim (primário) + Open-Meteo (fallback)
    │   ├── history/
    │   │   ├── memory.go                    # Memória de fatos por usuário
    │   │   └── message_history.go           # Histórico por JID em SQLite (contexto da IA)
    │   ├── ia/
    │   │   ├── groq.go                      # Client HTTP Groq
    │   │   ├── ia.go                        # Orquestração: histórico, busca, resposta
    │   │   ├── keywords.go                  # Detecção de intent de busca web
    │   │   ├── models.go                    # Constantes de modelos e parâmetros
    │   │   ├── prompts.go                   # System prompts da Shinobu
    │   │   ├── search.go                    # Busca web via Tavily
    │   │   ├── summary.go                   # Resumo persistente por usuário
    │   │   ├── tavily.go                    # Client HTTP Tavily
    │   │   └── utils.go                     # Helpers internos da IA
    │   ├── ignore/
    │   │   └── store.go                     # Persistência JSON de números ignorados
    │   ├── music/
    │   │   ├── audio_effects.go             # Efeitos ffmpeg (reverb, lofi, nightcore…)
    │   │   ├── mimetype.go                  # Resolução de MIME por extensão
    │   │   └── ytdlp_request.go             # Requisição HTTP ao servidor de música
    │   ├── scheduler/
    │   │   └── scheduler.go                 # Scheduler genérico (Job interface, ticker 1min, gosafe)
    │   ├── sticker/
    │   │   ├── convert.go                   # ffmpeg → WebP + injeção de metadados EXIF
    │   │   ├── handler.go                   # Subcomandos !fig (salvar, remover, lista)
    │   │   ├── send.go                      # Envio de figurinha salva
    │   │   └── store.go                     # Persistência JSON das figurinhas
    │   ├── weekday/
    │   │   └── job.go                       # WeekdayJob — implementa scheduler.Job
    │   └── weather/
    │       ├── weather.go                   # Open-Meteo — previsão por coordenadas
    │       └── weather_code.go              # Mapeamento de códigos WMO para texto
    │
    ├── infra/                              # Infraestrutura transversal
    │   ├── configs/
    │   │   └── config.go                    # Viper + .env — carregamento de configuração
    │   ├── database/
    │   │   └── database.go                  # Conexão SQLite para o whatsmeow
    │   ├── ffmpeg/
    │   │   ├── ffmpeg_exec.go               # exec.Cmd para ./bin/ffmpeg
    │   │   └── linux_process.go             # SysProcAttr — prioridade baixa no Linux
    │   ├── gosafe/
    │   │   └── safe.go                      # Go() — goroutine com recover
    │   ├── logger/
    │   │   └── logger.go                    # Logger whatsmeow (WARN)
    │   ├── phone/
    │   │   └── normalize.go                 # Normalização de números
    │   └── uptime/
    │       └── uptime.go                    # Timestamp de início do processo (!stats)
    │
    └── integration/                        # Adaptadores WhatsApp
        ├── media/
        │   ├── doc.go                       # Documentação do pacote
        │   └── download.go                  # DownloadFromEvent — imagem, vídeo, áudio, doc
        └── whatsapp/
            ├── audio.go                     # SendBundledOggVoice, SendAudioFileToJID
            ├── context.go                   # buildContext, replyContext, mentionContext
            ├── doc.go                       # Documentação do pacote
            ├── document.go                  # SendDocument
            ├── image.go                     # SendImage + geração de thumbnail JPEG
            ├── location.go                  # SendLocation
            ├── message_text.go              # PlainTextFromProto — extrai texto da mensagem
            ├── presence.go                  # withTyping — indicador de digitação
            ├── reaction.go                  # SendReaction
            ├── reply.go                     # Reply (atalho com quote)
            ├── sticker.go                   # SendSticker, SendStickerToJID
            ├── text.go                      # SendText, SendTextWithMentions, SendTextToJID, SendAllToJID
            └── video.go                     # SendVideo
```

---

## Criando um novo comando

**1.** Crie o handler em `internal/commands/public/` ou `internal/commands/admin/`:

```go
package public

import (
    "context"

    "github.com/Turgho/Shinobu-Whatsapp/internal/integration/whatsapp"
    "go.mau.fi/whatsmeow"
    "go.mau.fi/whatsmeow/types/events"
)

func HelloCommand(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
    return whatsapp.Reply(ctx, client, evt, "Olá!")
}
```

**2.** Registre em `internal/app/app.go`:

```go
r.RegisterCommand(commands.CommandMeta{
    Name:        "hello",
    Description: "Responde com uma saudação",
    Type:        commands.CommandTypeUtility,
}, public.HelloCommand)
```

O `!menu` lista automaticamente. Para restringir a owner/admins, adicione `Private: true` no `CommandMeta`.

---

## Contato

- Autor: **Turgho** — [github.com/Turgho](https://github.com/Turgho)
- Issues: [github.com/Turgho/Shinobu-Whatsapp/issues](https://github.com/Turgho/Shinobu-Whatsapp/issues)


