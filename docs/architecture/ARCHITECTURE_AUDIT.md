# Shinobu-Whatsapp — Auditoria Arquitetural

> Auditoria de **2026-11-10**, commit `0a60afa` (`v2.23.2`). Somente leitura —
> nenhum arquivo funcional foi alterado.
> Evidências em `arquivo:linha`. Achados marcados **[C]** = comprovado por leitura
> do código; **[S]** = suspeita que depende de condição externa não observada.

**Legenda de severidade:** 🔴 crítica · 🟠 alta · 🟡 média · ⚪ baixa

---

## 1. Resumo executivo

O projeto é um **monólito Go maduro e internamente coerente**, não um acúmulo
aleatório. O que já está certo e deve ser preservado:

- ✅ Zero `http.DefaultClient`; todo request usa `NewRequestWithContext` + client com `Timeout`.
- ✅ Zero goroutine crua — 100% via `gosafe.Go` (com `recover`).
- ✅ Zero `max_tokens` em aberto (11/11 chamadas Groq definem).
- ✅ Zero segredo em log; `.env`/`config.yaml` fora do git.
- ✅ Execução de binário externo via argv (sem shell) — sem command injection.
- ✅ Classificação de erro do yt-dlp com mensagem amigável por categoria.
- ✅ Kill de grupo de processos no yt-dlp (`Setpgid` + `SIGKILL`).
- ✅ Limpeza de temporários em 11/11 pontos de criação.
- ✅ `go vet` limpo; testes verdes; AGENTS.md como memória do projeto.

Os problemas **comprovados** concentram-se em cinco áreas: (1) **pânico por
indexação sem guarda** em integrações, (2) **sucesso falso reportado ao
usuário** em comandos admin, (3) **colisão de temporários por PID**, (4)
**ausência de tratamento de eventos de conexão do WhatsApp**, e (5)
**configuração com duas fontes de verdade divergentes**.

Três achados adicionais **críticos** apareceram na camada de identidade e
descoberta:

- **I-01** — três convenções incompatíveis de "owner" (JID completo × só
  dígitos × `Sender.User` cru) convivem no mesmo código, com **duas
  configurações** para o mesmo conceito (`UsersJID.Owner` e `Owner.Number`).
- **I-02** — `Router.SetBotJID` **nunca é chamado**: `r.botJID` fica `""` para
  sempre, e com ele a detecção de menção por `@jid` e o tratamento de "bare
  mention".
- **A-08** — `ErrNotACommand` só é tratado no caminho NLU: `!cotacao <moeda
  inválida>` termina em **silêncio absoluto** para o usuário.

**Nenhum achado exige infraestrutura nova.** Todos cabem em correções
localizadas dentro do processo único atual.

---

## 2. Problemas por área

### 2.1 Confiabilidade — pânicoss e respostas silenciosas

| ID | Sev. | Problema | Evidência | Consequência | Correção |
|---|---|---|---|---|---|
| A-01 | 🟠 | Indexação de arrays da Open-Meteo sem bounds check | `domain/weather/weather.go:193-197` e `:258-262` — só `ApparentTemperatureMax` tem guarda; `WeatherCode[i]`, `Temperature2MMax[i]`, `Temperature2MMin[i]`, `PrecipitationProbMax[i]` não | Resposta 200 com `daily` parcial → `index out of range` → panic recuperado pelo `gosafe.Go` (`bot/handler.go:51`) → **usuário não recebe nada** | Validar `len()` de cada array antes do loop (ou `min()` dos comprimentos) |
| A-02 | 🟠 | `GenerateForecastCard` acessa `forecasts[0]` sem validar tamanho | `domain/weather/card.go:86-91` | `!clima` com `daily.time = []` → panic → card não enviado, sem mensagem | Guard `len(forecasts) == 0` → fallback textual (o `weather.go` já tem `buildForecastText`) |
| A-03 | 🟠 | noteapi reporta **sucesso falso** | `domain/noteapi/noteapi.go:112-134` (`do` retorna `err == nil` para qualquer status HTTP) + `commands/admin/note.go:55-58, 72-75, 88-95, 107-112` (condição `if callErr != nil && code != 0` nunca é verdadeira) | Owner recebe "✅ API reiniciando" quando a chamada **falhou** | `if callErr != nil \|\| code != http.StatusOK` (1 linha por handler) |
| A-04 | 🟠 | Temporários com nome derivado de `os.Getpid()` | `domain/music/audio_effects.go:147`, `domain/sticker/convert.go:33` | Duas requisições concorrentes (`!efeito`, `!sticker`) escrevem no **mesmo arquivo** → `-y` sobrescreve e o `ReadFile` pode ler o output alheio → **usuário recebe mídia processada de outra requisição** | `os.CreateTemp` (nome único) — o padrão já usado em `music/cover.go:16` e `sticker/convert.go:21` |
| A-05 | 🟡 | `moveFile` contradiz o próprio comentário | `domain/music/ytdlp_exec.go:285-305`: comentário diz "destino ocupado: chamador serve o arquivo já existente", código **retorna erro** → `ErrUnknown` → `domain/music/service.go:52-58` cai no **fallback remoto** e devolve mp3 em vez de m4a | Duas requisições da mesma música: uma "vence", a outra vai pro túnel legado e retorna formato diferente | Alinhar código e comentário: erro de `O_EXCL` deve ser tratado como sucesso (usar o arquivo existente) |
| A-06 | 🟡 | `webpmux` executado sem `CommandContext` | `domain/sticker/convert.go:135` (`exec.Command`), enquanto yt-dlp/ffmpeg usam `exec.CommandContext` | Sem timeout/cancelamento; se travar, fica órfão (não há `Setpgid`/kill como no yt-dlp) | `exec.CommandContext` + `Setpgid`/kill de grupo, como `ytdlp_exec.go:131-140` |
| A-07 | 🟡 | `StartMemoryCleanup` **nunca é chamado** | `domain/history/memory.go:296-314` (só a definição); `app/app.go:61` chama apenas `store.StartCleanup` | Tabela `user_memory` cresce sem limite | Chamar `store.UserMemory.StartMemoryCleanup(ctx, 30*24h)` no boot |
| A-08 | 🟠 | `ErrNotACommand` só é tratado no caminho **NLU** | `commands/types.go:13` (sentinel), `commands/nlu.go:89` (tratado) — no caminho com prefixo, `commands/router.go:356-364` apenas loga e retorna | `!cotacao xyz` (moeda inválida, `commands/public/cotacao.go:37`) → **usuário recebe silêncio absoluto** | Tratar `ErrNotACommand` também no `router.go` (responder "não entendi a moeda") |

