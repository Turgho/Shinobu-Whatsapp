# Shinobu — WhatsApp Bot

[![Status](https://img.shields.io/badge/status-em%20desenvolvimento-red)](https://github.com/Turgho/Shinobu-Whatsapp)
[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-green)](./LICENSE)
[![Último commit](https://img.shields.io/github/last-commit/Turgho/Shinobu-Whatsapp)](https://github.com/Turgho/Shinobu-Whatsapp/commits/main)

Bot de WhatsApp escrito em Go, com arquitetura modular baseada em router de comandos, middlewares e injeção de dependências. Inclui IA via **Groq** com personalidade da Oshino Shinobu (Monogatari Series), histórico de conversa por usuário e busca web sob demanda via Tavily.

---

## Requisitos

- Go 1.25+
- [ffmpeg](https://ffmpeg.org/download.html) — necessário para o comando `!sticker`
- [yt-dlp](https://github.com/yt-dlp/yt-dlp) — necessário para o comando `!play`
- Conta no [Groq](https://console.groq.com) — gratuito, necessário para o `!shinobu`
- Conta no [Tavily](https://tavily.com) — gratuito (1.000 buscas/mês), para busca web automática

```bash
# Instalar ffmpeg

# Ubuntu / Debian
sudo apt install ffmpeg

# Fedora (requer RPM Fusion)
sudo dnf install https://mirrors.rpmfusion.org/free/fedora/rpmfusion-free-release-$(rpm -E %fedora).noarch.rpm
sudo dnf install ffmpeg

# Arch Linux / Manjaro
sudo pacman -S ffmpeg

# macOS
brew install ffmpeg
```

---

## Instalação

```bash
git clone https://github.com/Turgho/Shinobu-Whatsapp.git
cd Shinobu-Whatsapp
go mod tidy
```

### Configuração

Copie o arquivo de exemplo e ajuste:

```bash
cp config.example.yaml config.yaml
```

```yaml
bot:
  name: Shinobu
  prefix: "!"
  environment: "development"

database:
  driver: sqlite3
  dsn: "file:storage/storage.db?_foreign_keys=on"

log:
  level: debug   # debug | info | warn | error

usersJID:
  owner: "seu_jid@lid"   # JID do dono do bot (aparece nos logs na 1ª execução)
  admins: []             # JIDs adicionais com permissão de admin

apiUrls:
  geocoding: "https://nominatim.openstreetmap.org/search"
  weather: "https://api.open-meteo.com/v1/forecast"
```

Crie o arquivo `.env` com as credenciais:

```env
# IA — Groq (obrigatório para !shinobu)
GROQ_URL=https://api.groq.com/openai/v1/chat/completions
GROQ_API_KEY=gsk_sua_chave_aqui

# Busca web — Tavily (opcional; sem isso a Shinobu não pesquisa na internet)
TAVILY_API_KEY=tvly-sua-chave-aqui

# Número do dono do bot (sem + ou espaços, ex: 5511999999999)
OWNER_NUMBER=seu_numero

# URL do servidor de músicas — usado pelo !stats para exibir stats do notebook (opcional)
MUSIC_SERVER_URL=http://seu-servidor:porta
```

> **Como encontrar seu JID:** inicie o bot uma vez, envie qualquer mensagem para ele e o JID aparecerá nos logs.

> **Chave Groq:** acesse [console.groq.com](https://console.groq.com), crie uma API Key gratuita e cole em `GROQ_API_KEY`.

> **Chave Tavily:** acesse [tavily.com](https://tavily.com) e crie uma conta gratuita — o plano free inclui 1.000 buscas/mês.

---

## IA — Oshino Shinobu

O comando `!shinobu` usa a API **Groq** com modelos Llama 4 e personalidade customizada da Oshino Shinobu. Não é necessário instalar nenhum modelo localmente.

### Funcionalidades da IA

- Personalidade customizada via system prompt
- Histórico de conversa por usuário (SQLite, últimas mensagens / 12 horas)
- Resumo persistente da conversa — mantém contexto entre sessões
- Busca web automática via Tavily quando a pergunta exige dados atuais
- Detecção inteligente de busca em duas camadas: keywords locais → classificador leve (sem custo extra quando a keyword já é óbvia)
- Tom diferenciado para o owner do bot
- Menção direta pelo nome ("Shinobu, ...") funciona sem precisar do prefixo `!`

### Modelos utilizados

| Situação | Modelo |
|----------|--------|
| Conversa e resumo | `meta-llama/llama-4-scout-17b-16e-instruct` |
| Resposta com contexto web | `llama-3.3-70b-versatile` |
| Classificação de busca | `meta-llama/llama-4-scout-17b-16e-instruct` (MaxTokens=5) |

---

## Execução

```bash
go run cmd/bot/main.go
```

Na primeira execução, escaneie o QR Code com o WhatsApp para autenticar. As credenciais são salvas em `storage/storage.db` — nas próximas execuções a conexão é automática.

---

## Comandos disponíveis

| Comando | Descrição | Permissão |
|---------|-----------|-----------|
| `!menu` | Lista todos os comandos disponíveis | Pública |
| `!ping` | Verifica latência do bot | Pública |
| `!clima <cidade>` | Clima atual de uma cidade | Pública |
| `!sticker` | Converte imagem ou vídeo em figurinha | Pública |
| `!play <nome ou URL>` | Envia música por nome ou URL | Pública |
| `!shinobu <mensagem>` | Conversa com a IA Shinobu | Pública |
| `!mambo` | M A M B O 🏇 | Pública |
| `!dio` | Talvez o tempo pare... | Pública |
| `!cafe` | Não importa a hora! | Pública |
| `!stats` | Estatísticas de runtime do bot | Admin |
| `!shutdown` | Desliga o bot | Admin |
| `!fig <nome>` | Envia figurinha salva; em DM: `!fig salvar <nome>` / `!fig remover <nome>` | Admin |

> **Aviso sobre cookies do yt-dlp**
>
> O comando `!play` pode usar `cookies.txt` para autenticar requisições no YouTube.
>
> - **NÃO** compartilhe esse arquivo com ninguém
> - **NÃO** envie `cookies.txt` para o GitHub (já está no `.gitignore`)
> - Use cookies **apenas** da sua própria conta
>
> Sem cookies válidos, alguns vídeos podem não ser baixados.
> [Como usar cookies com yt-dlp](https://github.com/yt-dlp/yt-dlp/wiki/FAQ#how-do-i-pass-cookies-to-yt-dlp)

---

## Estrutura do projeto

```text
├── cmd/bot/main.go                    # Entry point
│
├── internal/
│   ├── app/app.go                     # Inicialização e injeção de dependências
│   ├── bot/
│   │   ├── client.go                  # Conexão com WhatsApp via whatsmeow
│   │   └── handler.go                 # Dispatcher de eventos
│   ├── commands/
│   │   ├── admin/                     # Comandos privados (owner / admins)
│   │   │   ├── save_sticker.go        # !fig — gerencia figurinhas salvas
│   │   │   ├── shutdown.go            # !shutdown
│   │   │   └── stats.go               # !stats
│   │   ├── public/                    # Comandos públicos
│   │   │   ├── bundled_audio.go       # FixedBundledAudioCommand — !mambo, !dio, !cafe
│   │   │   ├── menu.go                # !menu gerado automaticamente dos metadados
│   │   │   ├── ping.go                # !ping
│   │   │   ├── play.go                # !play via yt-dlp
│   │   │   ├── shinobu.go             # !shinobu — entrada da IA
│   │   │   ├── sticker.go             # !sticker
│   │   │   └── weather.go             # !clima
│   │   ├── middleware.go              # IgnoreSelf, IgnoreOldMessages, permissões
│   │   ├── router.go                  # Roteamento e pipeline de middlewares
│   │   └── types.go                   # HandlerFunc, CommandMeta, ArgMeta
│   ├── configs/                       # config.yaml e carregamento de configuração
│   ├── database/                      # Conexão com SQLite
│   └── utils/                         # SendText, SendAudio, SendImage, uptime…
│
└── pkg/
    ├── geocoding/                     # Geocoding via Nominatim (OpenStreetMap)
    ├── history/                       # Histórico de mensagens por usuário (SQLite)
    ├── ia/
    │   ├── groq.go                    # Cliente HTTP centralizado para o Groq
    │   ├── ia.go                      # AskIA — orquestra histórico, busca e modelo
    │   ├── keywords.go                # Keywords para decisão de busca sem LLM
    │   ├── models.go                  # Constantes e parâmetros dos modelos Groq
    │   ├── prompts.go                 # System prompts e modos de resposta
    │   ├── search.go                  # shouldSearchWeb + classificador leve
    │   ├── summary.go                 # Geração e atualização do resumo persistente
    │   ├── tavily.go                  # Cliente Tavily para busca web
    │   └── utils.go                   # cleanPrompt, truncateText
    ├── logger/                        # Logger baseado em Zap
    ├── music/                         # Download de áudio via yt-dlp
    ├── sticker/                       # Conversão de mídia para WebP via ffmpeg
    └── weather/                       # Cliente Open-Meteo + mapeamento de códigos
```

---

## Criando um novo comando

**1.** Crie o arquivo em `internal/commands/public/` ou `internal/commands/admin/`:

```go
package public

import (
    "context"

    "github.com/Turgho/YuukoWhatsapp/internal/utils"
    "go.mau.fi/whatsmeow"
    "go.mau.fi/whatsmeow/types/events"
)

func HelloCommand(ctx context.Context, client *whatsmeow.Client, evt *events.Message, args []string) error {
    return utils.Reply(ctx, client, evt, "Olá! 👋")
}
```

**2.** Registre em `internal/app/app.go` dentro de `registerPublicCommands()` (ou `registerAdminCommands()` para comandos privados):

```go
r.RegisterCommand(commands.CommandMeta{
    Name:        "hello",
    Description: "Responde com uma saudação",
    Type:        commands.CommandTypeUtility,
}, public.HelloCommand)
```

O comando aparece automaticamente no `!menu`. Para torná-lo privado (somente owner/admins), adicione `Private: true`:

```go
r.RegisterCommand(commands.CommandMeta{
    Name:    "hello",
    Private: true,
    // ...
}, public.HelloCommand)
```

---

## Contato

- Autor: **Turgho** — [github.com/Turgho](https://github.com/Turgho)
- Sugestões ou dúvidas: abra uma **issue** no repositório.