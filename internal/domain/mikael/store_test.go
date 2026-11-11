package mikael

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"go.uber.org/zap"
	_ "modernc.org/sqlite"
)

// O ajuste do contador (usado pelo !mikael somar/ajustar) precisa
// somar/definir o valor e persistir.
func TestAddCountESetCount(t *testing.T) {
	// Store com um DB temporário: usa o schema real via helper.
	s := newTestStore(t)

	ctx := context.Background()

	total, err := s.AddCount(ctx, "chat@g.us", "pix", 898)
	if err != nil {
		t.Fatalf("AddCount falhou: %v", err)
	}
	if total != 898 {
		t.Errorf("esperava 898, veio %d", total)
	}

	total, err = s.AddCount(ctx, "chat@g.us", "pix", 11)
	if err != nil {
		t.Fatalf("AddCount falhou: %v", err)
	}
	if total != 909 {
		t.Errorf("esperava 909, veio %d", total)
	}

	total, err = s.SetCount(ctx, "chat@g.us", "pix", 42)
	if err != nil {
		t.Fatalf("SetCount falhou: %v", err)
	}
	if total != 42 {
		t.Errorf("esperava 42, veio %d", total)
	}

	// valor absoluto não pode ficar negativo
	total, err = s.SetCount(ctx, "chat@g.us", "pix", -5)
	if err != nil {
		t.Fatalf("SetCount falhou: %v", err)
	}
	if total != 0 {
		t.Errorf("esperava 0, veio %d", total)
	}
}

// newTestStore cria um Store com um SQLite temporário.
func newTestStore(t *testing.T) *Store {
	t.Helper()

	db := openTestDB(t)
	s := &Store{
		db:           db,
		log:          testLogger(),
		groups:       map[string]bool{"chat@g.us": true},
		targetSender: "46549275017279@lid",
	}
	if err := s.initSchema(); err != nil {
		t.Fatalf("initSchema falhou: %v", err)
	}
	return s
}

// testLogger devolve um logger silencioso.
func testLogger() *zap.Logger { return zap.NewNop() }

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "mikael.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("sql.Open falhou: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}