### 2.1b Achados adicionais (segunda rodada de auditoria)

| ID | Sev. | Problema | Evidência | Consequência | Correção |
|---|---|---|---|---|---|
| A-09 | 🔴 | **DSN do `config.example.yaml` é incompatível com o driver atual** | `config.example.yaml` (bloco `database`): `?_foreign_keys=on` (sintaxe do **mattn**) — o `modernc` só entende `?_pragma=…` (`modernc.org/sqlite@v1.50.0/sqlite.go:136-166`) | `foreign_keys` fica **OFF** → `container.Upgrade` aborta com *"foreign keys are not enabled"* (`whatsmeow/store/sqlstore/container.go:100-110`) → **o bot não sobe** em deploy novo feito a partir do exemplo | Alinhar o exemplo ao driver em uso (`?_pragma=foreign_keys(1)`) |
| A-10 | 🟠 | **Handlers registrados depois do `Connect`** | `app/app.go:70` (`client.Connect`) vs `:148-149` (`RegisterHandlers`) — entre os dois, o cliente já está conectado e a lib despacha eventos para uma lista **vazia** (`whatsmeow/client.go:924-936`) | Mensagens do início do processo e o **backlog do offline-sync são descartados silenciosamente** | Registrar os handlers **antes** do `Connect` |
| A-11 | 🟡 | `storage/` não é garantido: nenhum `MkdirAll` e está no `.gitignore` | `app/app.go:48-51` (Ping falha se o dir não existe), `:54`; caminhos relativos ao CWD | Deploy limpo falha no boot com erro pouco claro | `os.MkdirAll("storage", 0o755)` antes de abrir os bancos |
| A-12 | 🟡 | Limite de 50 MB da mídia é avaliado **depois** do download completo | `integration/media/download.go:12,53-62` (o corpo inteiro já está na RAM quando o check roda) | Pico de RAM real num host de 512 MB com handlers concorrentes | Checar o tamanho durante a leitura (ou baixar para arquivo e medir) |
| A-13 | 🟡 | `DynamicJob` só marca `done` em sucesso → **re-execução a cada 15 s** | `domain/scheduler/dynamic_job.go:41-46,67-71`; `domain/scheduler/scheduler.go:183` | Lembrete que falha (WhatsApp fora) é reenviado indefinidamente, a cada tick | Marcar `done` também em falha (com log) ou limitar tentativas |
| A-14 | ⚪ | `yt-dlp` sem `Pdeathsig` (o ffmpeg tem) | `domain/music/ytdlp_exec.go:131-133` vs `infra/ffmpeg/linux_process.go:10-14` | No shutdown, o `yt-dlp` pode ficar órfão no container | `Pdeathsig: syscall.SIGTERM` no `SysProcAttr` do yt-dlp |
| A-15 | 🟡 | `mikael.initSchema` só **loga** o erro e o app segue; `migrateLegacyData` roda a **cada boot** | `domain/mikael/store.go:56-69,75-132` | Bot rodando sem a tabela `word_counts` (comandos falham depois); varredura O(n) no startup | Propagar o erro de schema; migrar só quando necessário |
| A-16 | ⚪ | `SetMaxOpenConns(1)` justificado por comentário **factualmente errado** ("WhatsApp Client é single-threaded") | `infra/database/database.go:48-49`; a lib **é** concorrente (`whatsmeow/client.go:869-895` spawna goroutine por node) | Serializa todo o I/O do store numa conexão; comentário engana quem mexer | Corrigir o comentário; avaliar 2–4 conexões |
| A-17 | ⚪ | Globais que crescem sem podar: `searchCache` (IA), `dupCache` (comandos), `birthday.js` (singleton de pacote) | `domain/ia/search.go:22-25`, `commands/public/shinobu.go:25-28`, `domain/birthday/store.go:22` | Vazamento lento de memória; `birthday.js` viola a regra de "sem estado global" do AGENTS.md | Poda por TTL / injeção de dependência |

### 2.2 Identidade, permissões e disparo do NLU

