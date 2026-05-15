# Shinobu

[![Status](https://img.shields.io/badge/status-em%20desenvolvimento-orange?style=flat-square)](https://github.com/Turgho/Shinobu-Whatsapp)
[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-green?style=flat-square)](./LICENSE)
[![Último commit](https://img.shields.io/github/last-commit/Turgho/Shinobu-Whatsapp?style=flat-square)](https://github.com/Turgho/Shinobu-Whatsapp/commits/main)

Bot de WhatsApp escrito em Go, construído sobre **whatsmeow**. Possui router de comandos com middlewares, IA com personalidade via **Groq**, histórico de conversa por usuário, busca web via **Tavily**, gerenciamento de aniversários em grupo, efeitos de áudio com **ffmpeg** e reprodução de música via servidor remoto com **yt-dlp**.

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

### Instalando ffmpeg

```bash
# Ubuntu / Debian
sudo apt install ffmpeg

# Fedora (requer RPM Fusion)
sudo dnf install https://mirrors.rpmfusion.org/free/fedora/rpmfusion-free-release-$(rpm -E %fedora).noarch.rpm
sudo dnf install ffmpeg

# Arch / Manjaro / CachyOS
sudo pacman -S ffmpeg

# macOS
brew install ffmpeg
```

O pacote **libwebp** geralmente inclui o `webpmux`. Copie ou crie um link simbólico para `./bin/webpmux` caso não esteja no `PATH` global.

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

O arquivo cobre `bot`, `database`, `log`, `usersJID` e `apiUrls`. Os campos podem ser sobrescritos por variáveis de ambiente via **Viper** (ex: `OWNER_JID`, `COMMAND_PREFIX`, `DB_DSN`).

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
| `!menu` | Lista todos os comandos registrados |
| `!ping` | Verifica latência e disponibilidade |
| `!clima <cidade>` | Clima atual via Nominatim + Open-Meteo |
| `!sticker` | Converte imagem ou vídeo em figurinha |
| `!play <nome ou URL>` | Reproduz música via servidor remoto (yt-dlp) |
| `!efeito [nome] [intensidade]` | Aplica efeito em um áudio. Sem args, lista os disponíveis. Intensidades: `leve`, `medio`, `forte` |
| `!shinobu <texto>` | Conversa com a IA |
| Menção **"shinobu"** | Atalho para o mesmo handler do `!shinobu` |
| `!aniversário` | Gerencia aniversários do grupo (ver abaixo) |
| `!mambo`, `!dio`, `!cafe` | Reproduz áudios OGG de `assets/audios/` |

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
| `!fig <nome>` | Envia figurinha salva |
| `!fig salvar <nome>` | Salva figurinha (enviar ou citar) |
| `!fig remover <nome>` | Remove figurinha salva |
| `!fig lista` | Lista figurinhas salvas |

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

## Aniversários

Dados persistidos em JSON pelo pacote `internal/domain/birthday`.

O scheduler roda em background e notifica os grupos com aniversariantes todos os dias às **08:00** (horário local do processo), mencionando os aniversariantes e todos do grupo.

---

## Estrutura do projeto

```text
.
├── cmd/bot/
│   └── main.go                              # Entry point — chama app.Run()
├── config.example.yaml                      # Modelo de configuração
├── scripts
│   └── setup.sh                             # Instala ffmpeg e webpmux em ./bin/
├── go.mod                                   # Módulo: github.com/Turgho/YuukoWhatsapp
│
├── assets/
│   ├── audios/                              # OGGs estáticos (!mambo, !dio, !cafe)
│   │   ├── hora_cafe.ogg
│   │   ├── mambo.ogg
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
    │   └── app.go                           # Inicialização de deps, router e handlers
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
    │   │   ├── save_sticker.go              # !fig — gerencia figurinhas salvas
    │   │   ├── shutdown.go                  # !shutdown
    │   │   └── stats.go                     # !stats — runtime + servidor remoto
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
    │   │   ├── scheduler.go                 # Loop diário às 08:00 — notifica grupos
    │   │   └── store.go                     # Persistência JSON + helpers (parseDate, etc.)
    │   ├── geocoding/
    │   │   └── geocode.go                   # Nominatim — coordenadas por nome de cidade
    │   ├── history/
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
    │   ├── music/
    │   │   ├── audio_effects.go             # Efeitos ffmpeg (reverb, lofi, nightcore…)
    │   │   ├── mimetype.go                  # Resolução de MIME por extensão
    │   │   └── ytdlp_request.go             # Requisição HTTP ao servidor de música
    │   ├── sticker/
    │   │   ├── convert.go                   # ffmpeg → WebP + injeção de metadados EXIF
    │   │   ├── handler.go                   # Subcomandos !fig (salvar, remover, lista)
    │   │   ├── send.go                      # Envio de figurinha salva
    │   │   └── store.go                     # Persistência JSON das figurinhas
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
    │   ├── logger/
    │   │   └── logger.go                    # Zap — configuração de log
    │   └── uptime/
    │       └── uptime.go                    # Timestamp de início do processo (!stats)
    │
    └── integration/                        # Adaptadores WhatsApp
        ├── media/
        │   ├── doc.go                       # Documentação do pacote
        │   └── download.go                  # DownloadFromEvent — imagem, vídeo, áudio, doc
        └── whatsapp/
            ├── audio.go                     # SendAudio
            ├── context.go                   # buildContext, replyContext, mentionContext
            ├── doc.go                       # Documentação do pacote
            ├── document.go                  # SendDocument
            ├── image.go                     # SendImage + geração de thumbnail JPEG
            ├── location.go                  # SendLocation
            ├── message_text.go              # PlainTextFromProto — extrai texto da mensagem
            ├── presence.go                  # withTyping — indicador de digitação
            ├── reaction.go                  # SendReaction
            ├── reply.go                     # Reply (atalho com quote)
            ├── sticker.go                   # SendSticker
            ├── text.go                      # SendText, SendTextWithMentions, SendTextToJID
            └── video.go                     # SendVideo
```

---

## Criando um novo comando

**1.** Crie o handler em `internal/commands/public/` ou `internal/commands/admin/`:

```go
package public

import (
    "context"

    "github.com/Turgho/YuukoWhatsapp/internal/integration/whatsapp"
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