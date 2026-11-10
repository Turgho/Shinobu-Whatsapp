# Shinobu — build em PC local (Arch), binário versionado no git.
# O plano da SquareCloud não compila em-container: `make release` faz
# testes → build otimizado → commit (com o binário) → tag → push, num comando só.
#
# Uso:
#   make release                     # bump de patch v2.22.0 → v2.22.1, msg padrão
#   make release BUMP=minor          # v2.22.0 → v2.23.0  (use minor: feature nova)
#   make release BUMP=major          # v2.22.0 → v3.0.0   (use major: breaking change)
#   make release VER=v2.23.0 MSG="feat: música local"   # versão exata, ignora BUMP
#   make release DRY=1               # só mostra o que faria
#   make build | test | run | setup  # utilitários individuais

MODULE  := $(shell go list -m)
BRANCH  := $(shell git rev-parse --abbrev-ref HEAD)
# BUMP define a parte do semver a incrementar (patch por padrão).
# VER= sempre vence — use BUMP só quando quiser inteligência automática.
BUMP    := $(or $(BUMP),patch)
LAST_TAG := $(shell git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//')
TAG     := $(if $(VER),$(VER),$(shell echo $(LAST_TAG) | awk -F. -vb="$(BUMP)" \
	'{ if (b=="minor") printf "v%d.%d.0", $$1, $$2+1; \
	  else if (b=="major") printf "v%d.0.0", $$1; \
	  else if (b=="patch") printf "v%d.%d.%d", $$1, $$2, $$3+1; \
	  else { print "make: BUMP invalido — use patch, minor ou major" > "/dev/stderr"; exit 1 } }'))
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
MSG     := $(if $(MSG),release: $(MSG),release: $(TAG))

# CGO_ENABLED=0: SQLite puro (modernc), sem gcc — binário estático e menor.
# -trimpath -s -w: sem paths locais e sem tabelas de debug → ~37MB → ~25MB.
# UPX descartado (RAM 512MB do contêiner não sobra pra descomprimir).
GOFLAGS := CGO_ENABLED=0 GOOS=linux GOARCH=amd64
LDFLAGS := -ldflags "-s -w -X '$(MODULE)/internal/infra/version.Version=$(TAG)' \
  -X '$(MODULE)/internal/infra/version.Commit=$(COMMIT)' \
  -X '$(MODULE)/internal/infra/version.Date=$(DATE)'"

build:
	$(GOFLAGS) go build -trimpath $(LDFLAGS) -o Shinobu ./cmd/bot/main.go

run:
	go run $(LDFLAGS) ./cmd/bot/main.go

# -race: as condições de corrida do projeto (scheduler, album, hooks) só
# aparecem com o detector ligado.
test:
	go test -race ./... -count=1

setup:
	@bash scripts/setup.sh

define RELEASE_CMDS
	# build otimizado com a versão nova cravada no binário
	$(GOFLAGS) go build -trimpath $(LDFLAGS) -o Shinobu ./cmd/bot/main.go

	# commit inclui TODO o trabalho pendente + binário (se houver algo pra commitar)
	git add -A
	git diff --cached --quiet || git commit -m "$(MSG)"

	# tag semver + push do branch e da tag juntos
	git tag -a $(TAG) -m "$(MSG)"
	git push origin $(BRANCH)
	git push origin $(TAG)
endef

.PHONY: release show-version cli
release:
	@test -n "$(TAG)" && echo "$(TAG)" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+$$' \
		|| (echo "make: BUMP/VER invalido — use BUMP=patch|minor|major ou VER=vX.Y.Z"; exit 1)
	@echo "→ branch: $(BRANCH) | tag: $(TAG) | msg: \"$(MSG)\""
	@echo "→ testes primeiro..."
	@go test -race ./... -count=1
ifeq ($(DRY),1)
	@echo "→ DRY RUN — nada será commitado ou enviado. Fluxo que rodaria:"
	@echo "  1. build     CGO_ENABLED=0 go build -trimpath (versão=$(TAG))"
	@echo "  2. commit    git add -A + msg: \"$(MSG)\""
	@echo "  3. tag       git tag -a $(TAG)"
	@echo "  4. push      git push origin $(BRANCH) && git push origin $(TAG)"
else
	$(RELEASE_CMDS)
	@echo "✅ $(TAG) publicado ($(BRANCH)) — binário: $$(du -h Shinobu | cut -f1)"
endif

# Painel de controle no terminal: go run ./cmd/shinobu
cli:
	go build -o bin/shinobu ./cmd/shinobu/

show-version:
	@echo "tag=$(TAG) commit=$(COMMIT)"
