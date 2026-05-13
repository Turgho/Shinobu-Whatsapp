# Shinobu — WhatsApp Bot

[![Status](https://img.shields.io/badge/status-em%20desenvolvimento-red)](https://github.com/Turgho/Shinobu-Whatsapp)
[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-green)](./LICENSE)
[![Último commit](https://img.shields.io/github/last-commit/Turgho/Shinobu-Whatsapp)](https://github.com/Turgho/Shinobu-Whatsapp/commits/main)

Bot de WhatsApp em Go (**whatsmeow**), com router de comandos, middlewares e dependências explícitas no arranque. Inclui IA via **Groq** (personalidade Oshino Shinobu), histórico por utilizador, busca web opcional (**Tavily**), aniversários em grupo com lembrete diário, efeitos de áudio (**ffmpeg**) e música via servidor remoto (**yt-dlp** no lado do servidor).

**Módulo Go:** `github.com/Turgho/YuukoWhatsapp` — nos `import` use sempre este caminho (o clone do Git pode chamar-se `Shinobu-Whatsapp`).

---

## Requisitos

| Item | Detalhe |
|------|---------|
| **Go** | 1.25+ (ver `go.mod`) |
| **ffmpeg** e **webpmux** | Arranque valida `./bin/ffmpeg` e `./bin/webpmux` relativos ao diretório de trabalho — execute `go run` a partir da **raiz do repositório** ou coloque os binários em `./bin/` |
| **Servidor de música** | `!play` e `!stats` usam `MUSIC_SERVER_URL` (ver abaixo) |
| **Groq** | Obrigatório para `!shinobu` / menção “shinobu” |
| **Tavily** | Opcional; sem chave a IA não faz busca web |

### Instalar ffmpeg (sistema)

```bash
# Ubuntu / Debian
sudo apt install ffmpeg

# Fedora (requer RPM Fusion)
sudo dnf install https://mirrors.rpmfusion.org/free/fedora/rpmfusion-free-release-$(rpm -E %fedora).noarch.rpm
sudo dnf install ffmpeg

# Arch Linux / CachyOS / Manjaro
sudo pacman -S ffmpeg

# macOS
brew install ffmpeg
```

O pacote **libwebp** costuma incluir o utilitário `webpmux`; copie ou linke para `./bin/webpmux` se não estiver no `PATH` global.

---

## Instalação

```bash
git clone https://github.com/Turgho/Shinobu-Whatsapp.git
cd Shinobu-Whatsapp
go mod tidy
```

### `config.yaml`

```bash
cp config.example.yaml config.yaml
```

O ficheiro de exemplo define `bot`, `database`, `log`, `usersJID` e `apiUrls`. O carregamento está em `internal/infra/configs/config.go` (**Viper** + `.env`): campos podem ser sobrescritos por variáveis de ambiente (por exemplo `OWNER_JID`, `COMMAND_PREFIX`, `DB_DSN`).

### `.env` — credenciais e URLs

```env
# IA — Groq (obrigatório para !shinobu)
GROQ_URL=https://api.groq.com/openai/v1/chat/completions
GROQ_API_KEY=gsk_sua_chave_aqui

# Busca web — Tavily (opcional)
TAVILY_API_KEY=tvly-sua-chave-aqui

# Número do dono (sem + ou espaços), usado em fluxos que comparam por número
OWNER_NUMBER=5511999999999

# Servidor de música: !play → POST /play (corpo texto = query)
# !stats → GET {MUSIC_SERVER_URL}/stats (JSON com stats remotas, se configurado)
MUSIC_SERVER_URL=http://seu-servidor:porta
```

> **JID do dono:** inicie o bot, envie uma mensagem e use o JID que aparece nos logs em `usersJID.owner` no `config.yaml`.

> **Groq:** [console.groq.com](https://console.groq.com) — API key gratuita.

> **Tavily:** [tavily.com](https://tavily.com) — plano free com limite mensal de buscas.

---

## Execução

```bash
go run cmd/bot/main.go
```

Na primeira execução, escaneie o QR Code. A sessão WhatsApp fica em SQLite conforme `database.dsn` (por defeito `storage/storage.db`). O histórico auxiliar da IA usa `storage/message_history.db`.

---

## Comandos

| Comando | Descrição | Permissão |
|---------|-----------|------------|
| `!menu` | Lista comandos a partir dos metadados registados | Pública |
| `!ping` | Latência / disponibilidade | Pública |
| `!clima <cidade>` | Clima (Nominatim + Open-Meteo) | Pública |
| `!sticker` | Imagem ou vídeo → figurinha (ffmpeg) | Pública |
| `!play <nome ou URL>` | Áudio via `MUSIC_SERVER_URL` (servidor com yt-dlp) | Pública |
| `!efeito` / `!efeito <nome> [intensidade]` | Lista ou aplica efeito num áudio (enviar ou citar áudio). Intensidades: `leve`, `medio`, `forte` | Pública |
| `!shinobu <texto>` | Conversa com a IA | Pública |
| Menção **“shinobu”** no texto | Atalho sem prefixo (mesmo handler que `!shinobu`) | Pública |
| `!aniversário` | **Grupo:** `DD/MM`, `lista`, `remover`, `salvar @pessoa DD/MM` (dono/admin), `remover @pessoa` (dono/admin) | Pública |
| `!mambo`, `!dio`, `!cafe` | Áudios OGG em `assets/audios/` (+ figurinhas salvas onde aplicável) | Pública |
| `!stats` | Stats locais + opcionalmente `GET …/stats` no mesmo host que a música | Admin |
| `!shutdown` | Encerra o processo do bot | Admin |
| `!fig` | Chat: `!fig <nome>`. **DM dono:** `!fig salvar <nome>`, `!fig remover <nome>`, `!fig lista` | Admin |

### Cookies (yt-dlp no servidor de música)

O servidor que atende `!play` pode usar `cookies.txt` para o YouTube. Não partilhe nem commite esse ficheiro. [FAQ yt-dlp — cookies](https://github.com/yt-dlp/yt-dlp/wiki/FAQ#how-do-i-pass-cookies-to-yt-dlp).

---

## IA — Oshino Shinobu

- System prompt com personalidade.
- Histórico por utilizador (SQLite, limpeza periódica).
- Resumo persistente entre sessões.
- Busca web (Tavily) quando a pergunta exige dados atuais.
- Decisão de busca: keywords locais → classificador leve (Groq com poucos tokens).
- Tom diferenciado para o owner.

### Modelos (referência em `internal/domain/ia`)

| Situação | Modelo |
|----------|--------|
| Conversa e resumo | `meta-llama/llama-4-scout-17b-16e-instruct` |
| Resposta com contexto web | `llama-3.3-70b-versatile` |
| Classificação de busca | Scout com `MaxTokens` reduzido |

---

## Aniversários

- Dados persistidos pelo pacote `internal/domain/birthday`.
- **Scheduler:** todos os dias às **08:00** (hora local do processo), notifica grupos com aniversariantes — `internal/domain/birthday/scheduler.go`.

---

## Estrutura do projeto

O código em `internal/` está agrupado em **`domain/`** (negócio do bot), **`infra/`** (config, DB, ferramentas, tempo de vida do processo) e **`integration/`** (envio e download no WhatsApp). A raiz de `internal/` mantém só `app`, `bot` e `commands`.

```text
.
├── cmd/bot/
│   └── main.go                              # Entry point — chama app.Run()
├── config.example.yaml                      # Modelo de configuração (copiar para config.yaml)
├── dependencies.sh                          # Script auxiliar de dependências (se existir no teu clone)
├── go.mod                                   # Módulo github.com/Turgho/YuukoWhatsapp
├── go.sum                                   # Checksums de dependências
├── LICENSE
├── README.md
├── assets/audios/                           # OGG para !mambo, !dio, !cafe (nomes referenciados em app.go)
├── storage/                                 # Criado em runtime: storage.db (WhatsApp), message_history.db, stickers, etc.
│
├── internal/
│   ├── app/                                 # Arranque: deps, router, registo de comandos
│   │   └── app.go
│   ├── bot/                                 # Cliente whatsmeow (sessão, eventos)
│   │   ├── client.go
│   │   └── handler.go
│   ├── commands/                            # Handlers !comando, router, middleware, tipos
│   │   ├── admin/
│   │   ├── public/
│   │   ├── middleware.go
│   │   ├── router.go
│   │   └── types.go
│   ├── domain/                              # Funcionalidades e serviços do bot (regras de negócio)
│   │   ├── birthday/                        # !aniversário — store, DM, scheduler 8h
│   │   ├── geocoding/                       # Nominatim
│   │   ├── history/                         # Histórico SQLite por JID (IA)
│   │   ├── ia/                              # Groq, Tavily, prompts, busca, resumo
│   │   ├── music/                           # !play remoto, efeitos, MIME
│   │   ├── sticker/                         # WebP, store, DM !fig
│   │   └── weather/                         # Open-Meteo + códigos WMO
│   ├── infra/                               # Configuração, persistência, ferramentas, relógio de processo
│   │   ├── configs/                         # Viper + .env
│   │   ├── database/                        # SQLite credenciais whatsmeow
│   │   ├── ffmpeg/                          # Exec ffmpeg (sticker, efeitos)
│   │   ├── logger/                          # Zap (cliente)
│   │   └── uptime/                          # Início do processo (!stats, mensagens antigas)
│   └── integration/                         # Adaptadores WhatsApp (envio + download de mídia)
│       ├── media/                           # DownloadFromEvent
│       └── whatsapp/                        # SendText, Reply, stickers, texto visível no evento, …
```

Ficheiros em `storage/` e `config.yaml` costumam estar no `.gitignore`; use `config.example.yaml` como base. **`internal/domain`** agrupa funcionalidades (IA, stickers, clima, música, aniversários, histórico). **`internal/infra`** cobre configuração (`configs`), SQLite do cliente (`database`), `ffmpeg`, `logger` e `uptime`. **`internal/integration`** concentra o que fala com a API WhatsApp: envio de mensagens (`whatsapp`) e download de anexos (`media`).

---

## Criar um novo comando

**1.** Ficheiro em `internal/commands/public/` ou `internal/commands/admin/`:

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

**2.** Registo em `internal/app/app.go` em `registerPublicCommands` ou `registerAdminCommands`:

```go
r.RegisterCommand(commands.CommandMeta{
    Name:        "hello",
    Description: "Responde com uma saudação",
    Type:        commands.CommandTypeUtility,
}, public.HelloCommand)
```

O `!menu` lista automaticamente. Comando só para owner/admins: `Private: true` no `CommandMeta`.

---

## Contato

- Autor: **Turgho** — [github.com/Turgho](https://github.com/Turgho)
- Issues: [github.com/Turgho/Shinobu-Whatsapp/issues](https://github.com/Turgho/Shinobu-Whatsapp/issues)
