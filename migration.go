package gomigrator

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"strings"
)

// Migration is the struct that represents a migration
type Migration struct {
	Version      int    // the version of the migration
	Name         string // the name of the migration
	MigrationSQL string // the sql of the migration
	FixturesSQL  string // the sql of the fixtures
}

// LoadMigrationsFromDir loads the migrations from a filesystem directory, it doesn't recurse into subdirectories
func LoadMigrationsFromDir(path string, fixtures bool) ([]Migration, error) {
	dirFS := os.DirFS(path)
	return LoadMigrationsFromFS(dirFS, fixtures)
}

// LoadMigrationsFromFS loads the migrations from an embedded filesystem
//   - DOES NOT recurse into subdirectories
func LoadMigrationsFromFS(afs fs.FS, fixtures bool) ([]Migration, error) {
	files, err := fs.ReadDir(afs, ".")
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations from embedded filesystem: %w", err)
	}

	if len(files) == 0 {
		return nil, errors.New("no migrations found")
	}

	migrations := make([]Migration, 0, len(files))

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		m, err := parseMigrationFile(afs, file, fixtures)
		if err != nil {
			return nil, err
		}

		if m != nil {
			migrations = append(migrations, *m)
		}
	}

	return migrations, nil
}

func parseMigrationFile(afs fs.FS, file fs.DirEntry, fixtures bool) (m *Migration, err error) {
	// File name format: <id>_<name>.sql
	fileName := file.Name()

	// Ignore non sql files
	if !strings.HasSuffix(fileName, ".sql") {
		return
	}

	// Ignore fixture files
	if strings.HasSuffix(fileName, ".fixture.sql") {
		return
	}

	fileName = strings.TrimSuffix(fileName, ".sql")

	m = &Migration{}

	// Find the first underscore to separate the ID from the name
	underscoreIndex := strings.Index(fileName, "_")
	if underscoreIndex == -1 {
		return m, fmt.Errorf("invalid migration file name format for file: %s, (the format is <id>_<name>.sql)", fileName)
	}

	versionStr := fileName[:underscoreIndex]
	m.Name = fileName[underscoreIndex+1:]

	m.Version, err = strconv.Atoi(versionStr)
	if err != nil {
		return m, fmt.Errorf("failed to parse migration file version (%s): %w", fileName, err)
	}

	if m.Version <= 0 {
		return m, fmt.Errorf("migration file version cannot be <= 0: %s", fileName)
	}

	migrationSQL, err := fs.ReadFile(afs, fileName+".sql")
	if err != nil {
		return m, fmt.Errorf("failed to read migration file (%s): %w", fileName, err)
	}
	m.MigrationSQL = string(migrationSQL)

	if fixtures {
		// Check if the fixture file exists and read it if it does
		if _, err := fs.Stat(afs, fileName+".fixture.sql"); err == nil {
			fixturesSQL, err := fs.ReadFile(afs, fileName+".fixture.sql")
			if err != nil {
				return m, fmt.Errorf("failed to read fixture file (%s): %w", fileName, err)
			}
			m.FixturesSQL = string(fixturesSQL)
		}
	}

	return m, nil
}