| ID | Sev. | Problema | Evidência | Consequência | Correção |
|---|---|---|---|---|---|
| I-01 | 🔴 | **3 convenções incompatíveis de "owner"** no mesmo código | `commands/middleware.go:64-66` compara o **JID completo** (`…@lid`); `domain/sticker/handler.go:26` e `domain/birthday/handler.go:203-213` comparam **só dígitos** (`phone.NormalizeNumber`); `commands/public/shinobu.go:45-46` compara `Sender.User` cru contra `cfg.Owner.Number` | `cfg.UsersJID.Owner` (`config.yaml:19` = `276807035179046@lid`) e `cfg.Owner.Number` (env `OWNER_NUMBER`) são **duas configurações para o mesmo conceito**. No endereçamento LID, `Sender.User` é o LID — o "modo mestre" da IA (`ia/context.go:17-22`) pode nunca ser ativado e o owner pode ser barrado dos comandos `Private` **[S: depende do modo de endereçamento]** | Uma função única de identidade (`auth.IsOwner(sender)`, com número **e** JID), usada por todos os pontos |
| I-02 | 🔴 | **`Router.SetBotJID` nunca é chamado** | `commands/router.go:79-82` (só a definição; nenhum call site) → `r.botJID == ""` permanentemente | `isMentioned` nunca detecta menção por `@jid` (`commands/nlu_detect.go:132`, curto-circuito `botJID != ""`); `isBareMention` nunca remove o JID do texto (`nlu_detect.go:162-167`) → reply só com o JID do bot não é classificado como menção simples | Chamar `r.SetBotJID(client.WAClient.Store.ID.ToNonAD().String())` no boot (o ID já está disponível no `Store`) |
| I-03 | 🟠 | **Lista de ignorados provavelmente nunca casa** | `commands/admin/ignore.go:42` grava `phone.NormalizeNumber(...)` (só dígitos); `domain/ignore/store.go:45-51` é consultado com o **JID completo** e com `Sender.User` (`commands/router.go:256`) | No endereçamento LID, `Sender.User` (LID) ≠ número gravado → o `!ignorar` **não bloqueia ninguém** **[S]** | Normalizar nos dois sentidos (número **e** JID) ou gravar as duas formas |
| I-04 | 🟡 | Rate limit é **global por sender**, não por chat | `commands/router.go:267-270` (chave = `Sender.ToNonAD().String()`); `commands/ratelimit.go:48-67` | 11 mensagens num grupo bloqueiam o usuário por 1 min **em todos os chats**; não diferencia comando caro (Groq) de comando local | Chave por (sender, chat) + limite diferenciado por categoria de comando |
| I-05 | ⚪ | `!manutencao` e `!ignorar` têm **escopo global** | `commands/router.go:104-107` (`atomic.Bool` global), `commands/admin/manutencao.go:16-25`, `domain/ignore/store.go:9` | Manutenção affects todos os grupos de uma vez | Manter global (é o desejado do owner) — **documentar** |


### 2.2 WhatsApp / whatsmeow — ciclo de vida e conexão

| ID | Sev. | Problema | Evidência | Consequência | Correção |
|---|---|---|---|---|---|
| W-01 | 🟠 | **Nenhum handler de evento de conexão** | `bot/handler.go:50-56` só trata `*events.Message`. Sem `Disconnected`, `LoggedOut`, `StreamReplaced`, `ConnectFailure`, `TemporaryBan` | Bot fica **offline silencioso** após queda de rede ou sessão substituída: o processo segue "running", sem log, sem aviso ao owner | Registrar handler desses eventos: log `Warn` + avisar o owner + (opcional) `EnableAutoReconnect = true` |
| W-02 | ⚪ | **Correção de auditoria:** o auto-reconnect **já é `true` por padrão** na lib (`whatsmeow/client.go:284`) — o bot reconecta sozinho | `whatsmeow/client.go:284` (`EnableAutoReconnect: true`), `:611-620` (`autoReconnect` com backoff) | O problema real **não é reconectar**, é **não saber**: sem handler de evento, a queda passa despercebida (W-01). `InitialAutoReconnect` é `false` (`:77`) → falha de rede no **startup** mata o processo | Manter o default; resolver a visibilidade em W-01. Se quiser sobreviver a falha no boot: `InitialAutoReconnect = true` |
| W-03 | 🟡 | Handler de mensagem cria goroutine **sem limite** | `bot/handler.go:51` (`gosafe.Go` por mensagem) | N mensagens ⇒ N goroutines; com `!play` (até 240 s) isso pode abrir dezenas de downloads — o semáforo do `music` limita a 1, mas as goroutines ficam esperando | Limite de concorrência no router (semáforo pequeno, ex. 8) ou fila curta |
| W-04 | ⚪ | `ctx` do handler não é cancelável no shutdown | `commands/router.go:352` cria `context.WithTimeout(context.Background(), 180s)` | Trabalho em andamento não é interrompido no desligamento | Derivar de um `ctx` raiz cancelável (ver R-01) |
| W-05 | ⚪ | `!restart` usa `syscall.Exec` (sem chance de flush do socket) | `commands/admin/restart.go:22-36` | Reinício abrupto; sem `Disconnect` completo nem checkpoint do WAL | Preferir `Disconnect()` + saída limpa supervisionada |

**Constatação importante (verificada no fonte do whatsmeow
`v0.0.0-20260713112832`):** `dispatchEvent` invoca os handlers **de forma
síncrona** (`client.go:923-937`) e, para mensagens, a chamada vem de
`message.go:839` no loop de leitura do websocket. Como o `EventHandler` do
projeto despacha para uma goroutine (`bot/handler.go:51`), **o read loop não
bloqueia** por comandos lentos — o keepalive (`keepalive.go:30`, goroutine
própria) continua respondendo. Ou seja: **o risco não é travar a conexão, é
multiplicar trabalho concorrente**. Isso rebaixa W-03 de crítico para médio e
confirma que a decisão de despachar em goroutine foi acertada.

### 2.3 Persistência (SQLite)

