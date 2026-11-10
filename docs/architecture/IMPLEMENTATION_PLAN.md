# Shinobu-Whatsapp — Plano de Implementação

> Derivado de `ARCHITECTURE_AUDIT.md` e `TARGET_ARCHITECTURE.md`.
> Cada etapa é **independente e publicável**. Nada avança sem aprovação.

---

## 0. Regras de execução (para todas as etapas)

- **Uma etapa por vez**, sem misturar refatorações não relacionadas.
- Antes de cada etapa: reler este plano e confirmar os arquivos afetados.
- Antes de cada etapa: `go build ./...`, `go vet ./...`, `gofmt -l .`,
  `go test ./...` (e `-race` a partir da E7).
- **Compatibilidade preservada**: nomes de comando, comportamento público, chaves
  de configuração existentes e o arquivo de sessão (`storage.db`) não mudam.
- **Publicação**: `make release` (build local → commit do binário → tag → push).
  Rollback = `git revert <commit>` + `make release` (ou `git revert` da tag).
- Cada etapa fecha com: testes, `go vet`, atualização do `AGENTS.md` (se muda
  convenção) e um resumo para o dono.
- **Nada de mudança de assinatura pública do WhatsApp** (item 14/15 do briefing):
  os handlers continuam recebendo `*whatsmeow.Client` e `*events.Message`.

---

## Etapa 1 — Não quebrar e não mentir ✅ (v2.23.3, 2026-11-10)

| Campo | Conteúdo |
|---|---|
| **Objetivo** | Eliminar pânico por indexação sem guarda, a colisão de temporários, as respostas de sucesso falso e o **silêncio** em comandos que falham |
| **Problemas** | A-01, A-02, A-03, A-04, A-05, A-06, A-07, **A-08**, **U-08** (+ I-07 parcial) |
| **Arquivos** | `domain/weather/weather.go`, `domain/weather/card.go`, `domain/noteapi/noteapi.go`, `commands/admin/note.go`, `domain/music/audio_effects.go`, `domain/sticker/convert.go`, `domain/music/ytdlp_exec.go`, `domain/history/message_history.go` (chamar cleanup), `app/app.go`, `commands/router.go` (ErrNotACommand), `integration/whatsapp/text.go` (truncar) |
| **Mudanças** | 1) `weather.go`: validar comprimento dos arrays antes do loop (usar o menor `len()`); 2) `card.go`: guard `len(forecasts)==0` → fallback textual; 3) `noteapi`: `do()` devolve erro quando `status != 200` **ou** handler admin passa a checar `code != http.StatusOK`; 4) temporários: `os.CreateTemp` em vez de `fmt.Sprintf("...-%d", os.Getpid())`; 5) `moveFile`: erro de `O_EXCL` tratado como sucesso (usar o arquivo já existente), alinhando código e comentário; 6) `webpmux`: `exec.CommandContext` + `Setpgid`/kill; 7) `StartMemoryCleanup` chamado no boot (`app.go`); 8) **router**: tratar `ErrNotACommand` no caminho com prefixo (hoje só o NLU trata) → responder "não entendi"; 9) **`whatsapp.SendText`**: truncar em ~3500 chars com "… (continua)" |
| **Dependências** | Nenhuma — é a primeira etapa |
| **Riscos** | Baixo. Ordem dos arrays da Open-Meteo muda? O guard usa o menor `len()`, então é seguro. `moveFile` semântica muda: o destino passa a ser reaproveitado (é o que o comentário já prometia) — risco: arquivo de cache corrompido anterior poderia ser servido; mitigação: validar tamanho do arquivo no `Get` do cache. Truncamento de texto muda mensagens longas (aceitável — antes o WhatsApp simplesmente recusava) |
| **Testes** | `weather`: tabela com `daily` vazio e com arrays de tamanhos diferentes (sem rede, structs direto); `card`: `GenerateForecastCard` com slice vazio não entra em panic; `noteapi`: `httptest` com 500 → erro; `music`: `moveFile` com destino existente → nil; `sticker`: dois `convert` concorrentes com o mesmo PID não colidem; `router`: handler que devolve `ErrNotACommand` → usuário recebe mensagem (não silêncio); `text`: texto de 9000 chars → enviado truncado |
| **Aceite** | `go test ./...` verde; nenhum `index out of range` possível nos paths de weather; `!play` concorrente não serve arquivo alheio; admin reporta falha quando a API falha; `!cotacao xyz` responde; nenhuma mensagem passa de ~3500 chars |
| **Rollback** | `git revert` do commit + `make release` |
| **Benefício** | Elimina falha silenciosa e resposta mentirosa — os dois piores sintomas para o usuário e para o dono |
| **Complexidade** | **Pequena** (~1 dia) |

