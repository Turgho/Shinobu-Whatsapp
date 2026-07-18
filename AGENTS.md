# Shinobu — AGENTS.md

Contexto global do projeto para o OpenCode. Leia este arquivo antes de qualquer tarefa.

## Visão geral

Bot de WhatsApp em Go (`github.com/Turgho/Shinobu-Whatsapp`) construído sobre
**whatsmeow**. Personalidade baseada em Oshino Shinobu (Monogatari Series).
Deploy no Square Cloud. Arquitetura Clean Architecture em todas as camadas.

---

## Convenções obrigatórias

- **Comentários:** português do Brasil
- **Commits:** inglês, conventional commits (`feat:`, `fix:`, `refactor:`, etc.)
- **Imports:** caminho do módulo sempre `github.com/Turgho/Shinobu-Whatsapp/...`
- **Logger:** `*zap.Logger` injetado via construtor — nunca global
- **Estado global:** proibido — toda dependência via construtor ou parâmetro
- **Goroutines:** sempre via `gosafe.Go(logger, fn)` — nunca `go func()` direto
- **Timeouts:** todo `http.Client` deve ter `Timeout` definido — nunca `http.DefaultClient`
- **Erros:** sempre com `%w` para wrap — nunca descartar com `_` sem justificativa
- **JSON NLU:** extrair bloco JSON via json.Unmarshal direto — não existe função extractJSON
- **max_tokens:** obrigatório em toda chamada de API — nunca deixar em aberto

---

## Arquitetura

```
cmd/bot/main.go              → entry point
internal/app/                → inicialização de todas as deps
  app.go                       → lifecycle: Run, buildLogger, connectDatabase, buildRouter, checkDeps
  app_commands.go              → registro de comandos públicos e admin
  app_aliases.go               → mapeamento de aliases e erros comuns
internal/bot/                → sessão whatsmeow + dispatcher de eventos
internal/commands/           → router, middlewares, handlers
  router.go                    → Router struct, registro, pipeline de mensagens
  album.go                     → AlbumCoordinator para buffering de albums
  nlu.go                       → triggers NLU, detecção de menção, dispatch via NLU
  ratelimit.go                 → rate limiting por sender
  middleware.go                → middlewares (self, old, not found, private)
  types.go                     → tipos públicos (HandlerFunc, BatchHandlerFunc, Middleware, CommandMeta)
internal/commands/public/    → handlers de comandos públicos
internal/commands/admin/     → handlers de comandos de admin/owner
internal/domain/             → regras de negócio (sem dependência de whatsmeow)
  ia/                          → IA: prompts, NLU, busca web, resumo, fatos
    ia.go                        → orquestração: AskIA, QuickChat, SearchWeb
    intent.go                    → DetectIntent (classificação de intenção)
    context.go                   → baseSystemMessages, appendPersistentAndRecent
    prompts.go                   → personalidade, modos de resposta
    search.go                    → shouldSearchWeb, classifyNeedsWebSearch
    keywords.go                  → palavras-chave para busca web
    summary.go                   → refreshChatSummary, generateConversationSummary
    summary_format.go            → formatMessagesForSummary
    facts.go                     → extractAndStoreFacts (fatos atômicos)
    tavily.go                    → integração Tavily
    tavily_query.go              → helpers de query Tavily
    groq.go                      → groqChat + callGroq (chamadas à API Groq)
    models.go                    → constantes de modelo, config, IARequest, IAResponse
    utils.go                     → helpers (cleanPrompt, sanitizePrompt, truncateText)
internal/infra/              → infraestrutura transversal (config, db, ffmpeg, gosafe)
internal/integration/        → adaptadores WhatsApp (send, download, presence)
```

Handlers em `internal/commands/` nunca devem conter lógica de negócio —
delegam para `internal/domain/`.

### Regra de tamanho: 150 linhas

Nenhum arquivo em `internal/commands/public/` ou `internal/domain/ia/`
deve exceder 150 linhas. Dividir por responsabilidade — uma preocupação
por arquivo. Exceção aceitável: system prompts extensos em `intent.go`.

---

## Decisões técnicas consolidadas

### Geocoding
- **Primário:** Nominatim (`https://nominatim.openstreetmap.org/search`)
  - params: `countrycodes=BR`, `addressdetails=1`, `format=json`
  - `User-Agent: Shinobu-Whatsapp/1.0` obrigatório
  - `DisplayName` NUNCA usar `display_name` cru do Nominatim
  - Montar sempre a partir de `address.city` > `address.town` > `address.village` + `address.state`
