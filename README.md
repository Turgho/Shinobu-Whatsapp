# Shinobu — WhatsApp Bot

[![Status](https://img.shields.io/badge/status-em%20desenvolvimento-red)](https://github.com/Turgho/YuukoWhatsapp)
[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-green)](./LICENSE)
[![Último commit](https://img.shields.io/github/last-commit/Turgho/YuukoWhatsapp)](https://github.com/Turgho/YuukoWhatsapp/commits/main)

Bot de WhatsApp escrito em Go, com arquitetura modular baseada em router de comandos, middlewares e injeção de dependências. Inclui IA local via Ollama com personalidade da Oshino Shinobu (Monogatari Series), histórico de conversa por usuário e busca web sob demanda via Tavily.

---

## Requisitos

- Go 1.22+
- [Ollama](https://ollama.com) — necessário para a IA local (`!shinobu`)
- [ffmpeg](https://ffmpeg.org/download.html) — necessário para o comando `!sticker`
- [yt-dlp](https://github.com/yt-dlp/yt-dlp) — necessário para o comando `!play`

```bash
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
git clone https://github.com/Turgho/YuukoWhatsapp.git
cd YuukoWhatsapp
go mod tidy
```

### Configuração

Configure o arquivo `internal/configs/configs.yaml`:

```yaml
bot:
  name: YuukoBot
  prefix: "!"
  environment: "development"

database:
  driver: sqlite3
  dsn: "file:storage/storage.db?_foreign_keys=on"

log:
  level: debug

usersJID:
  owner: "seu_jid@lid"
  admins: []

apiUrls:
  geocoding: "https://nominatim.openstreetmap.org/search"
  weather: "https://api.open-meteo.com/v1/forecast"
```

Configure o arquivo `.env` com as credenciais da IA:

```env
OLLAMA_URL=http://localhost:11434/api/chat
TAVILY_API_KEY=tvly-sua-chave-aquis
OWNER_NUMBER=seu-numero
```

> **Como encontrar seu JID:** inicie o bot uma vez, envie qualquer mensagem para ele e o JID aparecerá nos logs.

> **Tavily API Key:** crie uma conta gratuita em [tavily.com](https://tavily.com) — plano gratuito inclui 1.000 buscas/mês.

---

## IA — Oshino Shinobu

O comando `!shinobu` usa um modelo local via Ollama com personalidade customizada.

### Configuração do modelo

```bash
# Instalar Ollama
curl -fsSL https://ollama.com/install.sh | sh

# Baixar o modelo base
ollama pull gemma3:4b

# Criar o modelo com a personalidade da Shinobu
ollama create shinobu -f Modelfile

# Iniciar o Ollama
ollama serve
```

O `Modelfile` com a personalidade **não está incluso no repositório** — é um arquivo pessoal e privado. Crie o seu com base na documentação do Ollama.

### Funcionalidades da IA

- Personalidade customizada via Modelfile
- Histórico de conversa por usuário (últimas 5 mensagens / 2 horas)
- Busca web automática via Tavily quando a pergunta exige informações atuais
- Tom diferenciado para o owner do bot

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
| `!weather <cidade>` | Clima atual de uma cidade | Pública |
| `!sticker` | Converte imagem ou vídeo em figurinha | Pública |
| `!play <nome ou URL>` | Envia música por nome ou URL | Pública |
| `!shinobu <mensagem>` | Conversa com a IA Shinobu | Pública |
| `!stats` | Estatísticas de runtime do bot | Admin |
| `!shutdown` | Desliga o bot | Admin |

> **Aviso sobre cookies**
>
> Este projeto pode usar `cookies.txt` para autenticar requisições no YouTube via `yt-dlp`.
>
> - **NÃO** compartilhe esse arquivo com ninguém
> - **NÃO** envie `cookies.txt` para o GitHub
> - Use cookies **apenas** da sua própria conta
>
> Sem cookies válidos, alguns vídeos podem não ser baixados.
> [Como usar cookies com yt-dlp](https://github.com/yt-dlp/yt-dlp/wiki/FAQ#how-do-i-pass-cookies-to-yt-dlp)

---

## Estrutura do projeto

```text
├── cmd/bot/main.go              # Entry point
│
├── internal/
│   ├── app/app.go               # Inicialização e injeção de dependências
│   ├── bot/
│   │   ├── client.go            # Conexão com WhatsApp via whatsmeow
│   │   └── handler.go           # Dispatcher de eventos
│   ├── commands/
│   │   ├── admin/               # Comandos privados (owner / admins)
│   │   ├── public/              # Comandos públicos
│   │   ├── middleware.go        # IgnoreSelf, IgnoreOldMessages, permissões
│   │   ├── router.go            # Roteamento e pipeline de middlewares
│   │   └── types.go             # HandlerFunc, CommandMeta, ArgMeta
│   ├── configs/                 # configs.yaml e carregamento de configuração
│   ├── database/                # Conexão com SQLite
│   └── utils/                   # Reply, uptime
│
└── pkg/
├── geocoding/               # Geocoding via Nominatim (OpenStreetMap)
├── history/                 # Histórico de mensagens por usuário (SQLite)
├── ia/                      # Cliente Ollama + busca web via Tavily
├── logger/                  # Logger baseado em Zap
├── sticker/                 # Conversão de mídia para WebP via ffmpeg
└── weather/                 # Cliente Open-Meteo + mapeamento de códigos
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

**2.** Registre em `internal/app/app.go` dentro de `registerCommands()`:

```go
r.RegisterCommand(commands.CommandMeta{
    Name:        "hello",
    Description: "Responde com uma saudação",
}, public.HelloCommand)
```

O comando aparece automaticamente no `!menu`. Para comandos privados, adicione `Private: true`:

```go
r.RegisterCommand(commands.CommandMeta{
    Name:    "hello",
    Private: true,
}, public.HelloCommand)
```

---

## Contato

- Autor: **Turgho** — [github.com/Turgho](https://github.com/Turgho)
- Sugestões ou dúvidas: abra uma **issue** no repositório.