---

## Etapa 2 — Identidade do owner e menção do bot ✅ (2026-11-10)

| Campo | Conteúdo |
|---|---|
| **Objetivo** | Uma única definição de "quem é o dono", e o bot capaz de reconhecer menções ao próprio JID |
| **Problemas** | **I-01** (3 convenções de owner), **I-02** (`SetBotJID` nunca chamado), **I-03** (lista de ignorados não casa) |
| **Arquivos** | `infra/configs/config.go` (novo `OwnerIdentity`), `commands/identity.go` (novo), `commands/middleware.go`, `commands/public/shinobu.go`, `domain/sticker/handler.go`, `domain/birthday/handler.go`, `domain/ignore/store.go`, `commands/admin/ignore.go`, `commands/nlu_detect.go`, `app/app.go` |
| **Mudanças** | 1) `config.OwnerIdentity{Number, JID}` — **preserva** `OWNER_JID` e `OWNER_NUMBER` (compatibilidade) e passa a ser a única fonte; 2) `commands.IsOwner(sender, owner)` aceita número normalizado **ou** JID; 3) substituir as 4 comparações divergentes por `IsOwner`; 4) `r.SetBotJID(client.WAClient.Store.ID.ToNonAD().String())` no boot; 5) lista de ignorados: gravar/consultar por **número e JID** |
| **Dependências** | Nenhuma (independente). Convém depois da E1 (que corrige o silêncio) |
| **Riscos** | **Médio** — mexe em **permissão**. Mitigação: a função aceita **os dois** formatos, então comportamento atual continua válido; e o middleware passa a permitir o owner nos **dois** formatos (hoje só num). Isso **amplia** acesso de admin, nunca restringe — e é o comportamento pretendido |
| **Testes** | `IsOwner`: tabela (número, JID com @lid, JID @s.whatsapp.net, não-owner); `ignore`: ignorar por número bloqueia pelo JID equivalente; `nlu_detect`: `isMentioned` com `botJID` preenchido detecta "@<jid>"; `isBareMention` remove o JID do texto |
| **Aceite** | `!stats` (privado) responde para o owner **independentemente** do formato configurado; o "modo mestre" da IA é ativado para o owner; `!ignorar <número>` bloqueia o contato; menção por `@jid` é reconhecida |
| **Rollback** | `git revert` + `make release`; env vars não mudam → rollback seguro |
| **Benefício** | Elimina o risco de o dono ficar **sem acesso ao próprio painel** e a falha silenciosa da lista de ignorados |
| **Complexidade** | **Pequena–média** (~1 dia) |

## Etapa 3 — Ciclo de vida e reconexão automática ✅ (2026-11-10)