- **Fallback:** Open-Meteo Geocoding (`https://geocoding-api.open-meteo.com/v1/search`)
  - `DisplayName` montado como `"Cidade, Estado"` via campos `name` + `admin1`
- `GeoResult.Country` sempre separado do `DisplayName` — nunca concatenar

### Weather
- Endpoint: `https://api.open-meteo.com/v1/forecast`
- Usar parâmetro `current` (não `current_weather` — deprecated)
- Campos atuais: `temperature_2m`, `relative_humidity_2m`, `apparent_temperature`,
  `precipitation`, `weather_code`, `wind_speed_10m`, `wind_direction_10m`
- `precipitation_probability` só existe em `hourly` — alinhar via `findHourIndex`
- `GetForecastForDate` usa `forecast_days=16` para previsão futura
- **Daily forecast:** `GetDailyForecast` usa `daily` param (weather_code, temp max/min,
  precip prob max) — retorna `[]DailyForecast`, sem depender do hourly
- Fallback: wttr.in via `fetchWttr` — deve usar `w.httpClient`, nunca `http.DefaultClient`
- **Empty-location guard:** geocoding `Lookup` valida `strings.TrimSpace(query) == ""` no topo;
  `WeatherCommand` valida `utf8.RuneCountInString < 2` antes de chamar `Lookup`;
  DetectIntent prompt tem regra explícita "NUNCA invente cidade"
- **Location bug fix:** rastreado e confirmado que não há shadowing ou stale value em
  `WeatherCommand` — o `loc.DisplayName` passado para `GenerateForecastCard` é sempre o
  valor literal de `results[0]` do `geo.Lookup`. O "São Paulo" em queries não relacionadas
  era causado pelo Nominatim retornar state name como fallback quando city/town/village estão
  vazios. Guardas de query vazia + log de debug (`"clima: cidade resolvida"`) previnem e
  rastreiam o problema.
- **Card visual 5 dias (PNG):** `GenerateForecastCard` em `card.go` + `card_icons.go`
  - Biblioteca: `github.com/fogleman/gg` — puro Go, sem cgo, sem binário externo
  - Layout: hero section (220px) + daily list (4 rows de 90px cada) + padding
  - **Hero:** gradiente baseado no clima (sol/nuvem/chuva/etc), nome da cidade + país
    (bold 26px), temperatura grande (bold 72px), descrição textual do WMO code (22px),
    "Sensação X° · Máx Y° / Mín Z°" (18px, semi-transparente), ícone grande (100px)
    alinhado à direita. O gradiente cobre apenas a área do hero (0-220px), não o card
    inteiro.
  - **Daily list:** fundo sólido escuro (`#292e45`), 4 rows com label do dia
    (Amanhã/Sáb/Dom/Seg — bold 22px), ícone vetorial (50px), descrição (20px),
    temperatura máx bold + "/" + mín normal (alinhados à direita ~600px),
    indicador de precipitação (bolinha azul + %) se >30% (~720px).
  - Ícones escalam proporcionalmente: line widths ajustados via `size * factor`
  - Fundo alternado entre rows (semi-transparente) + divisores finos
  - Fallback para texto sempre que o card ou upload falhar
  - `!clima <cidade>` (sem data) → card 5 dias; `!clima <cidade> YYYY-MM-DD` → texto (legado)
  - `WeekdayPT` exportado em `weather` pkg para reúso por outros pacotes
  - Fonte: `gofont` (goregular + gobold) — NÃO usar Inter (está em assets/fonts/ mas não é
    referenciado em código)
  - `GenerateForecastCard` signature: `(forecasts []DailyForecast, current *WeatherResult, location, country string) ([]byte, error)`
    — `current` é opcional (nil = usa forecasts[0] como hero); se não-nil, prefere
    `current.Temperature` e `current.ApparentTemperature` no hero.

### NLU em grupo
- `bot.nlpGroupTrigger` em `config.yaml` (default: `false`)
- `false`: NLU em grupo SÓ dispara quando "shinobu" ou bot JID é mencionado
- `true`: heurísticas adicionais ativas (pergunta, verbos de intenção, endereçamento direto)
- DMs sempre disparam NLU independente do flag
- `isMentioned` usa match de palavra isolada (`wordMatch`) — "shinobuzinho" não dispara
- **Intent kill-switch:** `bot.intentEnabled` em `config.yaml` (default: `false`)
  - `false`: DetectIntent desativado — mensagens com menção vão direto para conversa IA
  - `true`: NLU ativa com detecção de intento via Groq
  - Código do Intent permanece intacto no codebase, pronto para reativar
