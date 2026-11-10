package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	// Driver SQLite puro (sem cgo) — remove o mattn/go-sqlite3 duplicado:
	// o history store já usa modernc.org/sqlite; duas stack do mesmo banco
	// encareciam o binário e o build.
	"go.uber.org/zap"
	_ "modernc.org/sqlite"
)

type Database struct {
	Driver string
	Dsn    string
	Log    *zap.Logger
}

func NewDatabase(driver, dsn string, log *zap.Logger) *Database {
	return &Database{
		Driver: driver,
		Dsn:    dsn,
		Log:    log,
	}
}

// Connect abre a conexão principal da aplicação.
// Nome "sqlite3" no config.yaml é histórico (mattn) — normalizado
// para o nome de registro do driver modernc, "sqlite".
func (d *Database) Connect() (*sql.DB, error) {
	driver := d.Driver
	if driver == "sqlite3" || driver == "sqlite" {
		driver = "sqlite"
	}
	d.Log.Info("Conectando ao banco de dados",
		zap.String("driver", d.Driver),
	)

	// Garante o diretório do banco (deploy limpo não tem storage/).
	if dir := dbDirFromDSN(d.Dsn); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("criar diretório do banco %s: %w", dir, err)
		}
	}

	db, err := sql.Open(driver, d.Dsn)
	if err != nil {
		d.Log.Error("Falha ao abrir conexão com o banco",
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Limita o número máximo de conexões abertas para 1: o SQLite serializa
	// escritas de qualquer forma, e um pool maior só aumenta o risco de
	// lock (o cliente do whatsmeow também usa esta conexão).
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		d.Log.Error("Falha ao testar conexão com o banco",
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	d.Log.Info("Banco de dados conectado com sucesso")

	return db, nil
}

// dbDirFromDSN extrai o diretório do DSN do SQLite ("file:storage/x.db?...").
func dbDirFromDSN(dsn string) string {
	path := dsn
	if i := strings.Index(path, "?"); i >= 0 {
		path = path[:i]
	}
	path = strings.TrimPrefix(path, "file:")
	if path == "" {
		return ""
	}
	return filepath.Dir(path)
}

// Open abre um banco SQLite com a política padrão do projeto:
// foreign_keys ligado, busy_timeout para não falhar logo em lock e uma
// única conexão (o SQLite serializa escritas).
//
// É o ponto único de abertura — o storage do whatsmeow e o histórico do
// app usam a mesma política (antes o histórico abria sem pragma nenhuma,
// com pool ilimitado: risco de SQLITE_BUSY).
func Open(path string, log *zap.Logger) (*sql.DB, error) {
	if log == nil {
		log = zap.NewNop()
	}

	// Garante o diretório do banco (deploy limpo não tem storage/).
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("criar diretório do banco %s: %w", dir, err)
		}
	}

	dsn := path
	if !strings.Contains(path, "?") {
		dsn = path + "?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("abrir banco %s: %w", path, err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("acessar banco %s: %w", path, err)
	}

	log.Info("Banco aberto",
		zap.String("path", path),
		zap.Bool("foreign_keys", true),
	)
	return db, nil
}
