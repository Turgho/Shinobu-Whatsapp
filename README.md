# Shinobu

[![Status](https://img.shields.io/badge/status-em%20desenvolvimento-orange?style=flat-square)](https://github.com/Turgho/Shinobu-Whatsapp)
[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-green?style=flat-square)](./LICENSE)
[![Último commit](https://img.shields.io/github/last-commit/Turgho/Shinobu-Whatsapp?style=flat-square)](https://github.com/Turgho/Shinobu-Whatsapp/commits/main)

Bot de WhatsApp escrito em Go, construído sobre **whatsmeow**. Possui router de comandos com middlewares, IA com personalidade via **Groq**, memória atômica por usuário, histórico de conversa em SQLite, busca web via **Tavily** com priorização de resultados brasileiros, gerenciamento de aniversários com scheduler diário, lembretes dinâmicos agendados por linguagem natural, scheduler genérico para jobs semanais/configuráveis (áudio + @all + sticker), efeitos de áudio com **ffmpeg**, reprodução de música via servidor remoto com **yt-dlp** e sistema de figurinhas salvas.

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
- [Scheduler de Jobs](#scheduler-de-jobs)
- [Estrutura do projeto](#estrutura-do-projeto)
- [Criando um novo comando](#criando-um-novo-comando)
- [Melhorias previstas](#melhorias-previstas)

---

## Requisitos

| Dependência | Detalhe |
|-------------|---------|
| **Go 1.26+** | Ver `go.mod` |
| **ffmpeg** | Necessário para `!sticker`, `!efeito` e extração de capa de áudio. Binário em `./bin/ffmpeg` |
| **webpmux** | Necessário para injetar metadados nos stickers. Binário em `./bin/webpmux` |
| **Servidor de música** | `!play` e `!stats` dependem de `MUSIC_SERVER_URL` (servidor com yt-dlp) |
| **Groq** | Obrigatório para `!shinobu`, NLU e memória da IA |
| **Tavily** | Opcional — habilita busca web na IA |

### Instalando dependências (Linux x86-64)

```bash
./scripts/setup.sh
```

O script baixa automaticamente as versões mais recentes de **ffmpeg**, **webpmux** e **yt-dlp** para `./bin/`. O webpmux resolve a versão atual via GitHub API.

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

O arquivo cobre `bot`, `database`, `log`, `usersJID`, `apiUrls` e `scheduledJobs`. Os campos podem ser sobrescritos por variáveis de ambiente via **Viper**.

#### Configurações do bot

| Chave | Tipo | Default | Descrição |
|-------|------|---------|-----------|
| `bot.name` | string | `Shinobu` | Nome do bot |
| `bot.prefix` | string | `!` | Prefixo dos comandos |
| `bot.environment` | string | `development` | Ambiente de execução |
| `bot.timezone` | string | `America/Sao_Paulo` | Fuso horário |
| `bot.nlpGroupTrigger` | bool | `false` | Se `true`, NLU dispara em grupo sem mencionar Shinobu |

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
MUSIC_SERVER_URL=http://seu-servidor:porta
```

> **JID do dono:** inicie o bot, envie uma mensagem e copie o JID que aparece nos logs. Cole em `usersJID.owner` no `config.yaml`.
>
> **Geocoding:** Nominatim (OpenStreetMap) como primário com `countrycodes=BR`. Open-Meteo Geocoding como fallback.
>
> **Groq:** API key gratuita em [console.groq.com](https://console.groq.com).
>
> **Tavily:** plano free em [tavily.com](https://tavily.com).

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
| `!clima <cidade> [data]` (atalho: `!c`, `!tempo`) | Clima atual ou previsão para data futura. Data aceita: "amanhã", "sexta", "28/06", "5 de janeiro" |
| `!sticker` (atalho: `!s`, `!figurinha`) | Converte imagem ou vídeo em figurinha |
| `!play <nome ou URL>` (atalho: `!p`) | Baixa e envia música como documento com nome e capa do álbum |
| `!efeito [nome] [intensidade]` (atalho: `!e`) | Aplica efeito em áudio citado. Sem args lista os disponíveis |
| `!shinobu <texto>` | Conversa com a IA — também ativada mencionando "shinobu" |
| `!aniversário` (atalho: `!a`, `!aniver`) | Gerencia aniversários do grupo |
| `!agenda <data> <mensagem>` | Agenda lembretes. Aceita linguagem natural: "daqui 5 minutos", "amanhã às 9h", "28/06 14:00", "5 de janeiro" |
| `!agenda lista` | Lista lembretes agendados |
| `!agenda remover <número>` | Remove um lembrete pelo número da lista |
| `!mambo`, `!dio`, `!cafe` | Reproduz áudios OGG estáticos |
| `!cotacao` (atalho: `!cot`, `!dolar`, `!euro`) | Cotação do dólar e euro em reais |
| `!feriado` (atalho: `!feriados`) | Próximos feriados nacionais |
| `!noticia` (atalho: `!news`) | Principais notícias do dia |
| `!receita <prato>` | Busca uma receita culinária |
| `!piada` | Conta uma piada |
| `!fato` (atalho: `!curiosidade`) | Compartilha um fato curioso |
| `!filme [gênero]` (atalho: `!movie`) | Recomenda um filme |
| `!contagem <evento> <data>` (atalho: `!dias`) | Conta os dias para uma data |
| `!unsticker` (atalho: `!us`) | Converte figurinha de volta em imagem |
| `!traduz [idioma]` (atalho: `!translate`) | Traduz texto para português ou outro idioma |

> A IA também entende comandos sem prefixo quando mencionada: *"Shinobu, toca uma música do Metallica"*, *"Shinobu, qual o clima em Campinas?"*, *"Shinobu, me lembra daqui 10 minutos de ligar pro médico"*.

### `!agenda` — formatos de data aceitos

| Usuário digita | Interpretado como |
|----------------|-------------------|
| `daqui 5 minutos` | agora + 5 min |
| `em 2 horas` | agora + 2h |
| `amanhã às 9h` | amanhã 09:00 |
| `28/06 14:00` | 28 jun, 14:00 |
| `5 de janeiro` | 5 jan 08:00 (próximo) |
| `2026-06-28T09:00` | ISO8601 direto |

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
| `!memoria` | Mostra resumo e fatos da IA para o chat atual |
| `!memoria limpar` | Apaga toda a memória da IA para o chat |
| `!memoria limpar @usuário` | Apaga fatos de um usuário específico |
| `!testjob [audioPath] [stickerName]` | Testa envio de áudio+@all+sticker no chat atual |
| `!manutencao` | Ativa/desativa modo manutenção |

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

- Personalidade definida via system prompt (sarcástica, direta, natural).
- Histórico por usuário armazenado em SQLite com limpeza periódica.
- **Memória em duas camadas:**
  - Resumo textual por chat (`chat_summaries`) — regenerado a cada 2h com 30+ mensagens
  - Fatos atômicos por usuário (`user_facts`) — extraídos a cada mensagem, expiram em 30 dias
- Busca web via Tavily com priorização de fontes brasileiras e enriquecimento de queries de preço.
- Tom diferenciado para o owner.
- **NLU (linguagem natural):** detecta intenção e despacha comandos internamente sem prefixo.
  Resolve datas relativas ("amanhã", "sexta", "daqui 10 minutos") para ISO8601 antes de despachar.
  - **DMs:** sempre ativa — qualquer mensagem sem prefixo passa pela NLU
  - **Grupos (default):** só dispara quando "shinobu" ou o JID do bot é mencionado explicitamente
  - **Grupos (`nlpGroupTrigger: true`):** heurísticas adicionais ativas (pergunta com `?`, verbos de intenção, endereçamento direto)
  - `isMentioned` usa match de palavra isolada — "shinobuzinho" não dispara

### Modelos

| Uso | Modelo |
|-----|--------|
| Conversa e resumo | `qwen/qwen3.6-27b` |
| Resposta com contexto web | `openai/gpt-oss-120b` |
| NLU / classificação | `llama-3.1-8b-instant` com `MaxTokens` reduzido |

---

## Scheduler de Jobs

Interface `Job` em `internal/domain/scheduler/`:

```go
type Job interface {
    Name() string
    Next(now time.Time) time.Time
    Run(ctx context.Context) error
}
```

Ticker de **15 segundos**. Jobs one-shot retornam `time.Time{}` em `Next()` após execução e são removidos automaticamente. Cada job tem contexto com timeout de 5 minutos e recuperação individual de panics.

### Aniversários

`BirthdayJob` roda diariamente às **08:00** (horário de Brasília), notificando grupos com aniversariantes via `@all` nativo.

### Lembretes Dinâmicos

`DynamicJob` criado via `!agenda` ou linguagem natural. Persistido em `storage/dynamic_jobs.json` e restaurado na inicialização — jobs expirados são ignorados automaticamente.

### Jobs de Dia da Semana

Configurável via `config.yaml`:

```yaml
scheduledJobs:
  - name: "sextou"
    day: "friday"
    enabled: true
    hour: 10
    minute: 0
    audioPath: "assets/audios/play_tv.ogg"
    stickerName: "play_tv"
    targetGroups:
      - "grupo_jid@g.us"
```

Envia sequencialmente: áudio PTT → mensagem com `@all` → sticker salvo.

---

## Estrutura do projeto

```text
.
├── cmd/bot/main.go                          # Entry point
├── AGENTS.md                                # Contexto do projeto para o OpenCode
├── config.example.yaml                      # Modelo de configuração
├── scripts/setup.sh                         # Instala ffmpeg, webpmux e yt-dlp em ./bin/
├── go.mod                                   # Módulo: github.com/Turgho/Shinobu-Whatsapp
│
├── assets/
│   ├── audios/                              # OGGs estáticos
│   ├── images/                              # Banner do !menu
│   ├── stickers/                            # JSON do store de figurinhas
│   └── videos/                              # Vídeos estáticos (uso futuro)
│
├── storage/                                 # Gerado em runtime — não commitar
│   ├── storage.db                           # Sessão whatsmeow
│   ├── message_history.db                   # Histórico e memória da IA
│   └── dynamic_jobs.json                    # Lembretes agendados
│
└── internal/
    ├── app/                                 # Inicialização de todas as deps
    │   ├── app.go                             # lifecycle: Run, buildLogger, connectDatabase, buildRouter
    │   ├── app_commands.go                    # Registro de comandos públicos e admin
    │   └── app_aliases.go                     # Mapeamento de aliases e erros comuns
    ├── bot/                                 # Sessão whatsmeow + dispatcher de eventos
    ├── commands/
    │   ├── router.go                        # Router struct, registro, pipeline de mensagens
    │   ├── nlu.go                           # Triggers NLU, detecção de menção, dispatch via NLU
    │   ├── ratelimit.go                     # Rate limiting por sender
    │   ├── middleware.go                     # IgnoreOld, NotFound, PrivateCommands
    │   ├── types.go                         # CommandMeta, HandlerFunc, ArgMeta
    │   ├── admin/                           # stats, shutdown, restart, fig, ignorar, memoria, testjob
    │   └── public/                          # clima, play, sticker, efeito, shinobu, agenda, aniversário
    ├── domain/
    │   ├── birthday/                        # BirthdayJob + store JSON
    │   ├── geocoding/                       # Nominatim (primário) + Open-Meteo (fallback)
    │   ├── history/                         # Histórico SQLite + user_facts + resumo
    │   ├── ia/                              # Groq, Tavily, NLU, prompts, memória
    │   │   ├── ia.go                          # Orquestração: AskIA, callGroq, QuickChat
    │   │   ├── intent.go                      # DetectIntent (classificação de intenção)
    │   │   ├── prompts.go                     # Personalidade, modos de resposta
    │   │   ├── search.go                      # shouldSearchWeb, classifyNeedsWebSearch
    │   │   ├── keywords.go                    # Palavras-chave para busca web
    │   │   ├── summary.go                     # refreshChatSummary, generateConversationSummary
    │   │   ├── facts.go                       # extractAndStoreFacts (fatos atômicos)
    │   │   ├── tavily.go                      # Integração Tavily
    │   │   ├── groq.go                        # Chamadas à API Groq
    │   │   ├── models.go                      # Constantes de modelo e config
    │   │   └── utils.go                       # Helpers (cleanPrompt, sanitizePrompt, truncateText)
    │   ├── ignore/                          # Store JSON de números ignorados
    │   ├── music/                           # Efeitos ffmpeg, MIME, requisição yt-dlp
    │   ├── scheduler/                       # Scheduler genérico + DynamicJob + DynamicStore
    │   ├── sticker/                         # Conversão WebP, store, envio
    │   ├── weekday/                         # WeekdayJob
    │   └── weather/                         # Open-Meteo forecast + mapa de códigos WMO
    ├── infra/
    │   ├── configs/                         # Viper + .env
    │   ├── database/                        # SQLite para whatsmeow
    │   ├── ffmpeg/                          # exec.Cmd + prioridade de processo
    │   ├── gosafe/                          # goroutine com recover
    │   ├── logger/                          # Logger whatsmeow
    │   ├── phone/                           # Normalização de números
    │   └── uptime/                          # Timestamp de início
    └── integration/
        ├── media/                           # Download de mídia de eventos
        └── whatsapp/                        # Send (audio, doc, image, sticker, text, video)
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

**2.** Registre em `internal/app/app_commands.go`:

```go
r.RegisterCommand(commands.CommandMeta{
    Name:        "hello",
    Description: "Responde com uma saudação",
    Type:        commands.CommandTypeUtility,
}, public.HelloCommand)
```

Para restringir a owner/admins: `Private: true` no `CommandMeta`.
Para comandos com dependências: use o padrão closure `XyzHandler(dep) HandlerFunc`.

---

## Melhorias previstas

- **Memória por usuário em grupos:** separar contexto por remetente dentro de um mesmo grupo
- **Refactor `weather.go` (~322 linhas):** extrair helper `fetchHourly` para eliminar duplicação entre `fetchOpenMeteo` e `GetForecastForDate`
- **Padronizar `mapstructure`** em `configs/config.go`
- **`StartCleanup`** em `history` migrar para `gosafe.Go`
- **`weekday/job.go Run()`:** remover IIFEs anônimas `func() { ... }()` no loop — usar `defer cancel()` direto

---

## Contato

- Autor: **Turgho** — [github.com/Turgho](https://github.com/Turgho)
- Issues: [github.com/Turgho/Shinobu-Whatsapp/issues](https://github.com/Turgho/Shinobu-Whatsapp/issues)