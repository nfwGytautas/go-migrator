package main

import (
	"context"
	"log"
	"time"

	"github.com/caarlos0/env/v9"
	gomigrator "github.com/nfwGytautas/go-migrator"
	"github.com/nfwGytautas/go-migrator/drivers"
)

type config struct {
	DatabaseDSN   string        `env:"DATABASE_DSN,required" envDefault:""`
	MigrationsDir string        `env:"MIGRATIONS_DIR,required" envDefault:""`
	Driver        string        `env:"DRIVER,required" envDefault:"postgres"`
	MaxRetries    int           `env:"MAX_RETRIES" envDefault:"5"`
	RetryDelay    time.Duration `env:"RETRY_DELAY" envDefault:"3s"`
	Timeout       time.Duration `env:"TIMEOUT" envDefault:"30s"`
}

func main() {
	cfg := config{}
	err := env.Parse(&cfg)
	if err != nil {
		log.Fatalf("Failed to load environment variables: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	migrations, err := gomigrator.LoadMigrationsFromDir(cfg.MigrationsDir)
	if err != nil {
		log.Fatal(err)
	}

	driver := getDriver(cfg)

	for range cfg.MaxRetries {
		err = gomigrator.RunMigrations(ctx, driver, migrations)
		if err != nil {
			log.Println(err)
			time.Sleep(cfg.RetryDelay)
			continue
		}
		break
	}
}

func getDriver(cfg config) gomigrator.MigrationDriver {
	switch cfg.Driver {
	case "postgres":
		return drivers.NewPostgresDriver(cfg.DatabaseDSN)
	default:
		log.Fatalf("Unsupported driver: %s", cfg.Driver)
		return nil
	}
}
