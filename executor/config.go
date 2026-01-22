package executor

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	gomigrator "github.com/nfwGytautas/go-migrator"
	"github.com/nfwGytautas/go-migrator/drivers"
	"github.com/stretchr/testify/assert/yaml"
)

type Config struct {
	MaxRetries int               `yaml:"max-retries"`
	RetryDelay time.Duration     `yaml:"retry-delay"`
	Timeout    time.Duration     `yaml:"timeout"`
	Fixtures   bool              `yaml:"fixtures"`
	Migrations []migrationConfig `yaml:"migrations"`
}

type migrationConfig struct {
	Name     string          `yaml:"name"`
	Source   string          `yaml:"source"`
	Postgres *postgresConfig `yaml:"postgres"`
}

type postgresConfig struct {
	DSN string `yaml:"dsn"`
}

// LoadConfig loads the config from the specified path or `gomigrator.yaml` if no path is provided
func LoadConfig(path string) (*Config, error) {
	if path == "" {
		path = "gomigrator.yaml"
	}

	// Check if the file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", path)
	}

	cfg := &Config{}

	// Read config file to string
	contentRaw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Substitute environment variables in the config file
	contentSubstituted := os.ExpandEnv(string(contentRaw))

	// Parse the config file
	err = yaml.Unmarshal([]byte(contentSubstituted), cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	faults := cfg.validate()
	if len(faults) > 0 {
		log.Println("Config is invalid:")
		log.Println(strings.Join(faults, "\n"))
		return nil, fmt.Errorf("config is invalid")

	}

	return cfg, nil
}

func (c *Config) validate() []string {
	faults := []string{}
	if c.Timeout.Seconds() <= 0 {
		faults = append(faults, "timeout must be greater than 0")
	}

	if len(c.Migrations) == 0 {
		faults = append(faults, "no migrations specified")
	}

	for i, migration := range c.Migrations {
		if migration.Name == "" {
			migration.Name = fmt.Sprintf("migration-%d", i)
		}

		faults = append(faults, migration.validate()...)
	}

	return faults
}

func (m *migrationConfig) validate() []string {
	faults := []string{}
	if m.Source == "" {
		faults = append(faults, fmt.Sprintf("[%s] source must be specified", m.Name))
	}

	specifiedDrivers := 0

	if m.Postgres != nil {
		specifiedDrivers++
		faults = append(faults, m.Postgres.validate()...)
	}

	if specifiedDrivers == 0 {
		faults = append(faults, fmt.Sprintf("[%s] a driver must be specified", m.Name))
	}

	if specifiedDrivers > 1 {
		faults = append(faults, fmt.Sprintf("[%s] only one driver can be specified", m.Name))
	}

	return faults
}

func (m *migrationConfig) getDriver() gomigrator.MigrationDriver {
	if m.Postgres != nil {
		return drivers.NewPostgresDriver(m.Postgres.DSN)
	}

	return nil
}

func (p *postgresConfig) validate() []string {
	faults := []string{}
	if p.DSN == "" {
		faults = append(faults, "dsn must be specified")
	}

	return faults
}
