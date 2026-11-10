package configs

import (
	"os"
	"path/filepath"
	"testing"
)

// writeConfig grava um config.yaml temporário e posiciona o processo nele.
func writeConfig(t *testing.T, content string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
}

const minimalConfig = `
bot:
  name: Shinobu
  prefix: "!"
  timezone: "America/Sao_Paulo"
database:
  driver: sqlite3
  dsn: "file:storage/storage.db?_pragma=foreign_keys(1)"
usersJID:
  owner: "5511999999999@lid"
`

func TestLoadConfigMinima(t *testing.T) {
	writeConfig(t, minimalConfig)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("config mínima deveria carregar: %v", err)
	}
	if cfg.Bot.Prefix != "!" {
		t.Errorf("prefixo errado: %q", cfg.Bot.Prefix)
	}
	if cfg.Owner.JID != "5511999999999@lid" {
		t.Errorf("owner (JID) não carregou: %q", cfg.Owner.JID)
	}
}

// Sem owner o bot não sobe: falha no boot, não no primeiro comando.
func TestLoadConfigSemOwner(t *testing.T) {
	writeConfig(t, `
bot:
  prefix: "!"
  timezone: "America/Sao_Paulo"
database:
  dsn: "file:storage/storage.db"
`)

	if _, err := Load(); err == nil {
		t.Fatal("config sem owner deveria falhar")
	}
}

func TestLoadConfigSemPrefix(t *testing.T) {
	writeConfig(t, `
bot:
  timezone: "America/Sao_Paulo"
database:
  dsn: "file:storage/storage.db"
usersJID:
  owner: "5511999999999@lid"
`)

	if _, err := Load(); err == nil {
		t.Fatal("config sem prefixo deveria falhar")
	}
}

// O path canônico (YAML) e o legado (env) precisam funcionar.
func TestLoadConfigOwnerPorEnv(t *testing.T) {
	writeConfig(t, `
bot:
  prefix: "!"
  timezone: "America/Sao_Paulo"
database:
  dsn: "file:storage/storage.db"
`)

	t.Setenv("OWNER_NUMBER", "5511999999999")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("owner por env deveria carregar: %v", err)
	}
	if cfg.Owner.Number != "5511999999999" {
		t.Errorf("owner (número) não carregou: %q", cfg.Owner.Number)
	}
}

// O path legado "apicalls.groq.apiKey" continua valendo (compatibilidade).
func TestLoadConfigGroqPorPathLegado(t *testing.T) {
	writeConfig(t, minimalConfig)

	t.Setenv("GROQ_API_KEY", "gsk-legado")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("config mínima deveria carregar: %v", err)
	}
	if cfg.Groq.APIKey != "gsk-legado" {
		t.Errorf("Groq por env não carregou: %q", cfg.Groq.APIKey)
	}
}

// A chance de sticker configurada em 0 precisa ser respeitada (desligar).
func TestLoadConfigStickerChanceZero(t *testing.T) {
	writeConfig(t, `
bot:
  prefix: "!"
  timezone: "America/Sao_Paulo"
  normalStickerChance: 0
database:
  dsn: "file:storage/storage.db"
usersJID:
  owner: "5511999999999@lid"
`)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("config mínima deveria carregar: %v", err)
	}
	if cfg.Bot.NormalStickerChance != 0 {
		t.Errorf("chance de sticker deveria ficar 0, veio %v", cfg.Bot.NormalStickerChance)
	}
}

// Sem a chave, a chance assume o default (0.2).
func TestLoadConfigStickerChanceDefault(t *testing.T) {
	writeConfig(t, minimalConfig)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("config mínima deveria carregar: %v", err)
	}
	if cfg.Bot.NormalStickerChance != 0.2 {
		t.Errorf("default da chance de sticker deveria ser 0.2, veio %v", cfg.Bot.NormalStickerChance)
	}
}