- **Reply-to-bot:** toda resposta (reply) a uma mensagem do bot dispara NLU mesmo sem menção
  - `isReplyToBot` em `nlu_detect.go` checa se `ContextInfo.Participant` contém o JID do bot
  - O texto da mensagem citada é extraído via `ExtractQuotedText` em `message_text.go`
  - Passado como `quotedContext` para `DetectIntent` e para o fallback `handleShinobuMention`
  - No `DetectIntent`, o contexto é incluído no user content com label "Contexto da conversa"
  - No fallback IA, o contexto é prependido ao prompt para continuidade natural

### IA / Groq
- **Conversa e resumo:** `openai/gpt-oss-120b` (antes: qwen/qwen3.6-27b — thinking mode consumia todo max_tokens; antes disso llama-4-scout — descontinuado)
- **Contexto web:** `openai/gpt-oss-120b`
- **NLU / classificação:** `llama-3.1-8b-instant` com `MaxTokens` reduzido (50 para DetectIntent, 100–150 para outros)
- System prompts: nunca repetir regras no user content — cada turno só no system
- `buildUserContent` deve conter apenas o conteúdo, sem regras duplicadas
- `classifyPromptMode` retorna `ModeBrief` por padrão (mais econômico)
- Busca web: `keywords.go` roda primeiro (grátis); Groq só como fallback se ambíguo
- `shouldSearchWeb` via Groq: `max_tokens: 10`, retorna só "yes"/"no"

### NLU / DetectIntent — regras de desambiguação
A lista de comandos reconhecidos fica em `internal/domain/ia/commands.go` (única
fonte de verdade). O prompt do `DetectIntent` em `internal/domain/ia/intent.go`
usa `{commandList}` placeholder inserido via `strings.ReplaceAll` — nunca
hardcode a lista no texto do prompt. Ao adicionar comando novo, edite
`nluCommands` + `nluCommandDesc`.

**Formato do prompt (compressão):** O prompt está comprimido para minimizar
tokens de entrada (~1715 chars estáticos). Ao adicionar comandos novos, siga
o formato denso: 1 linha por comando com variações separadas por `/`, sem
linhas extras. JSON example explícito só para comandos com args complexos
(agenda, contagem, feriado). Regras de desambiguação em notação compacta
`gatilho→comando` na seção DESAMBIGUAÇÃO, não em frases longas. Não adicione
exemplos repetitivos que demonstrem o mesmo padrão com wordings diferentes —
o modelo generaliza com poucos exemplos densos.

As regras abaixo estão codificadas no system prompt de `DetectIntent` em
`internal/domain/ia/intent.go`. Ao alterar o prompt, mantenha esta documentação
sincronizada.

| Gatilho | Contexto → Comando | Contexto → Outro comando |
|---|---|---|
| `coloca` | música/som/banda → **play** | efeito (echo, reverb, robot) → **efeito** |
| `manda` | música/som → **play** | lembrete/aviso/recado → **agenda** |
| `conta` | piada/me faz rir → **piada** | fato/curiosidade → **fato**; dias/tempo → **contagem** |
| `transforma` | foto/imagem → figurinha → **sticker** | figurinha/sticker → foto → **unsticker** |
| `quantos dias` / `faltam` | contagem regressiva → **contagem** | aviso/lembrete → **agenda** |
| `cotação` de | dólar/euro/moeda → **cotacao** | produto/serviço → vazio |
| pergunta climática curta | `chove?` `faz frio?` `vai ter temporal?` → **clima** | — |

Fallback para ambíguos: `coloca` → play, `manda` → agenda, `conta` sozinho → piada.

### Tavily
- `Country: "brazil"` em toda busca
- `IncludeDomains` só em buscas de preço/produto (não em buscas gerais)
- `enrichQueryBR`: adiciona "preço brasil reais" automaticamente em `isPriceQuery`
- `Days`: 3 para news, 7 para preço, 0 para geral
- `IncludeRawContent: true`, `IncludeAnswer: true`
- Score mínimo: `0.3` para filtrar resultados

### Memória da IA
- Duas camadas coexistem: `chat_summaries` (resumo textual) + `user_facts` (fatos atômicos)
- `user_facts`: fatos discretos por `(chat, user_jid)` com campo `confidence`
- `NeedsSummary`: checa count de mensagens E `updated_at` do resumo (cooldown 2h)
- `refreshChatSummary`: assíncrono, dedup por chat, `max_tokens: 350`
- `extractAndStoreFacts`: assíncrono, dedup por `chat+user`, `max_tokens: 200`
- Fatos expirados (30 dias sem update) removidos pelo `StartCleanup`
- `formatMessagesForSummary`: incluir sender no label para grupos

