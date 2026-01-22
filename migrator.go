package gomigrator

import (
	"context"
	"fmt"
	"sort"
)

// Logger is the interface that must be implemented by a logger
type Logger interface {
	// Info logs a debug/informational message
	Info(string)

	// Error logs an error message
	Error(error)
}

// MigrationDriver is the interface that must be implemented by a specific database dialect
type MigrationDriver interface {
	// Connect to the database
	Connect(ctx context.Context) error

	// Create the migrations table
	CreateMigrationsTable(ctx context.Context) error

	// Get the current version of the migrations
	GetCurrentVersion(ctx context.Context) (int, error)

	// Apply a migration
	ApplyMigration(ctx context.Context, migration Migration) error

	// Close the connection to the database
	Close(ctx context.Context) error
}

// RunMigrations runs the migrations on a given driver
func RunMigrations(ctx context.Context, driver MigrationDriver, migrations []Migration, logger Logger) error {
	// Sort the migrations by their version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	// Try to connect to the database and ensure the migrations table exists
	if err := driver.Connect(ctx); err != nil {
		if logger != nil {
			logger.Error(fmt.Errorf("failed to connect to the database: %w", err))
		}
		return fmt.Errorf("failed to connect to the database: %w", err)
	}
	defer driver.Close(ctx)

	if err := driver.CreateMigrationsTable(ctx); err != nil {
		if logger != nil {
			logger.Error(fmt.Errorf("failed to create migrations table: %w", err))
		}
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	currentVersion, err := driver.GetCurrentVersion(ctx)
	if err != nil {
		if logger != nil {
			logger.Error(fmt.Errorf("failed to get current version: %w", err))
		}
		return fmt.Errorf("failed to get current version: %w", err)
	}

	if logger != nil {
		logger.Info(fmt.Sprintf("current version: %d", currentVersion))
	}

	// Sequentially apply the migrations
	appliedMigrations := 0
	for i := currentVersion; i < len(migrations); i++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if logger != nil {
			logger.Info(fmt.Sprintf("applying migration %d: %s", migrations[i].Version, migrations[i].Name))
		}

		// Apply the migration at index `i`
		// This is valid because we sorted the migrations
		// Migration ID `0` is reserved
		// So element 0 is the migration with ID `1`
		// Element 1 is the migration with ID `2`
		if err := driver.ApplyMigration(ctx, migrations[i]); err != nil {
			if logger != nil {
				logger.Error(fmt.Errorf("failed to apply migration %d: %w", migrations[i].Version, err))
			}
			return fmt.Errorf("failed to apply migration %d: %w", migrations[i].Version, err)
		}

		appliedMigrations++
	}

	if logger != nil {
		logger.Info(fmt.Sprintf("applied %d migrations", appliedMigrations))
	}

	return nil
}