| Campo | Conteúdo |
|---|---|
| **Objetivo** | O bot deve saber (e avisar) quando a conexão cai, reconectar sozinho e encerrar de forma ordenada |
| **Problemas** | W-01, W-02, W-04, R-01, R-02, R-03, R-05 |
| **Arquivos** | `bot/handler.go`, `bot/client.go`, `app/app.go`, `domain/scheduler/scheduler.go`, `commands/router.go` (ctx), `cmd/bot/main.go` |
| **Mudanças** | 0) **registrar os handlers ANTES do `Connect`** (`app/app.go:70` vs `:148-149`) — hoje as mensagens do início do processo e o backlog do offline-sync são descartados (A-10); 1) `bot.Handler`: tratar `events.Connected`, `Disconnected`, `LoggedOut`, `StreamReplaced`, `ConnectFailure`, `TemporaryBan` → log estruturado + aviso ao owner nos casos graves; 2) auto-reconnect já é default da lib (`whatsmeow/client.go:284`) — manter e **só** registrar os eventos; 3) `app.Run`: `ctx, cancel := context.WithCancel(...)`, `defer cancel()`, `defer sched.Stop()`, `defer client.Disconnect()` na ordem; 4) `bot.Listen(ctx)`: sai por `ctx.Done()` **ou** sinal; 5) scheduler: `checkAndRun` usa o `ctx` do scheduler (não `Background()`) e executa o job em goroutine própria com `gosafe.Go`; 6) `router`: `ctx` do handler derivado do `ctx` raiz (com timeout preservado); 7) `!shutdown` sinaliza o `app.Run` (canal) em vez de `os.Exit(0)` — assim os `defer` (logger.Sync, db.Close) rodam; 8) `yt-dlp`: `Pdeathsig: syscall.SIGTERM` no `SysProcAttr` (o ffmpeg já tem) para não deixar filho órfão no shutdown (A-14) |
| **Dependências** | Não depende da E1. Única dependência: o aviso ao owner usa `whatsapp.Reply`/`SendText` (disponível) |
| **Riscos** | **Médio** — toca no ciclo de vida. `EnableAutoReconnect` muda o comportamento em queda de conexão (reconecta sozinho; antes ficava offline). Se o `ctx` raiz passar a ser cancelado cedo demais, jobs param de rodar. Mitigação: `cancel()` só no `defer` do `Run` (após `Listen` retornar) |
| **Testes** | `bot.Handler`: tabela de eventos → (log, aviso, nenhuma ação); `scheduler`: job com `ctx` cancelado não executa; `app`: teste de composição com dependências fake verificando ordem de start/stop |
| **Aceite** | Desligar a rede por 30 s → o bot volta sozinho e deixa log de " reconectado"; sessão invalidada → owner recebe aviso; SIGTERM → processo encerra em < 5 s com log "encerrando" |
| **Rollback** | `git revert` + `make release`; a mudança é isolada em `bot/`+`app/`, revert sem impacto em dados |
| **Benefício** | Transforma "bot offline sem explicação" em "bot que avisa e volta" — o item nº 1 de confiabilidade |
| **Complexidade** | **Média** (~1–2 dias) |

---

## Etapa 4 — Configuração previsível