### Scheduler
- Ticker: **15 segundos** (não 1 minuto)
- Interface `Job`: `Name() string`, `Next(time.Time) time.Time`, `Run(ctx) error`
- Jobs one-shot: `Next()` retorna `time.Time{}` após execução → removidos automaticamente
- `DynamicJob`: persistido em `storage/dynamic_jobs.json` via `DynamicStore`
- Jobs expirados (`RunAt.Before(now)`) são ignorados na carga inicial
- Log obrigatório: início, sucesso, erro e remoção de cada job

### Agenda
- `parseAgendaTime`: suporta ISO8601, `DD/MM HH:MM`, `DD/MM/YYYY HH:MM`,
  `D de mês HH:MM`, `D de mês de YYYY HH:MM`, tempo relativo ("daqui 5 minutos")
- `normalizePtMonths`: lowercase + substituição antes do `time.Parse`
- Datas sem hora: assume 08:00 local
- Datas sem ano: assume ano atual; se já passou, avança para o próximo ano
- Limite: máximo 30 dias à frente
- Subcomandos: `lista` (jobs futuros ordenados por RunAt) e `remover <número>`
- `message := strings.Join(args[1:], " ")` — nunca `args[1]`

### Audio / Play
- PTT (`AudioMessage` + `PTT: true`): usado em mambo, dio, cafe, efeito, scheduler jobs
- Música (`!play`): `DocumentMessage` com `FileName`, `Mimetype`, `JpegThumbnail`
  - Upload via `whatsmeow.MediaDocument`
  - `extractCoverArt`: ffmpeg `-an -vcodec mjpeg -vframes 1`, retorna nil se falhar
  - `FileName`: `"Artista - Título.ogg"`, fallback `"audio.ogg"` se ambos vazios
- OGG para WhatsApp: `-c:a libopus -b:a 64k -ar 48000 -ac 1`

### WhatsApp / whatsmeow
- JID: sempre usar `ToNonAD()` para comparações e logs
- Grupos: JID termina em `@g.us`; usuários em `@s.whatsapp.net`
- `@all` nativo: `ExtendedTextMessage` com texto `"@all"` e `ContextInfo.NonJIDMentions = 1`
- Fingerprint SSL: confirmar na primeira conexão cliente→servidor
- Status updates: `evt.Info.Chat` = `status@broadcast` (Server: "broadcast")
  - Filtro em `HandleMessage`: `chat.Server == "broadcast"` descarta status e broadcast lists
  - Respostas a status são visíveis publicamente — nunca responder
  - Broadcast lists também usam Server `"broadcast"` mas com User diferente de `"status"`

### Album (múltiplas mídias)
- WhatsApp entrega cada item do album como `*events.Message` separado
- Primeiro item: `AlbumMessage` protobuf com `ExpectedImageCount` + `ExpectedVideoCount`
- Itens subsequentes: `MessageAssociation` (type `MEDIA_ALBUM`) com `ParentMessageKey` apontando pro pai
- `whatsmeow` **não** reagrupa albums — cada item é evento independente
- `AlbumCoordinator` em `internal/commands/album.go`:
  - Bufferiza itens até todos chegarem (ou timeout 10s)
  - `pendingAlbum`: map[parentMsgID] → items, expected, timer
  - Itens filho sem texto são capturados antes do guard `msg == ""` no router
- `BatchHandlerFunc` em `types.go`: assinatura para handlers de album
- `RegisterBatchCommand` no Router: registra handler normal + batch
- Comando `!sticker` registrado via `RegisterBatchCommand` com `StickerAlbumCommand`
- Quando `handlePrefixedCommand` detecta `AlbumMessage` + `BatchHandler != nil`:
  - Chama `albumCoordinator.Bufferize(parentID, evt, cmdName, args, expected)`
  - Se todos chegaram, `Dispatch` → `dispatchBatch` → `BatchHandler`
  - Senão, timer de 10s dispara dispatch parcial
- Limite: `albumMaxItems = 30` itens por album

---

## Padrões de código

### Handler público
```go
func XyzCommand(ctx context.Context, client *whatsmeow.Client,
    evt *events.Message, args []string) error {
    // validação de args
    // delegação para domain
    // whatsapp.Reply(...)
}
```