| ID | Sev. | Problema | Evidência | Consequência | Correção |
|---|---|---|---|---|---|
| P-01 | 🟡 | `history.Store` sem `SetMaxOpenConns` nem `busy_timeout` | `domain/history/message_history.go:45` (`sql.Open("sqlite", path)`) — enquanto o banco principal usa `SetMaxOpenConns(1)` (`infra/database/database.go:49`) | Escritas concorrentes (hooks, resumo, fatos, scheduler) → `SQLITE_BUSY` **[S]** | `SetMaxOpenConns(1)` + `?_pragma=busy_timeout(5000)` no DSN |
| P-02 | 🟡 | Dois bancos SQLite para o mesmo domínio | `storage.db` (só whatsmeow, 22 MB) e `message_history.db` (dados do app, 1,1 MB) | Duas pools, doisschemas, backup em dois arquivos; `storage.db` é o ativo crítico (sessão) e o app não escreve nele | Manter separado (é correto: sessão ≠ domínio), mas **documentar** e unificar a política de pragmas |
| P-03 | ⚪ | DSN divergente entre `config.yaml` e `config.example.yaml` | `config.yaml:18` (`?_pragma=foreign_keys(1)`) vs `config.example.yaml:18` (`?_foreign_keys=on`, sintaxe do driver **mattn**, não do modernc) | Quem copiar o exemplo perde `foreign_keys` silenciosamente | Alinhar o exemplo ao driver em uso |
| P-04 | ⚪ | Schema do app sem versionamento | `domain/history/message_history.go` cria tabelas com `CREATE TABLE IF NOT EXISTS` | Mudança de coluna exige migração manual | Manter assim por ora (5 tabelas estáveis); introduzir versionamento só se mudar o schema |
| P-05 | ⚪ | `!restart` não fecha o DB (sem checkpoint do WAL) | `commands/admin/restart.go:36` | WAL cresce até o próximo restart; SQLite se recupera, mas o arquivo `-wal` fica no disco | Aceitável; documentar |

### 2.4 Configuração

| ID | Sev. | Problema | Evidência | Consequência | Correção |
|---|---|---|---|---|---|
| C-01 | 🟠 | **Duas fontes de verdade divergentes** | `infra/configs/config.go:140-163`: `Groq`, `Tavily`, `Music`, `Owner`, `Feriados` são lidos de `viper.GetString("apicalls.*")` — prefixo `apicalls` que **não existe** no YAML. Só as env vars funcionam | Groq/Tavily/Feriados **não são configuráveis por YAML**, apesar de `config.example.yaml` documentá-lo; quem editar o YAML acha que configurou e não configurou | `mapstructure` tags corretas (`groq`, `tavily`, `feriados`) + `BindEnv` nos mesmos paths; eliminar o bloco manual |
| C-02 | 🟡 | `Load()` não valida nada | `infra/configs/config.go:71-170` — sem erro de retorno, sem validação de DSN, owner JID, API keys, timezone | Bot sobe "saudável" e falha só no uso; `log.level: debug` mascara | `Load() (*Config, error)` + validação mínima (owner, DSN, prefix) e **fail-fast** com mensagem clara |
| C-03 | 🟡 | `NormalStickerChance == 0` é sobrescrito para `0.2` | `infra/configs/config.go:135` | impossível desabilitar stickers aleatórios por config | usar *ponteiro* ou sentinel (`-1`) para distinguir "não definido" de "zero" |
| C-04 | ⚪ | `viper.AutomaticEnv()` sem efeito + prefixo vazio | `infra/configs/config.go:79-81` | Falsa sensação de que qualquer env var é lida | remover ou configurar `SetEnvKeyReplacer` explicitamente |
| C-05 | ⚪ | `log.Fatal` da stdlib dentro do carregamento de config | `infra/configs/config.go:127` | saída sem log estruturado nem flush do zap | devolver erro e deixar `app.Run` decidir |
| C-06 | ⚪ | AGENTS.md afirma algo que o código não faz | `AGENTS.md:333` ("adicionadas mapstructure tags em GroqConfig…") vs `configs/config.go:140-163` | documentação engana quem vai mexer | atualizar o AGENTS.md junto com a correção de C-01 |

### 2.5 Integrações externas