| Campo | Conteúdo |
|---|---|
| **Objetivo** | Uma fonte de verdade para configuração, validada na subida, sem quebrar quem já configurou por env |
| **Problemas** | C-01, C-02, C-03, C-04, C-05, C-06, I-09, I-10 |
| **Arquivos** | `infra/configs/config.go`, `config.example.yaml`, `config.yaml` (só se necessário), `app/app.go`, `AGENTS.md` |
| **Mudanças** | 0) **corrigir o DSN do `config.example.yaml`**: `?_foreign_keys=on` (sintaxe mattn) é ignorado pelo `modernc` → `foreign_keys` OFF → `container.Upgrade` falha → **o bot não sobe** em deploy novo (A-09, crítico); 1) `configs.Load()` passa a devolver `(*Config, error)`; 2) `Config` ganha `mapstructure` em `groq`, `tavily`, `feriados`, `owner`; 3) `BindEnv` usa os mesmos paths (`groq.apiKey`→`GROQ_API_KEY`), **mantendo** a leitura do path legado `apicalls.groq.apiKey` como fallback (compatibilidade); 4) remover o bloco manual `cfg.Groq = ...`; 5) validação mínima (prefixo, DSN, owner, timezone) com mensagem clara; 6) `NormalStickerChance`: `*float64` ou sentinel para permitir `0`; 7) `DenoPath` com default `./bin/deno`; 8) `checkDeps` condicional a `music.localEnabled`; 9) remover `viper.AutomaticEnv()` sem efeito; 10) `log.Fatal` → erro retornado; 11) `AGENTS.md`: corrigir a afirmação divergente |
| **Dependências** | Nenhuma (pode rodar antes da E2; se rodar depois, `Load()` com erro se integra ao fail-fast da E2) |
| **Riscos** | **Médio** — mexer em config pode trocar comportamento silenciosamente. Mitigação: ordem de precedência idêntica (env > YAML > default); teste de precedência obrigatório; `config.yaml` de produção **não** muda |
| **Testes** | `configs`: tabela (só YAML / só env / ambos → env vence); erro de validação com campo ausente; `normalStickerChance=0` respeitado; path legado `apicalls.groq.apiKey` ainda lido |
| **Aceite** | `GROQ_API_KEY` no env funciona como hoje; `groq.apiKey` no YAML passa a funcionar; bot com DSN inválido **não sobe** e diz por quê; nenhuma env atual deixa de ser lida |
| **Rollback** | `git revert` + `make release` (env vars não mudam → rollback seguro) |
| **Benefício** | Elimina a classe "achei que configurei, mas não" — uma das fontes mais caras de erro em produção |
| **Complexidade** | **Média** (~1 dia) |

---

## Etapa 5 — Persistência e integrações externas

| Campo | Conteúdo |
|---|---|
| **Objetivo** | Uma política de SQLite previsível, cache das consultas caras e respostas externas limitadas |
| **Problemas** | P-01, P-02, P-03, I-01, I-02, I-03, I-04, I-05, I-06, I-07, I-08, I-11 |
| **Arquivos** | `infra/database/database.go` (novo helper `Open`), `domain/history/message_history.go`, `domain/geocoding/geocode.go`, `domain/weather/weather.go`, `domain/cotacao/cotacao.go`, `domain/feriado/feriado.go`, `domain/ia/groq.go`, `domain/ia/tavily.go`, `domain/ia/search.go`, `domain/ia/summary.go`, `domain/ia/facts.go`, `commands/public/shinobu.go`, `infra/cache/` (novo, pequeno) |
| **Mudanças** | 0) `os.MkdirAll("storage", 0o755)` antes de abrir os bancos (deploy limpo falha no `Ping` — A-11); 0b) limite de 50 MB da mídia checado **durante** a leitura, não depois (A-12); 0c) `mikael.initSchema` propaga o erro e `migrateLegacyData` só quando necessário (A-15); 0d) índice em `messages(sent_at)` (a limpeza horária hoje varre a tabela toda); 1) `database.Open(path)`: PRAGMA `foreign_keys(1)`, `busy_timeout(5000)`, `SetMaxOpenConns(1)` — usado nos dois bancos; 2) alinhar `config.example.yaml` com o DSN real; 3) `infra/cache`: map+TTL (~60 linhas), usado por geocoding (10 min), clima (10 min), cotação (30 min), feriado (6 h); 4) `io.LimitReader` em toda leitura de body externo; 5) `shinobu.go`: reutilizar o `ia.Config`/client do router (com `Log`) em vez de criar outro; 6) retry do Groq só na resposta principal; 7) `geocode`: checar erro de `ParseFloat`; 8) `cotacao`: sinalizar valor indisponível em vez de `0,00`; 9) `wttr.in`: `User-Agent`; 10) remover `Cache.Hit` morto |
| **Dependências** | Nenhuma (independente). Convém rodar **depois** da E3 (a config do cache é nova) |
| **Riscos** | **Baixo–Médio**. Cache muda o comportamento (mesma pergunta por 10 min → resposta velha) — aceitável e desejado, mas precisa de TTL explícito e chave normalizada. `busy_timeout` muda o tempo de espera em lock (5 s) — melhor que erro imediato |
| **Testes** | `database.Open`: PRAGMA foreign_keys ativo (consulta real); `cache`: TTL expira, evict por chave; `geocode`: coordenada inválida → erro (não (0,0)); `cotacao`: campo ruim → "indisponível"; `shinobu`: `ia.Config` com `Log` não-nil |
| **Aceite** | `foreign_keys=1` nos dois bancos; `!clima São Paulo` 2× em 1 min → **1** requisição ao Nominatim; nenhum body externo lido sem limite; `!shinobu` não cria `http.Client` por mensagem |
| **Rollback** | `git revert` + `make release`; cache é em memória (some no restart) — nenhum risco de dados |
| **Benefício** | Menos chamadas externas (cota e latência), menos risco de lock, erros mais claros |
| **Complexidade** | **Média** (~2 dias) |