### Handler com dependências (closure)
```go
func XyzHandler(dep *domain.Dep, logger *zap.Logger) commands.HandlerFunc {
    return func(ctx context.Context, client *whatsmeow.Client,
        evt *events.Message, args []string) error {
        return XyzCommand(ctx, client, evt, args, dep, logger)
    }
}
```

### Registro em app.go
```go
r.RegisterCommand(commands.CommandMeta{
    Name:        "xyz",
    Description: "Descrição do comando",
    Type:        commands.CommandTypeUtility,
    Private:     false,
}, public.XyzHandler(dep, logger))
```

### Job do scheduler
```go
type XyzJob struct { ... }
func (j *XyzJob) Name() string            { return "xyz" }
func (j *XyzJob) Next(now time.Time) time.Time { ... }
func (j *XyzJob) Run(ctx context.Context) error { ... }
```

---

## O que NÃO fazer

- Não usar `http.DefaultClient` — sempre criar `&http.Client{Timeout: ...}`
- Não usar `go func()` direto — sempre `gosafe.Go`
- Não usar `display_name` cru do Nominatim no `DisplayName`
- Não repetir regras de comportamento no user content se já estão no system prompt
- Não deixar `max_tokens` em aberto em chamadas de API
- Não deixar `<think>` de modelos de raciocínio vazar — `callGroq` já aplica
  `ReasoningFormat: "hidden"` + `stripThinkTags`; se montar `IARequest`
  manualmente (ex: `DetectIntent`), aplicar `stripThinkTags` no response
- Não adicionar abstrações desnecessárias — só remover as que atrapalham
- Não modificar handlers existentes para adicionar nova feature — criar novo ou extender
- Não commitar a pasta `storage/` nem binários de `bin/`

---

## Melhorias pendentes (tracking)

### Feito nesta sessão (2026-06-30)
- `handler.go`, `client.go`, `stats.go`: 3 raw `go` convertidas para `gosafe.Go` com logger injetado
- `nlu.go:163` e `router.go:254`: erros silenciosos de handler/reply agora logados com `r.log.Warn`
- `nlu.go` (230 linhas) dividido em `nlu.go` (dispatch) + `nlu_detect.go` (heurísticas e detecção)
- `tavily.go` (169 linhas) dividido em `tavily.go` (core search) + `tavily_query.go` (helpers de query)
- Doc comments adicionados em toda a API pública do `Router`, `Scheduler`, `Job`, `Config`, `Handler`

### Feito nesta sessão (2026-07-01)
- `dispatchableCommand` duplicado removido: exportado `ia.DispatchableCommand()`, commands/nlu.go
  usa a centralizada em vez de manter cópia própria em nlu_detect.go
- `tavily.go`: criava `http.Client{Timeout: 10s}` local — agora recebe `*http.Client` via parâmetro
- `router.go`: hooks recebiam `context.Background()` — agora passam contexto com 30s de timeout
- `jid.go`: `ResolveContactName` e `ResolvePNFromLID` usavam `context.Background()` —
  agora aceitam `ctx context.Context` como primeiro parâmetro
- `ia/ia.go` (169L → 83L): tipos movidos para `models.go`, `callGroq` para `groq.go`,
  `baseSystemMessages` e `appendPersistentAndRecent` para novo `context.go`
- `ia/summary.go` (175L → 145L): `formatMessagesForSummary` extraído para `summary_format.go`
- `weekday/job.go Run()`: IIFEs anônimas removidas — `cancel()` explícito após cada operação
- `weatherHandler` local em `app_commands.go` substituído por `public.WeatherHandler()` exportado
- `configs/config.go`: adicionadas `mapstructure` tags em GroqConfig, TavilyConfig, MusicConfig, OwnerConfig
- `config.example.yaml`: seção de variáveis de ambiente documentada
- `admin/debug.go` (235L → 126L): `resolveTargetJID`, `pushNameFromEvent`, `WhoisHandler`
  extraídos para `whois.go`

### Feito nesta sessão (2026-07-02)
- `commands.go` criado: fonte única de verdade para NLU commands (`nluCommands` + `nluCommandDesc`);
  `DispatchableCommand` migrado de switch para `map[string]struct{}` com `sync.Once`;
  `buildNLUPromptSection()` gera lista de comandos para o prompt
- `intent.go` reescrito: template compacto (~500 tokens, down from ~1700), MaxTokens 150→50,
  placeholders via `{today}`/`{tomorrow}`/`{commandList}`, extração JSON limpa sem `extractJSON`
- Weather bug fix: `WeatherCommand` (guard utf8.RuneCountInString < 2), `Lookup` (guard
  strings.TrimSpace vazio), prompt de DetectIntent (regra "NUNCA invente cidade")
