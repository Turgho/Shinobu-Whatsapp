# 🤖 Yuuko — WhatsApp Bot em Go

[![Status](https://img.shields.io/badge/status-em%20desenvolvimento-red)](https://github.com/Turgho/YuukoWhatsapp)
[![Linguagem](https://img.shields.io/badge/Linguagem-Go-blue)](https://go.dev/doc/)
[![Último commit](https://img.shields.io/github/last-commit/Turgho/YuukoWhatsapp)](https://github.com/Turgho/YuukoWhatsapp/commits/main)
[![License: MIT](https://img.shields.io/badge/License-MIT-green)](./LICENSE)

**Yuuko** é um bot de **WhatsApp desenvolvido em Go**, focado em automação de comandos, utilidades e integração com APIs externas.

O bot utiliza a biblioteca **Whatsmeow** para conexão com o WhatsApp Web e possui uma arquitetura modular para facilitar a criação de novos comandos.

> **Status atual**: desenvolvimento ativo — novas funcionalidades estão sendo adicionadas continuamente.

---

## 🚀 Funcionalidades atuais

O bot atualmente possui algumas funcionalidades básicas e estrutura para expansão:

🔧 Utilidades

- ✅ Ping — verifica se o bot está online
- ✅ Weather — consulta clima de uma cidade usando geocoding + API de clima
- ✅ Comandos com prefixo configurável

⚙️ Sistema de comandos

- ✅ Router de comandos
- ✅ Middlewares
- ✅ Comandos privados (admins / owner)
- ✅ Tratamento de comandos inexistentes

🛡️ Segurança

- ✅ Filtro de mensagens antigas
- ✅ Ignorar mensagens do próprio bot
- ✅ Sistema de permissões para comandos privados

---

## 🧱 Arquitetura do projeto

A estrutura do projeto segue um modelo modular para facilitar manutenção e expansão:

```text
.
├── cmd
│   └── bot
│       └── main.go          # Entry point da aplicação
│
├── internal                 # Código interno do bot
│   ├── app
│   │   └── app.go           # Inicialização da aplicação
│   │
│   ├── bot
│   │   ├── client.go        # Cliente WhatsApp (Whatsmeow)
│   │   └── handler.go       # Handler de eventos
│   │
│   ├── commands             # Sistema de comandos
│   │   ├── admin            # Comandos administrativos
│   │   │   ├── shutdown.go
│   │   │   └── stats.go
│   │   │
│   │   ├── public           # Comandos públicos
│   │   │   ├── ping.go
│   │   │   └── weather.go
│   │   │
│   │   ├── middleware.go    # Middlewares do router
│   │   ├── router.go        # Router de comandos
│   │   └── types.go         # Tipos e interfaces
│   │
│   ├── configs
│   │   ├── config.go        # Carregamento de configuração
│   │   └── config.yaml      # Arquivo de configuração
│   │
│   ├── database
│   │   └── database.go      # Conexão com banco
│   │
│   └── utils                # Funções utilitárias
│       ├── message.go       # Envio de mensagens
│       └── uptime.go        # Controle de uptime
│
├── pkg                      # Pacotes reutilizáveis
│   ├── geocoding
│   │   └── geocode.go       # Geocoding via OpenStreetMap
│   │
│   ├── logger
│   │   └── logger.go        # Logger baseado em Zap
│   │
│   └── weather
│       ├── weather.go       # Client da API de clima
│       └── weather_code.go  # Mapeamento de weather codes
│
├── storage
│   └── storage.db           # Banco SQLite local
│
├── go.mod
├── go.sum
├── LICENSE
└── README.md
```

---

## 📦 Organização do código

O projeto utiliza duas pastas principais:

`internal/`

Contém **toda a lógica do bot**, incluindo:

- inicialização da aplicação
- sistema de comandos
- conexão com banco
- utilitários
- handlers do WhatsApp

> Esses pacotes são privados ao projeto.

`pkg/`

Contém **bibliotecas reutilizáveis**, como:

- integração com APIs externas
- logger
- serviços independentes do bot

> Esses pacotes podem ser reutilizados em outros projetos.

---

## 🌐 APIs utilizadas

O bot utiliza algumas **APIs externas** para fornecer funcionalidades:

- Geocoding: OpenStreetMap / Nominatim
- Weather: Open-Meteo

> APIs permitem converter **nome de cidade** → **coordenadas** e **buscar dados climáticos atualizados**.

### 🧪 Exemplo de comando

```bash
!weather São Paulo
```

Resposta:

```bash
🌍 Local: São Paulo, Brasil
🌡️ Temperatura: 27°C
🤗 Sensação térmica: 29°C
☔ Chuva: 0mm (0%)
💨 Vento: 10 km/h
☀️ Céu limpo
```

---

## 🛠️ Tecnologias

- **Linguagem**: Go
- **WhatsApp API**: Whatsmeow
- **Logs**: Zap
- **Banco de dados**: SQL (dependendo da configuração)
- **Arquitetura**: Modular + Router de comandos

---

## ⚙️ Como executar

1️⃣ Clonar o repositório

```bash
git clone https://github.com/Turgho/YuukoWhatsapp.git
cd YuukoWhatsapp
```

2️⃣ Instalar dependências

```bash
go mod tidy
```

3️⃣ Rodar o bot

```bash
go run cmd/bot/main.go
```

> Na primeira execução será necessário escanejar o QR Code do WhatsApp para conectar o bot.

---

## ✨ Criando novos comandos

Novos comandos podem ser adicionados dentro da pasta:

```bash
internal/commands/public
```

ou

```bash
internal/commands/admin
```

Depois basta registrar o comando no router:

```bash
r.RegisterCommand("ping", public.PingCommand)
```

---

## ⚡ Contato

- Autor / Maintainer: **Turgho** — perfil no GitHub: [Turgho](https://github.com/Turgho)
- Para sugestões ou dúvidas, abra uma **issue** no repositório.

---

Obrigado por visitar o **Shinobu-Discord**