| ID | Sev. | Problema | Evidência | Consequência | Correção |
|---|---|---|---|---|---|
| I-01 | 🟡 | Retry de 429 no Groq **amplifica** consumo | `domain/ia/groq.go:19-22` (2 retries, backoff 2s→4s→10s) aplicado a classificador (`ia/search.go:132`), resposta (`ia/ia.go:61`), resumo (`ia/summary.go:143`) e fatos (`ia/facts.go:81`) | Até **12 requisições Groq por mensagem** numa rajada de 429, contra o orçamento de 1K RPD documentado em `ia/models.go:12-20` | Retry só na resposta principal; classificador/resumo/fatos sem retry (ou 1 tentativa) |
| I-02 | 🟡 | `ia.Config.Log` não é setado no caminho `!shinobu` | `commands/public/shinobu.go:77-82` monta `ia.Config` **sem `Log`** → `ia/summary.go:25-28` e `ia/facts.go:30-33` caem em `zap.NewNop()` | Falhas de resumo e extração de fatos **não são logadas** | reutilizar o `ia.Config` do router (já tem `Log`) em vez de criar outro |
| I-03 | 🟡 | Sem cache em geocoding, weather, cotação, feriado | `domain/geocoding/geocode.go:43-52`, `domain/weather/weather.go:175-209`, `domain/cotacao/cotacao.go:62-130`, `domain/feriado/feriado.go:62-93` | `!clima` repetido = N chamadas ao Nominatim (política de uso pede ≤1 req/s); cotação e feriado mudam 1x/dia | Cache em memória com TTL curto (clima/geocoding: 10 min; cotação/feriado: 6 h) |
| I-04 | 🟡 | `io.ReadAll` sem limite em respostas externas | `domain/weather/weather.go:235,301,345,379`, `domain/ia/groq.go:124`, `domain/music/ytdlp_request.go:49`, `domain/cotacao/cotacao.go:78` — enquanto `geocode_helpers.go:19` faz certo (`LimitReader` 1024) | Resposta grande Allocate RAM desnecessariamente e impede reuse da conexão | `io.LimitReader` com teto por integração |
| I-05 | ⚪ | `http.Client` novo por mensagem no caminho `!shinobu` | `commands/public/shinobu.go:81` | Descarta o pool de conexões; handshake TLS novo por mensagem | reutilizar o client do router |
| I-06 | ⚪ | `strconv.ParseFloat` de lat/lon com erro descartado | `domain/geocoding/geocode.go:106-107` | coordenada inválida vira `(0,0)` silenciosamente (Golfo de Guinea) | checar o erro e devolver erro de geocoding |
| I-07 | ⚪ | Cotação com campo malformado vira `R$ 0,00` | `domain/cotacao/cotacao.go:116-121,133-139` | usuário vê zero como se fosse preço real | sinalizar "indisponível" quando o parse falha |
| I-08 | ⚪ | `wttr.in` sem `User-Agent` | `domain/weather/weather.go:362-371` (todos os outros clientes setam `Shinobu-Whatsapp/1.0`) | fallback pode devolver HTML/403 | adicionar o UA ou trocar o fallback |
| I-09 | ⚪ | `DenoPath` sem default apesar do comentário | `domain/music/exec_config.go:17` vs `:26-43`; `config.example.yaml:52` comenta a chave | em deploy com o exemplo, yt-dlp roda **sem runtime JS** → erros `Sign in to confirm` | definir o default (`./bin/deno`) |
| I-10 | ⚪ | `checkDeps` exige yt-dlp/deno mesmo com `localEnabled: false` | `app/app.go:200-215` (vs checagem granular em `app_commands.go:67-73`) | o bot **não sobe** sem binários que não vai usar | tornar a checagem condicional ao modo de música |
| I-11 | ⚪ | `Cache.Hit` é código morto | `domain/music/music_cache.go:83-92` | documentação do cache não corresponde à implementação | remover ou implementar |

### 2.6 Organização, dependências e acoplamento

| ID | Sev. | Problema | Evidência | Consequência | Correção |
|---|---|---|---|---|---|
| O-01 | 🟡 | Domínio acoplado ao whatsmeow | 8 arquivos de `internal/domain/*` importam `go.mau.fi/whatsmeow` (`domain/birthday/{handler,job}.go`, `domain/mikael/store.go`, `domain/scheduler/dynamic_job.go`, `domain/sticker/{handler,send,store}.go`, `domain/weekday/job.go`); 66 arquivos no total | Regras de negócio não são testáveis sem a lib; jobs do scheduler dependem de `*whatsmeow.Client` | **Não** criar uma camada de abstração de WhatsApp. Extrair apenas o que é testável e caro de simular (envio de mensagem) — o resto pode continuar tipado com whatsmeow |
| O-02 | 🟡 | `internal/integration/whatsapp` é uma abstração **parcial** | 14 arquivos de helpers, mas `commands/types.go:19` (`HandlerFunc`) e todos os handlers recebem `*whatsmeow.Client` e `*events.Message` | Duas "portas" para a mesma coisa: uma fina (helpers) e outra crua (assinaturas) | Manter a assinatura crua (é idiomático no whatsmeow) e tratar o pacote `whatsapp` como **camada de envio** (o que ele já é). Documentar a fronteira |
| O-03 | 🟡 | Erro de handler **não vira mensagem ao usuário** | `commands/router.go:356-363` — o erro é apenas logado (`r.log.Error`) e o fluxo termina | Falha silenciosa para quem usa o bot: "não aconteceu nada". O padrão correto já existe em `commands/public/messages.go` | Responder com mensagem amigável por categoria (reaproveitar o padrão de `domain/music` `ErrCategory`/`CategoryText`) antes de logar |
| O-04 | ⚪ | `internal/app` concentra a composição de tudo | `app/app.go` (216 linhas) + `app_commands.go` (236) + `app_commands_admin.go` (169) | `app_commands.go` passa 8 dependências por função | Extrair a struct de dependências do registro de comandos (1 struct, sem mudar a ordem de inicialização) |
| O-05 | ⚪ | Comando admin `noteapi` (notebook) é legado do fallback de música | `domain/noteapi` + 6 comandos em `commands/admin/note.go`; o `!play` local substituiu o túnel (`config.yaml:42` "fallback LEGADO") | Código de um subsistema que talvez não exista mais | Decidir com o dono: manter (é útil p/ operar o notebook) ou remover |
| O-06 | ⚪ | `cmd/shinobu` (CLI) duplica fluxo do Makefile | `cmd/shinobu/work.go:9-20` cria `exec.Cmd` para `git`/`go`/`make` | Duas fontes de verdade para o fluxo de release | Manter a CLI delegando ao Makefile (já faz) e documentar |
| O-07 | ⚪ | `gosafe.Go` sem `WaitGroup`/cancelamento | `infra/gosafe/safe.go:9-22` | Não é possível esperar tarefas no shutdown | Adicionar `gosafe.Group` (WaitGroup + recover) **apenas se** o shutdown precisar esperar (ver R-01) |
| O-08 | ⚪ | `time.AfterFunc` do album sem `recover` | `commands/album.go:140-142` → `dispatchBatch` → `BatchHandler` | Panic nesse caminho **derruba o processo** (único caminho sem `gosafe.Go`) | Envolver o dispatch em `gosafe.Go` |
| O-09 | ⚪ | Comentário com caracteres CJK corrompidos | `domain/scheduler/scheduler.go:175` ("disparar哪怕 1s") | ruído de revisão automática | corrigir o texto |

### 2.7 Concorrência, recursos e ciclo de vida