- Prompt compression: exemplos de detectIntent convertidos de JSON-por-linha para formato denso
  `input→args`, redundâncias removidas, ~1200→~500 tokens
- **Part 3.1 — strings.Builder simplificados** em 9 arquivos: whois.go, debug.go, birthday/job.go,
  birthday/handler.go, ia/commands.go, ia/prompts.go, ia/tavily.go, feriado.go, agenda.go,
  cotacao.go, admin/memoria.go, domain/history/memory.go (FormatFacts + FormatAtomicFacts)
- Mantidos como strings.Builder: menu.go (nested loop complexo), message_history.go (loop-based),
  summary_format.go (loop), normalize.go (char filtering), ia/prompts.go (já é retorno direto)
- **Part 3.2 — Go idioms**: código já usa `slices.Contains` e `max()` nativamente; nenhum
  novo candidato encontrado para `slices.Contains`, `min`/`max` ou `maps.Copy`
- AGENTS.md atualizado com convenção weather empty-location guard e sistema commands.go

### Feito nesta sessão (2026-07-02, reply-context)
- `message_text.go`: adicionados `ExtractContextInfo` e `ExtractQuotedText` públicos
- `nlu_detect.go`: adicionado `isReplyToBot` — checa `ContextInfo.Participant` vs bot JID
- `isNLApplicable`: nova heurística (2) para reply-to-bot, antes do `nlpGroupTrigger`
- `intent.go`: `DetectIntent` aceita `quotedContext`, injeta como "Contexto da conversa"
  no user content quando presente; regra 7 adicionada ao prompt
- `nlu.go`: `handleNaturalLanguage` extrai quoted text e passa para `DetectIntent` e
  `handleShinobuMention`; `handleShinobuMention` anexa contexto ao prompt no fallback
- AGENTS.md atualizado com seção reply-to-bot

### Feito nesta sessão (2026-07-02, weather card — single-day)
- `card.go` + `card_icons.go`: gerador de card PNG via `fogleman/gg` — gradiente,
  ícones vetoriais (sol/nuvem/chuva/neve/tempestade/nevoeiro), temperatura,
  localização, sensação/umidade/vento
- `fonts/Inter-{Regular,Bold}.ttf` baixadas em `assets/fonts/` e cacheadas via `sync.Once`
- `whatsapp/image.go`: `SendImageBytes` conveniência upload+send
- `weather.go`: `WeatherCommand` tenta card visual, fallback para texto se falhar
- `app_commands.go`: `WeatherHandler` agora recebe logger
- AGENTS.md: seção Weather atualizada com card visual e gg

### Feito nesta sessão (2026-07-02, 5-day vertical card)
- **Location bug diagnosed:** sem shadowing/stale value em WeatherCommand — `loc.DisplayName`
  vem sempre de `results[0]` do `geo.Lookup`. "São Paulo" em queries não relacionadas era
  Nominatim retornando state como fallback quando city/town/village vazios. Adicionado log
  de debug `"clima: cidade resolvida"` para traceabilidade.
- **`card.go` reescrito:** `GenerateCard` → `GenerateForecastCard(forecasts, location, country)`
  — layout vertical 5 dias: header (130px) + rows (100px cada), fundo gradiente neutro,
  fundo alternado entre rows, divisores, indicador de precipitação
- **`card_icons.go`:** line widths now proportional to size (`size * factor`) — icons scale
  cleanly at 50px for day rows
- **`weather/weather.go`:** `DailyForecast` struct + `dailyResponse` + `GetDailyForecast`
  (`daily` param, independente de hourly) + `fetchDaily` (mesmo padrão de fetchHourly)
- **`weather.WeekdayPT`:** exportado em `card.go` — reusável por outros pacotes (ex: texto
  fallback em weather.go)
- **`commands/public/weather.go`:**
  - Sem data → `GetDailyForecast` + `GenerateForecastCard` + `buildForecastText` fallback
  - Com data (`-YYYY-MM-DD`) → `GetForecastForDate` + texto (legado)
  - `buildForecastText`: fallback textual de 5 dias com labels, emoji, temp, precip
  - `logger.Debug("clima: cidade resolvida", ...)` para rastrear location vs query
- AGENTS.md: seção Weather atualizada com 5-day card, location bug fix, WeekdayPT

### Feito nesta sessão (2026-07-02, weather card — hero+list)
- **Location bug re-verified:** flow traceado — `loc := results[0]` é a única atribuição, sem
  shadowing, sem stale value, sem variável intermediária. Root cause confirmado como Nominatim
  retornando state name quando city/town/village vazios. Guards já existentes são suficientes.
