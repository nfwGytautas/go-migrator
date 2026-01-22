package executor

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	gomigrator "github.com/nfwGytautas/go-migrator"
)

type stdoutLogger struct {
	name string
}

func (l *stdoutLogger) Info(message string) {
	log.Printf("[%s] %s\n", l.name, message)
}

func (l *stdoutLogger) Error(err error) {
	log.Printf("[%s] %v\n", l.name, err)
}

func Execute(ctx context.Context, cfg *Config) bool {
	wg := sync.WaitGroup{}
	wg.Add(len(cfg.Migrations))

	for _, migration := range cfg.Migrations {
		go func() {
			executeMigration(ctx, cfg, migration)
			wg.Done()
		}()
	}

	wg.Wait()

	return true
}

func executeMigration(ctx context.Context, cfg *Config, migrationCfg migrationConfig) {
	logger := &stdoutLogger{name: migrationCfg.Name}

	driver := migrationCfg.getDriver()
	if driver == nil {
		logger.Error(fmt.Errorf("failed to resolve driver"))
		return
	}

	migrations, err := gomigrator.LoadMigrationsFromDir(migrationCfg.Source, cfg.Fixtures)
	if err != nil {
		logger.Error(fmt.Errorf("failed to load migrations: %w", err))
		return
	}

	ctx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	for i := 0; i < cfg.MaxRetries; i++ {
		logger.name = fmt.Sprintf("%s (attempt %d)", migrationCfg.Name, i+1)

		err := gomigrator.RunMigrations(ctx, driver, migrations, logger)
		if err == nil {
			break
		}

		time.Sleep(cfg.RetryDelay)
	}
}
