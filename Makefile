MODULE  := $(shell go list -m)
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -ldflags "\
  -X '$(MODULE)/internal/infra/version.Version=$(VERSION)' \
  -X '$(MODULE)/internal/infra/version.Commit=$(COMMIT)' \
  -X '$(MODULE)/internal/infra/version.Date=$(DATE)'"

build:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o Shinobu ./cmd/bot/main.go

run:
	go run $(LDFLAGS) ./cmd/bot/main.go

test:
	go test ./... -count=1

setup:
	@bash scripts/setup.sh