- `apparent_temperature_max` adicionado aos params daily da Open-Meteo: `DailyForecast.ApparentTempMax`,
  `dailyResponse.Daily.ApparentTemperatureMax`, `GetDailyForecast` popula o campo, `fetchDaily` URL
  atualizada — permite hero mostrar "Sensação X°" sem segunda chamada de API.
- **`card.go` reescrito com hero + list:** layout vertical: hero (220px, gradiente por WMO code)
  + daily list (4 rows de 90px, fundo sólido escuro). `drawHeroBackground`, `drawHeroSection`,
  `drawBigIcon`, `drawDailyList`, funções auxiliares de row. "Hoje" só no hero, nunca repetido na lista.
- **`GenerateForecastCard` signature:** `(forecasts, current *WeatherResult, location, country)`
  — `current` é opcional (nil = usa forecasts[0] como hero). Handler passa nil (single-call approach).
- **Hero icon:** 100px (vs 50px nas rows), desenhado à direita (~620px).
- **`WeatherCommand`:** chamada única `GetDailyForecast(ctx, lat, lon, 5)`, passa `nil` para `current`.
- AGENTS.md: seção Weather atualizada com hero+list, fonte gofont, signature.
- `go build ./...` e `go vet ./...` passam sem erros.

### Feito nesta sessão (2026-07-04, NLU bare-mention fix)
- **Root cause confirmada:** dupla — (1) sinal fraco: `DetectIntent("Shinobu", "")` sem conteúdo
  faz o modelo "chutar" um comando qualquer; (2) vazamento de contexto: reply a mensagem do bot
  com apenas "Shinobu" injetava o texto anterior (ex: piada) no `DetectIntent`, fazendo o modelo
  re-despachar o mesmo comando.
- **`isBareMention`** adicionado em `nlu_detect.go`: roda ANTES de qualquer classificação ou
  extração de contexto. Remove "shinobu" e o JID do bot, verifica se sobrou <3 runas —
  mensagens só com o nome do bot vão direto pra conversa, sem passar pelo `DetectIntent`.
- **`hasActionableContent`** adicionado em `nlu_detect.go`: só injeta `quotedContext` no
  `DetectIntent` se a mensagem atual tiver ≥2 palavras. Impede que saudações ("tudo bem?")
  recebam contexto que contaminaria a classificação.
- **Prompt hardening** em `intent.go`: regra 8 adicionada — "se for apenas saudação/nome do
  bot/sem pedido claro, retorne SEMPRE `{"command":"","args":[]}`" com exemplos explícitos.
- `go build ./...` e `go vet ./...` passam.

### Feito nesta sessão (2026-07-04, multi-intent + Groq 429 + card text measurement)
- **Multi-intent precedence (Part 1):** regra 9 adicionada ao prompt de `DetectIntent` em `intent.go` —
  "se múltiplos pedidos, retorne APENAS o primeiro comando na ordem de leitura". Exemplos concretos
  adicionados no bloco de exemplos do prompt.
- **Groq 429 retry with backoff (Part 2):**
  - `RateLimitError` type em `groq.go` com error message Shinobu-toned
  - `groqChatRaw` (HTTP real, sem retry) + `groqChat` wrapper com retry (2 tentativas, backoff
    exponencial 2s→4s→10s max, ctx cancel respect)
  - 429 detection no status check de `groqChatRaw` — retorna `RateLimitError` em vez de generic error
  - Todos os 7 call sites (`callGroq` via `AskIA`/`QuickChat`, `DetectIntent`, `classifyNeedsWebSearch`,
    `extractAndStoreFacts`, `generateConversationSummary`) passam por `groqChat` e herdam o retry
- **Weather card text measurement (Part 3):**
  - `card_text.go` criado com `truncateToWidth` (binary-search truncation) e `measureWidth`
  - `faceBold20` + `faceReg16` adicionados em `card.go` para fallback de tamanho
  - `drawHeroSection`: label cidade+país medido em `faceBold26`, fallback para `faceBold22` → `faceReg20`
    com truncamento se exceder 725px
  - `drawDayLabel`: truncamento via `truncateToWidth` se exceder `rowLabelW` (120px)
  - `drawDayDescription`: truncamento via `truncateToWidth` se exceder `rowDescMaxW` (260px)
- **AGENTS.md limpo:** referência stale a `extractJSON(s)` substituída pela convenção real de
  `json.Unmarshal` direto
- `go build ./...` e `go vet ./...` passam.

