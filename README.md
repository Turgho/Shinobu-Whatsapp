# Shinobu — WhatsApp Bot

[![Status](https://img.shields.io/badge/status-em%20desenvolvimento-red)](https://github.com/Turgho/YuukoWhatsapp)
[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-green)](./LICENSE)
[![Último commit](https://img.shields.io/github/last-commit/Turgho/YuukoWhatsapp)](https://github.com/Turgho/YuukoWhatsapp/commits/main)

Bot de WhatsApp escrito em Go, com arquitetura modular baseada em router de comandos, middlewares e injeção de dependências.

---

## Requisitos

- Go 1.22+
- [ffmpeg](https://ffmpeg.org/download.html) — necessário para o comando `!sticker`
- [yt-dlp](https://github.com/yt-dlp/yt-dlp) - necessário para o comando `!play`

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

Configure o arquivo `internal/configs/configs.yaml` com suas credenciais e preferências.

```yaml
bot:
  name: YuukoBot
  prefix: "!"          # prefixo dos comandos
  environment: "development"

database:
  driver: sqlite3
  dsn: "file:storage/storage.db?_foreign_keys=on"

log:
  level: debug         # debug | info | warn | error

usersJID:
  owner: "seu_jid@lid" # JID do dono do bot (aparece no log ao iniciar)
  admins: []           # JIDs adicionais com permissão de admin

apiUrls:
  geocoding: "https://nominatim.openstreetmap.org/search"
  weather: "https://api.open-meteo.com/v1/forecast"
```

> **Como encontrar seu JID:** inicie o bot uma vez, envie qualquer mensagem para ele e o JID do remetente aparecerá nos logs. Cole-o no campo `owner`.

---

## Execução

```bash
go run cmd/bot/main.go
```

Na primeira execução, escaneie o QR Code com o WhatsApp para autenticar. As credenciais são salvas localmente em `storage/storage.db` — nas próximas execuções a conexão é automática.

---

## Comandos disponíveis

| Comando | Descrição | Permissão |
|---------|-----------|-----------|
| `!menu` | Lista todos os comandos disponíveis | Pública |
| `!ping` | Verifica latência do bot | Pública |
| `!weather <cidade>` | Clima atual de uma cidade | Pública |
| `!sticker` | Converte imagem ou vídeo em figurinha | Pública |
| `!stats` | Estatísticas de runtime do bot | Admin |
| `!shutdown` | Desliga o bot | Admin |
| `!play` | Envia música por nome ou URL | Pública |

> **Aviso sobre cookies**
>
> Este projeto pode usar `cookies.txt` para autenticar requisições em serviços como o YouTube via `yt-dlp`, especialmente para conteúdos com restrição de idade, região ou sessão autenticada.
>
> **Importante:**  
> - NÃO compartilhe esse arquivo com ninguém;  
> - NÃO envie `cookies.txt` para o GitHub;  
> - Adicione `cookies.txt` ao `.gitignore`;  
> - Use cookies APENAS sua própria conta.
>
> Sem cookies válidos, alguns vídeos e músicas podem não ser baixados.
>
> FAQ do repositório oficial - [YT-DLP Cookies](https://github.com/yt-dlp/yt-dlp/wiki/FAQ#how-do-i-pass-cookies-to-yt-dlp)
>
> Link de um vídeo explicando como corrigir - [Sign in to confirm you’re not a bot](https://youtu.be/nT_uI1raf6k?si=Nl2XfW_TCSYKVD6x)

---

## Estrutura do projeto

```
.
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
│   ├── configs/                 # configs.toml e carregamento de configuração
│   ├── database/                # Conexão com SQLite
│   └── utils/                   # Reply, uptime
│
└── pkg/
    ├── geocoding/               # Geocoding via Nominatim (OpenStreetMap)
    ├── logger/                  # Logger baseado em Zap
    ├── sticker/                 # Conversão de mídia para WebP via ffmpeg
    └── weather/                 # Cliente Open-Meteo + mapeamento de códigos
```

---

## Criando um novo comando

**1.** Crie o arquivo em `internal/commands/public/` ou `internal/commands/admin/`:

```go
// internal/commands/public/hello.go
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

O comando aparece automaticamente no `!menu` — sem mais nenhuma alteração.

Para comandos privados (apenas owner/admins), adicione `Private: true` nos metadados:

```go
r.RegisterCommand(commands.CommandMeta{
    Name:    "hello",
    Private: true,
}, public.HelloCommand)
```

---

## ⚡ Contato

- Autor / Maintainer: **Turgho** — perfil no GitHub: [Turgho](https://github.com/Turgho)
- Para sugestões ou dúvidas, abra uma **issue** no repositório.

---

Obrigado por visitar o **BarraTour**