| ID | Sev. | Problema | Evidência | Consequência | Correção |
|---|---|---|---|---|---|
| R-01 | 🟠 | **Shutdown não cancela o contexto raiz** | `app/app.go:52` (`ctx := context.Background()`); `Scheduler.Stop()` existe (`domain/scheduler/scheduler.go:88`) e **nunca é chamado**; `store.StartCleanup(ctx, 24h)` usa o mesmo ctx | Scheduler, cleanup e hooks continuam vivos durante o desligamento; `bot.Listen()` retorna e o processo morre sem esperar | `ctx, cancel := context.WithCancel(...)`; `defer cancel()`; chamar `sched.Stop()` e `client.Disconnect()` no `defer` |
| R-02 | 🟡 | Jobs do scheduler usam `context.Background()` | `domain/scheduler/scheduler.go:207` (`checkAndRun`) | Job em andamento ignora o cancelamento do scheduler | derivar do `ctx` do scheduler |
| R-03 | 🟡 | Scheduler executa jobs **na goroutine do ticker** | `domain/scheduler/scheduler.go:198-215` (mutex liberado, mas o ticker espera) | Um job lento atrasa todos os outros (ticker de 15 s) | Executar o job em goroutine com `gosafe.Go` (já há recover no local) |
| R-04 | ⚪ | `io.ReadAll` do arquivo inteiro na RAM | `domain/music/service.go:89` (até 25 MB) + `domain/music/cover.go:16,71` (cópias) | ~50–75 MB de pico por música (medido: pico real de 66 MB/24 h) | Aceitável no limite atual de 512 MB; **documentar o预算** e manter `maxFileMB: 25` |
| R-05 | ⚪ | `middlewares` usam `context.Background()` sem timeout | `commands/middleware.go:47,70` | goroutine potentially presa se o WhatsApp travar | `context.WithTimeout` curto |

### 2.8 Observabilidade e experiência do usuário

| ID | Sev. | Problema | Evidência | Consequência | Correção |
|---|---|---|---|---|---|
| U-01 | 🟠 | Falha de handler não chega ao usuário | `commands/router.go:356-363` | "não aconteceu nada" | usar o padrão de `messages.go` (já existem 41 mensagens prontas) |
| U-02 | 🟡 | `handlerTimeoutMention` (30 s) é curto para o pipeline de IA | `commands/router.go:22`; o pipeline soma classificador (3 tentativas + 6 s de backoff) + Tavily (30 s) + Groq (3 tentativas + 6 s) — `ia/search.go:132`, `ia/ia.go:61`, `ia/groq.go:63-87` | Em rajada de 429 o usuário recebe "Algo deu errado" mesmo com a API recuperável | Dar timeout próprio à menção (ex. 90 s) ou reduzir as tentativas do classificador |
| U-03 | ⚪ | `!shinobu` cria `http.Client` por mensagem e `ia.Config` sem `Log` | `commands/public/shinobu.go:77-82` | custo de conexão + falhas invisíveis | reutilizar config do router |
| U-04 | ⚪ | Corpo bruto de resposta externa logado em Debug | `domain/cotacao/cotacao.go:83-86`, `domain/ia/facts.go:99` | ruído e PII potencial em log | truncar/classificar antes de logar |
| U-05 | ⚪ | Argumentos de comando logados crus | `commands/router.go:315` (`zap.Strings("args", args)`), `commands/nlu.go:108,133` | texto do usuário em log (PII) | logar só o nome do comando + tamanho |
| U-06 | ⚪ | `msg` truncado por bytes (não runes) | `commands/nlu_detect.go:55` | rune partido no log | truncar por runes |
| U-07 | 🟡 | Falha ao persistir histórico/fatos é **descartada** | `commands/public/shinobu.go:88` (`_ = store.Save(...)`), `domain/ia/summary.go:85` (`store.SetSummary`), `domain/ia/facts.go` | A IA "responde" mas a memória **não é gravada** — o bot esquece sem ninguém saber | logar o erro (warn) e contabilizar em `!stats` |
| U-08 | 🟠 | **Limite de tamanho de texto do WhatsApp não é aplicado** | `integration/whatsapp/text.go:15-29` (`Text: proto.String(text)` sem truncamento); o único truncador é o de debug (`commands/admin/debug.go:17-22`, 3000 chars) | `!agenda lista` lista **todos** os lembretes (`commands/public/agenda.go:146-161`), `!memoria` soma fatos de N usuários (`commands/admin/memoria.go:72-94`), `!menu` cresce com o nº de comandos → acima de ~4096 chars o WhatsApp **rejeita a mensagem** | Truncar no `SendText` (limite seguro, ex. 3500) com aviso "(continua...)" |
| U-09 | 🟡 | **Erro interno vaza ao usuário** em 6 pontos | `commands/admin/ignore.go:32,45`, `commands/admin/jobs.go:82`, `commands/admin/debug.go:70`, `domain/sticker/handler.go:70,87` (`"❌ Erro ao …: "+err.Error()`) | Contradiz a política declarada em `commands/public/messages.go:3` ("sem expor detalhes internos") | Usar a taxonomia de falha (E5) — mensagem amigável + log técnico |
| U-10 | 🟡 | Logs **contraditórios** no mesmo comando | `commands/router.go:357` (Error "Erro no comando") seguido de `commands/router.go:366` (Info "Comando executado") — o handler devolve `nil` depois de responder a falha | O log sugere sucesso onde houve falha | Só logar "Comando executado" quando `err == nil` |
| U-11 | ⚪ | Erros silenciados com `_ =` (13 pontos) | `commands/public/shinobu.go:62,67,89`, `domain/ignore/store.go:42`, `domain/ia/summary.go:85`, `domain/history/memory.go:305`, `domain/sticker/handler.go:111`, `commands/public/play.go:36-39` | Perda de histórico, de memória e da lista de ignorados **sem nenhum sinal** | log em nível warn (não é erro fatal) |
| U-12 | ⚪ | Cache de duplicatas cresce sem limite | `commands/public/shinobu.go:97-119` (`setCachedAnswer` só adiciona; o TTL só impede o reuso) | Vazamento lento de memória; e a mesma pergunta de 2 pessoas no mesmo chat em 30 s devolve a resposta idêntica | Evicção por TTL no `setCachedAnswer` |
| U-13 | ⚪ | Ctx dos hooks é **cancelado imediatamente** | `commands/router.go:261-265` (`hookCancel()` logo após os hooks) | O `INSERT` do mikael (`domain/mikael/store.go:145-151`) pode ser abortado → mensagens perdidas | Timeout próprio do hook (ex. 5 s), cancelado só no fim |
| U-14 | ⚪ | Panics **sem stack trace** | `app/app.go:162` (`DisableStacktrace = true`) + `infra/gosafe/safe.go:17` (`zap.Any("panic", r)`) | Panic recuperado fica sem rastro executável | `zap.Stack("stack")` no recover |
| U-15 | ⚪ | `!shutdown` chama `os.Exit(0)` **dentro do handler** | `commands/admin/shutdown.go:28` | `defer logger.Sync()` (`app/app.go:44`) e `defer db.Close()` (`app/app.go:52`) **nunca rodam** → log final perdido, WAL sem checkpoint | Sinalizar o `app.Run` para encerrar (canal) em vez de `os.Exit` |
| U-16 | ⚪ | Descoberta de comandos incompleta | Só existe `!menu` (`app/app_commands.go:28-32`); **69 aliases** não aparecem no menu (`app/app_aliases.go:7-108` vs `commands/public/menu.go:80-98`); comandos `Private` são ocultados (`menu.go:36-38`) → as categorias "🛡️ Administração"/"👑 Owner" ficam **sempre vazias** | O owner não descobre `!stats`, `!jobs`, `!memoria`; o usuário não descobre `!p` = `!play` | Mostrar aliases no menu e uma seção de comandos privados para owner/admin |
| U-17 | ⚪ | `!ajuda` não existe | grep por `ajuda\|help` → nenhum comando registrado | Quem não sabe o prefixo não tem ponto de entrada | Adicionar `!ajuda` como alias de `!menu` (1 linha) |