### Feito nesta sessão (2026-07-04, prompt compression)
- `intent.go` promptTmpl comprimido: 2860→1715 chars estáticos (-40%), 46→38 linhas
  - Exemplos: formato denso 1 linha/comando com `/` para variações
  - JSON examples mantidos só para comandos com args complexos (agenda, contagem, feriado)
  - Regras de desambiguação (9 preservadas) consolidadas em notação `gatilho→comando`
  - Seção DATAS enxugada — commandList já descreve formato de args
  - Segurança e regras de bugfix (bare mention, multi-intent, context) preservadas
- AGENTS.md: nota de convenção de formato comprimido adicionada para futuros comandos

### Feito nesta sessão (2026-07-04, Qwen3 think tags fix)
- `models.go`: adicionado campo `ReasoningFormat` ao `IARequest` (`json:"reasoning_format,omitempty"`)
- `groq.go`: `isReasoningModel()` detecta qwen/gpt-oss; `callGroq` define `ReasoningFormat: "hidden"`
  para modelos de raciocínio e aplica `stripThinkTags` no response — fix central, todos os
  callers herdam automaticamente
- `utils.go`: `stripThinkTags` — regex `(?s)<think>.*?</think>` como safety net
  contra vazamento residual de blocos de raciocínio
- `intent.go`: `DetectIntent` aplica `stripThinkTags` no content antes do parse JSON (safety net)
- `groq.go` e `models.go`: gofmt aplicado (alinhamento de campos do struct)

### Feito nesta sessão (2026-07-04, intent kill-switch + model revert)
- **Model revert:** `ModelScoutFast` de `qwen/qwen3.6-27b` → `openai/gpt-oss-120b` —
  Qwen thinking mode consumia todo max_tokens com `reasoning_format: hidden`, resultando
  em respostas vazias. GPT-OSS-120b é estável e já provado como `ModelWebStrong`.
- **ReasoningEffort:** `IARequest.ReasoningFormat` renomeado para `ReasoningEffort` com valor
  `"low"` — minimiza tokens de raciocínio, maximiza tokens de resposta. `stripThinkTags`
  mantido como safety net.
- **Intent kill-switch:** `bot.intentEnabled` (default: `false`) adicionado a `BotConfig`,
  `Router` e `buildRouter`. `handleNaturalLanguage` pula `DetectIntent` quando `false` —
  mensagens com menção vão direto para `handleShinobuMention` (conversa IA).
  Todo o código de Intent permanece intacto, pronto para reativar.
- Arquivos alterados: `models.go`, `groq.go`, `config.go`, `config.example.yaml`,
  `router.go`, `nlu.go`, `app.go`

### Feito nesta sessão (2026-07-17, album sticker support)
- **Album Coordinator** (`internal/commands/album.go`): buffering de itens de album
  - Detecta `AlbumMessage` (primeiro item) e `MessageAssociation` (itens filhos)
  - Bufferiza até todos chegarem ou timeout 10s, limite 30 itens
  - Thread-safe com `sync.Mutex`, `dispatched` flag evita double-dispatch
- **BatchHandlerFunc** (`internal/commands/types.go`): novo tipo para handlers de album
- **RegisterBatchCommand** (`internal/commands/router.go`): regista handler + batch
- **Router album detection** (`internal/commands/router.go`):
  - Itens filho (`HasMediaAlbumAssociation`) são capturados antes do guard `msg == ""`
  - `handlePrefixedCommand` detecta `AlbumMessage` + `BatchHandler` → redireciona pro coordinator
  - `dispatchBatch` despacha com timeout de 120s
- **StickerAlbumCommand** (`internal/commands/public/sticker_batch.go`):
  - Processa múltiplos itens em lote, envia status único
  - Continua processando mesmo se itens individuais falharem
  - Reporta ok/fail ao final
- **Comando sticker**: registrado via `RegisterBatchCommand` com `StickerAlbumCommand`
- Arquivos criados: `album.go`, `sticker_batch.go`
- Arquivos alterados: `types.go`, `router.go`, `app_commands.go`, `AGENTS.md`
- `go build ./...` e `go vet ./...` passam sem erros

### Ainda pendente
- `configs/config.go`: usar `mapstructure` tags nos BindEnv paths para alinhar com struct layout
- `history/message_history.go StartCleanup`: migrar para `gosafe.Go`
- `ia/search.go`: auditar se `shouldSearchWeb` e `keywords.go` coexistem no hot path
- Memória em grupos: separar resumo por usuário dentro do mesmo chat de grupo