package database

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3" // Importa o driver do SQLite3
	"go.uber.org/zap"
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

func (d *Database) Connect() (*sql.DB, error) {
	d.Log.Info("Conectando ao banco de dados",
		zap.String("driver", d.Driver),
	)

	db, err := sql.Open(d.Driver, d.Dsn)
	if err != nil {
		d.Log.Error("Falha ao abrir conexão com o banco",
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Limita o número máximo de conexões abertas para 1, já que o WhatsApp Client é single-threaded
	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		d.Log.Error("Falha ao testar conexão com o banco",
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	d.Log.Info("Banco de dados conectado com sucesso")

	return db, nil
}
