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
	Version int    // the version of the migration
	Name    string // the name of the migration
	SQL     string // the sql of the migration
}

// LoadMigrationsFromDir loads the migrations from a filesystem directory, it doesn't recurse into subdirectories
func LoadMigrationsFromDir(path string) ([]Migration, error) {
	dirFS := os.DirFS(path)
	return LoadMigrationsFromFS(dirFS)
}

// LoadMigrationsFromFS loads the migrations from an embedded filesystem
//   - DOES NOT recurse into subdirectories
func LoadMigrationsFromFS(afs fs.FS) ([]Migration, error) {
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

		content, err := fs.ReadFile(afs, file.Name())
		if err != nil {
			return nil, fmt.Errorf("failed to read migration file (%s): %w", file.Name(), err)
		}

		version, name, err := parseMigrationFileName(file.Name())
		if err != nil {
			return nil, err
		}

		migrations = append(migrations, Migration{
			Version: version,
			Name:    name,
			SQL:     string(content),
		})
	}

	return migrations, nil
}

func parseMigrationFileName(fileName string) (version int, name string, err error) {
	// File name format: <id>_<name>.sql

	// Find the first underscore to separate the ID from the name
	underscoreIndex := strings.Index(fileName, "_")
	if underscoreIndex == -1 {
		return 0, "", fmt.Errorf("invalid migration file name format for file: %s, (the format is <id>_<name>.sql)", fileName)
	}

	versionStr := fileName[:underscoreIndex]
	name = fileName[underscoreIndex+1:]

	version, err = strconv.Atoi(versionStr)
	if err != nil {
		return 0, "", fmt.Errorf("failed to parse migration file version (%s): %w", fileName, err)
	}

	if version <= 0 {
		return 0, "", fmt.Errorf("migration file version cannot be <= 0: %s", fileName)
	}

	return version, name, nil
}