### 2.9 Testabilidade

| ID | Sev. | Problema | Evidência | Correção |
|---|---|---|---|---|
| T-01 | 🟡 | Cobertura **0%** em `app`, `bot`, `commands/admin`, `configs`, `database`, `media`, `whatsapp` | `go test -cover ./...` | Testes de contrato para: parsing de config, mapping de erro → mensagem, e uma "fita" do router com client falso |
| T-02 | 🟡 | `internal/commands` com 29,2% e `domain/ia` com 5,1% | idem | O NLU e a decisão de rota são a parte mais sensível do produto — testes de tabela para `DetectIntent` (JSON) e para o roteamento |
| T-03 | ⚪ | Sem `go test -race` no fluxo | `Makefile` só tem `go test ./...` | Rodar `-race` no CI (barato para este tamanho de projeto) |
| T-04 | ⚪ | Testes de integração de API real sem gate visível | `domain/cotacao/integration_cotacao_test.go`, `domain/geocoding/integration_geo_test.go`, `domain/music/integration_ytdlp_test.go` | Manter build-tag `integration` + documentar como rodar |
| T-05 | 🟠 | **`go test -race ./...` falha** | Reproduzido: `go test -race ./internal/commands/` → `FAIL ... race detected during execution of test` — `commands/album_test.go:225` (escrita na goroutine do timer) vs `commands/album_test.go:231` (leitura na principal) | A race **é no teste**, não em produção — mas o `-race` nunca roda (`Makefile:43,68` só `go test ./...`), então nem isso aparece | Corrigir o teste (mutex como em `album_test.go:170-201`) e **adotar `-race` no CI** |
| T-06 | 🟡 | **Nenhum teste** do pipeline de roteamento | `commands/router.go` (418 L), `commands/middleware.go`, `commands/nlu.go`, `commands/nlu_detect.go` — 0 testes | A parte mais sensível do produto (decidir o que fazer com uma mensagem) não tem rede de segurança | Testes de tabela para `runMiddlewares`, `resolveAlias`, `isNLApplicable`, `isBareMention` |
| T-07 | 🟡 | `HandlerFunc` recebe `*whatsmeow.Client` **concreto** | `commands/types.go:17` | Nenhum handler é testável sem client real — `internal/integration/whatsapp` (que faz o envio) tem 0 testes | Injetar um `Sender` mínimo **apenas onde o teste exige** (menu, agenda, clima); não criar abstração de WhatsApp completa |
| T-08 | ⚪ | `gofmt` pendente | `go fmt -l .` → `internal/infra/configs/config.go` (espaço em `cfg.Music = MusicConfig {`, linha 148) | hygiene | `gofmt -w` |

### 2.10 Build e deploy

| ID | Sev. | Problema | Evidência | Consequência | Correção |
|---|---|---|---|---|---|
| D-01 | 🟡 | Binário de 25 MB **versionado no git** | `Makefile:74` (`git add -A` inclui `Shinobu`), `.gitignore:11` lista `/Shinobu` — contradição entre regra e prática | Repo cresce a cada release; `git clone` fica pesado | Decidir: versionar (build local) ou compilar no container. **Não** mudar sem o dono — o plano da SquareCloud não compila em container |
| D-02 | ⚪ | 216 MB de binários externos (`bin/`) fora do git | `bin/` ignorado; `scripts/setup.sh` baixa yt-dlp/deno/ffmpeg/webpmux | Setup manual em cada ambiente novo; risco de drift de versão entre ambientes | Fixar versões no `setup.sh` (hoje usa `latest`) |