---

## Etapa 6 — Taxonomia de erro e resposta ao usuário

| Campo | Conteúdo |
|---|---|
| **Objetivo** | Nenhuma falha termina em silêncio; a mensagem diz o que houve, no tom da Shinobu |
| **Problemas** | O-03, U-01, U-02 (+ O-05 parcial) |
| **Arquivos** | **novo** `domain/failure/failure.go`, `domain/music/ytdlp_exec.go` (migrar `ErrCategory`), `domain/weather`, `domain/geocoding`, `domain/feriado`, `domain/cotacao`, `commands/router.go`, `commands/public/messages.go`, `commands/public/shinobu.go` |
| **Mudanças** | 1) `domain/failure`: `Category` + `Classify(err)` + `UserMessage()` — generalização do padrão que já existe no `music`; 2) `music`: `ErrCategory` vira alias/bridge para `failure` (mantém as 8 categorias específicas no log); 3) `weather/geocoding/feriado/cotacao`: classificar a falha; 4) `router`: erro não tratado → `failure.Classify(err).UserMessage()` → **responde ao usuário** e loga; 5) `handlerTimeoutMention`: 30 s → 90 s (o pipeline da IA precisa de 3 chamadas encadeadas); 6) **vazamento de erro interno**: `admin/ignore.go:32,45`, `admin/jobs.go:82`, `admin/debug.go:70`, `domain/sticker/handler.go:70,87` passam a usar a taxonomia (mensagem amigável + log técnico); 7) **descoberta**: `!ajuda` como alias de `!menu`; aliases exibidos no menu; seção de comandos privados visível para owner/admin (hoje as categorias "🛡️ Administração"/"👑 Owner" ficam sempre vazias); 8) `!menu` e `!agenda lista` com teto de itens (o texto já é truncado na E1) |
| **Dependências** | Depende da E1 (falhas reais classificadas primeiro). Sugestão: rodar depois da E5 (os domínios já terão erros melhor estruturados) |
| **Riscos** | **Médio** — muda o texto que o usuário vê. Mitigação: as mensagens existentes em `messages.go` **permanecem**; a nova camada só age em erros **não** tratados antes. Nenhum comando público muda de comportamento no caminho feliz |
| **Testes** | `failure`: tabela categoria ↔ mensagem; `router`: handler que devolve erro → usuário recebe mensagem e o erro é logado; `music`: as 8 categorias continuam mapeando para as mesmas mensagens (garantia de não regredir a UX já entregue) |
| **Aceite** | Nenhum comando falha em silêncio; `!clima` com API fora diz "não consegui agora, tenta daqui a pouco"; o tom continua sendo o da Shinobu |
| **Rollback** | `git revert` + `make release`; risco de UX baixo (reverte para silêncio, que é o estado atual) |
| **Benefício** | O usuário entende o que aconteceu; o operador distingue "YouTube recusou" de "sem internet" |
| **Complexidade** | **Média–grande** (~2–3 dias) |

