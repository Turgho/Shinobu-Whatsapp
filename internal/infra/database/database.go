package database

import (
	"database/sql"
	"fmt"

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

	db, err := sql.Open(driver, d.Dsn)
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