---

## 3. O que **não** é problema (verificado)

Para evitar retrabalho, registram-se como **corretos** (mantidos):

- `internal/commands` pipeline: middlewares em ordem previsível, rate limit por sender, filtro de status/broadcast, album coordinator com timeout — bempensado.
- `domain/music`: execução de processo externo é exemplar (`CommandContext`, argv, semáforo, `Setpgid`+kill, timeout, limite de tamanho duplo, temp dir por tarefa, erros categorizados).
- `domain/ia`: `max_tokens` sempre definido, retry com backoff que respeita `ctx`, `stripThinkTags`, degradação silenciosa quando Tavily falha (usuário não vê erro).
- `domain/history`: cleanup diário implementado (`StartCleanup`), cobertura 57%.
- `domain/scheduler`: `Job` interface mínima, jobs one-shot removidos ao término, restore de jobs dinâmicos com expiração.
- `infra/ffmpeg`: `Pdeathsig` (filho morre com o pai) + `nice`.
- `internal/integration/whatsapp`: helpers de envio corretos e reutilizados.
- `commands/public/messages.go`: 41 mensagens amigáveis já escritas na voz da Shinobu — a base da melhoria de UX já existe.
- `commands/public/weather.go:97-108` — `replyOrUnderlying` é o **melhor padrão do repositório**: responde a mensagem amigável **e** devolve o erro interno para o router logar. Deveria ser o padrão de todos os handlers.
- `commands/router.go`: pipeline com ordem previsível (broadcast → album → texto → ignore → hooks → rate limit → prefixo/NLU), middlewares que **resolvem alias antes de decidir** (`router.go:179,185`), e `CommandNotFound`/`Private` que **respondem e bloqueiam** em vez de falhar calado.
- `commands/app_aliases.go`: alias é validado no registro (não colide com comando e o alvo precisa existir) e descartado com log de warn se inválido.
- `internal/infra/ffmpeg`: `Pdeathsig` (filho morre com o pai) + `nice` — impede órfãos.
- `internal/infra/gosafe`: todo `go` do projeto passa por ele (13 call sites), com `recover` — **exceto** o `time.AfterFunc` do album (O-08).
- `commands/album.go`: buffer com mutex, `dispatched` idempotente, despacho **fora** do lock, timeouts diferenciados (10 s com `expected` conhecido, 3 s sem).

**Estado real de produção:** `config.yaml:6-7` deixa `nlpGroupTrigger: false` e
`intentEnabled: false` → o caminho de `DetectIntent` (`domain/ia/intent.go`) está
**desligado em produção**; o código do NLU por intenção é maintenance-ready,
não o caminho ativo. Isso reduz o custo por mensagem (≈3 chamadas Groq em vez
de 4) e muda a prioridade das correções de NLU.

---

## 4. Suspeitas (precisam de validação em runtime)

| ID | Sev. | Suspeita | Como validar |
|---|---|---|---|
| S-01 | 🟡 | `SQLITE_BUSY` no `message_history.db` sob concorrência (P-01) | Reproduzir com N goroutines de `Save` em teste de integração; adicionar `busy_timeout` de qualquer forma |
| S-02 | 🟡 | Pipeline da IA estourando o timeout de 30 s em rajada de 429 (U-02) | Log de `duration` já existe em `router.go:366`; monitorar p95 |
| S-03 | ⚪ | Colisão de temporário (A-04) ocorrendo em produção | Busca por `audio-out-<pid>`/`sticker-out-<pid>` no tmp do contêiner |
| S-04 | ⚪ | Fallback remoto do `!play` acionado por `moveFile` (A-05) | Logar `category` (já existe em `ytdlp_exec.go:159`) e correlacionar com `ErrUnknown` |
| S-05 | ⚪ | Crescimento de `user_memory` (A-07) | `SELECT count(*) FROM user_memory` no contêiner |

---

## 5. Melhorias opcionais (não custam o plano)

| ID | Sev. | Melhoria | Esforço |
|---|---|---|---|
| M-01 | ⚪ | Teste `-race` no CI (`make test` → `go test -race ./...`) | pequeno |
| M-02 | ⚪ | `logs` com `sampling` do zap (o nível `debug` em produção gera volume) — `config.yaml:16` está em `debug` | pequeno |
| M-03 | ⚪ | Métricas internas simples (contadores em memória expostos no `!stats`): mensagens/segundo, erros por subsistema, fila do NLU | médio |
| M-04 | ⚪ | Correlação por `operationID` nos logs de download/tradução (o `music` já tem `ErrCategory`) | médio |
| M-05 | ⚪ | `README` com o mapa de responsabilidades atual (hoje descreve o MVP) | pequeno |

---

## 6. Débito técnico que pode esperar

| ID | Item | Por que esperar |
|---|---|---|
| D-01 | Migrações versionadas do schema do app | 5 tabelas estáveis; só vale se o schema mudar |
| D-02 | Repository pattern para o histórico | `history.Store` já isola o SQL e tem 57% de cobertura |
| D-03 | Fila durável de mensagens | Um processo, 43 comandos, 1 sessão — não justifica |
| D-04 | Camada de abstração de WhatsApp completa | O whatsmeow já é a abstração; reimplementar seria duplicar |
| D-05 | Separar `!shinobu` (IA) em serviço | Monólito cabe no orçamento (66 MB de 512 MB) |
| D-06 | Reduzir `bin/` (216 MB) | É disco, não RAM; só pesa se o limite do plano apertar |