---

## Etapa 7 — Limite de concorrência e recursos

| Campo | Conteúdo |
|---|---|
| **Objetivo** | Nenhuma rajada de mensagens cria trabalho ilimitado; nenhum processo filho fica órfão |
| **Problemas** | W-03, O-08, R-04 (documentar), A-08 |
| **Arquivos** | `commands/router.go`, `commands/album.go`, `infra/gosafe/gosafe.go`, `infra/ffmpeg/linux_process.go` |
| **Mudanças** | 1) semáforo de 4 handlers no router (configurável) com mensagem curta ao excedente; 2) `album.go`: dispatch por `time.AfterFunc` envolvido em `gosafe.Go` (hoje é o único caminho sem `recover`); 3) `gosafe.Group`: `WaitGroup` + `recover` para esperar tarefas no shutdown; 4) `ffmpeg`: `RunLowPriority` não sobrescrever `SysProcAttr` do caller (preservar `Setpgid`); 5) **scheduler**: corrigir a iteração de slice obsoleta durante unlock (`scheduler.go:178-225`) —_snapshot_ da lista de jobs antes de soltar o mutex — para impedir **execução duplicada** de job (envio duplicado de lembrete); 5b) `DynamicJob` marca `done` também em falha (hoje re-executa a cada 15 s indefinidamente — A-13); 6) `albumMaxItems`: cortar também os itens que excederem o limite depois da criação do pending; 7) rate limit por **(sender, chat)** em vez de sender global (`router.go:267-270`), com limite diferenciado para comandos caros (Groq); 8) documentar o orçamento de RAM por `!play` |
| **Dependências** | Depende da E3 (shutdown ordenado usa o group) |
| **Riscos** | **Baixo–Médio**. Semáforo pode rejeitar comandos legítimo sob carga — por isso a mensagem ao usuário e o limite ser generoso (4). `gosafe.Group` exige结构调整 mínima |
| **Testes** | router: N comandos simultâneos → no máximo 4 em execução; album: panic no batch handler **não** derruba o processo (teste de subprocesso ou `recover` verificado) |
| **Aceite** | 10 `!play` simultâneos → ≤4 handlers ativos, sem "too many open files"; panic no album → processo vivo |
| **Rollback** | `git revert` + `make release` |
| **Benefício** | Comportamento previsível sob rajada — importante com 512 MB e sessão única |
| **Complexidade** | **Média** (~1 dia) |

---

## Etapa 8 — Observabilidade e testes

