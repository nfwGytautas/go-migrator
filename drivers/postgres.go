package drivers

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	gomigrator "github.com/nfwGytautas/go-migrator"
)

// postgresDriver is a driver for the migrator using the jackc/pgx library
type postgresDriver struct {
	dsn string
	db  *pgx.Conn
}

// NewPostgresDriver creates a new postgres driver
func NewPostgresDriver(connString string) *postgresDriver {
	return &postgresDriver{dsn: connString}
}

func (d *postgresDriver) Connect(ctx context.Context) (err error) {
	d.db, err = pgx.Connect(ctx, d.dsn)
	return
}

func (d *postgresDriver) CreateMigrationsTable(ctx context.Context) (err error) {
	const tableSchema = `
	CREATE TABLE IF NOT EXISTS {{.Table}} (
		id 			INT PRIMARY KEY,
		name 		VARCHAR(255) NOT NULL,
		applied_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	)
	`
	_, err = d.db.Exec(ctx, tableSchema, migrationsTable)
	return
}

func (d *postgresDriver) GetCurrentVersion(ctx context.Context) (version int, err error) {
	const query = `
	SELECT MAX(id) FROM {{.Table}}
	`
	err = d.db.QueryRow(ctx, query, migrationsTable).Scan(&version)
	if err == pgx.ErrNoRows {
		return 0, nil
	}
	return
}

func (d *postgresDriver) ApplyMigration(ctx context.Context, migration gomigrator.Migration) error {
	const query = `
	INSERT INTO {{.Table}} (id, name, applied_at) VALUES ($1, $2, $3)
	`

	// Start a transaction
	tx, err := d.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Apply the migration
	_, err = tx.Exec(ctx, migration.SQL)
	if err != nil {
		return fmt.Errorf("failed to apply migration (%s): %w", migration.Name, err)
	}

	// Log the migration
	_, err = d.db.Exec(
		ctx,
		query,
		migration.Version,
		migration.Name,
		time.Now().Unix(),
	)

	// Commit the transaction
	err = tx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
