package gomigrator

import (
	"context"
	"fmt"
	"log"
	"sort"
)

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
func RunMigrations(ctx context.Context, driver MigrationDriver, migrations []Migration) error {
	// Sort the migrations by their version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	// Try to connect to the database and ensure the migrations table exists
	if err := driver.Connect(ctx); err != nil {
		return err
	}
	defer driver.Close(ctx)

	if err := driver.CreateMigrationsTable(ctx); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	currentVersion, err := driver.GetCurrentVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	log.Println("Current version: ", currentVersion)
	log.Println("Latest  version: ", len(migrations))

	// Sequentially apply the migrations
	for i := currentVersion; i < len(migrations); i++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		log.Printf("Applying migration: '%s' (version: %d)\n", migrations[i].Name, migrations[i].Version)

		// Apply the migration at index `i`
		// This is valid because we sorted the migrations
		// Migration ID `0` is reserved
		// So element 0 is the migration with ID `1`
		// Element 1 is the migration with ID `2`
		if err := driver.ApplyMigration(ctx, migrations[i]); err != nil {
			return err
		}
	}

	return nil
}