| Campo | Conteúdo |
|---|---|
| **Objetivo** | Falha de produção diagnosticável em minutos, e a rede de segurança de testes para as etapas seguintes |
| **Problemas** | T-01, T-02, T-03, T-04, M-01, M-02, M-03, M-04, M-05, O-04, O-05, O-07, O-09, D-02 |
| **Arquivos** | `Makefile`, `infra/logger`, `commands/router.go` (log de args), `commands/nlu_detect.go` (truncamento por runes), `domain/scheduler/scheduler.go` (comentário), `domain/history` (fakes), `commands` (fakes), `infra/configs` (testes), `scripts/setup.sh` (versões fixas), `README.md` |
| **Mudanças** | 1) `make test` → `go test -race ./...`; 2) **corrigir a race** em `commands/album_test.go:225/231` (escrita sem mutex na goroutine do timer); 3) `zap` com sampling em `debug`; 4) `zap.Stack("stack")` no recover (`gosafe`) e `DisableStacktrace` só em produção; 5) remover log de args crus / PII (`router.go:315`, `nlu.go:108,133`) e truncar por runes (`nlu_detect.go:55`); 6) só logar "Comando executado" quando `err == nil` (U-10); 7) erros silenciados com `_ =` passam a log em warn (U-11); 8) `dupCache`: evicção por TTL (U-12); 9) hook com timeout próprio, cancelado só no fim (U-13); 10) fakes de teste: `whatsapp.FakeSender`, `history` com SQLite temporário, `configs` com arquivo temporário; 11) testes de tabela para `failure`, `DetectIntent` (JSON), `runMiddlewares`, `resolveAlias`, `isNLApplicable`, `isBareMention`; 12) `!stats` com contadores internos; 13) `setup.sh` com versões fixas; 14) README com o mapa de responsabilidades; 15) `gofmt -w` no arquivo pendente |
| **Dependências** | Último — usa os formatos estabilizados nas etapas anteriores |
| **Riscos** | **Baixo** — é aditivo. `-race` pode revelar data races pré-existentes → corrigir o que aparecer ou isolar com comentário explícito (não mascarar). O teste do album precisa ser reescrito com mutex (`album_test.go:170-201` já faz isso — serve de modelo) |
| **Testes** | Os próprios (a etapa é sobre testes) + `-race` verde |
| **Aceite** | `make test` roda com `-race`; cobertura de `internal/commands` ≥ 40% e de `domain/ia` ≥ 20%; `!stats` mostra contadores; nenhum log com PII do usuário |
| **Rollback** | `git revert` + `make release` |
| **Benefício** | Regressões passam a ser detectadas antes do deploy, e falha em produção é diagnosticável |
| **Complexidade** | **Média** (~2 dias) |

---

## Prioridade e justificativa

| Ordem | Etapa | Por que nesta ordem |
|---|---|---|
| 1º | **E1 — Não quebrar e não mentir** | Pânico e sucesso falso são falhas **visíveis ao usuário** e **baratas de corrigir**; risco de regressão baixo → melhor relação custo/benefício |
| 2º | **E2 — Identidade do owner e menção** | Crítico e de baixo risco: hoje o dono pode ficar sem acesso aos comandos privados, dependendo do formato do JID |
| 3º | **E3 — Ciclo de vida e reconexão** | O modo de falha mais caro do bot é ficar offline sem saber; a biblioteca já dá o que falta (`EnableAutoReconnect`, eventos) |
| 4º | **E4 — Configuração** | Multiplicador de bugs: todo comando errado de config vira bug de runtime; e é pré-requisito de E5 (cache) |
| 5º | **E5 — Persistência e integrações** | Confiabilidade das integrações e economia de chamadas; baixa dependência de outras etapas |
| 6º | **E6 — Taxonomia e UX** | Melhoria visível de UX, mas toca muitos arquivos → depois que as falhas reais estão classificadas |
| 7º | **E7 — Concorrência** | Específica até haver evidência de rajada; fica pronta para o crescimento |
| 8º | **E8 — Observabilidade e testes** | Rede de segurança para o futuro; pode rodar em qualquer momento, mas rende mais no fim |

**Custo total estimado:** ~9–12 dias de trabalho (sem contar revisão).
Cada etapa é publicável isoladamente — dá para parar quando quiser.

---

## Decisões aprovadas pelo dono (2026-11-10)

| # | Decisão | Escolha |
|---|---|---|
| D1 | `noteapi` (notebook) e seus 6 comandos | **manter** — mas seguindo a arquitetura do projeto: usar a taxonomia de falha da E6, sem camada nova e sem vazar erro interno |
| D2 | Binário versionado no git | **manter** — a CPU do plano não aguenta o build em container |
| D3 | `MaxDownloads` da música | **manter 1** por enquanto (sem single-flight nesta rodada) |
| D4 | `checkDeps` com `localEnabled: false` | **condicional** |
| D5 | Limite de handlers simultâneos | **5** |
| D6 | TTL do cache | **música: 24 h** (era 7 dias). Demais caches conforme a E5: geocoding/clima 10 min, cotação 30 min, feriado 6 h |
