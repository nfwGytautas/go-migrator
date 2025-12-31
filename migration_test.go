package gomigrator

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadMigrationsFromDir(t *testing.T) {
	t.Run("successful load from directory", func(t *testing.T) {
		// Create a temporary directory
		tmpDir := t.TempDir()

		// Save current working directory
		oldWd, err := os.Getwd()
		require.NoError(t, err, "failed to get working directory")
		defer os.Chdir(oldWd)

		// Change to the temp directory because LoadMigrationsFromDir uses file.Name()
		// which is just the filename, not the full path
		err = os.Chdir(tmpDir)
		require.NoError(t, err, "failed to change directory")

		// Create test migration files
		// Format: <id>_<name>.sql - the name part can contain multiple underscores
		migrationFiles := map[string]string{
			"1_create_users.sql":       "CREATE TABLE users (id INT PRIMARY KEY);",
			"2_create_posts.sql":       "CREATE TABLE posts (id INT PRIMARY KEY);",
			"3_add_index_to_users.sql": "CREATE INDEX idx_users_id ON users(id);",
		}

		for fileName, content := range migrationFiles {
			err := os.WriteFile(fileName, []byte(content), 0644)
			require.NoError(t, err, "failed to create test file")
		}

		migrations, err := LoadMigrationsFromDir(tmpDir)
		require.NoError(t, err)
		require.Len(t, migrations, 3, "expected 3 migrations")

		// Verify migrations are loaded (order may vary due to filesystem)
		versionMap := make(map[int]Migration)
		for _, m := range migrations {
			versionMap[m.Version] = m
		}

		m, ok := versionMap[1]
		require.True(t, ok, "expected migration 1 to exist")
		assert.Equal(t, "create_users.sql", m.Name, "expected migration 1 with name 'create_users.sql'")

		m, ok = versionMap[2]
		require.True(t, ok, "expected migration 2 to exist")
		assert.Equal(t, "create_posts.sql", m.Name, "expected migration 2 with name 'create_posts.sql'")

		m, ok = versionMap[3]
		require.True(t, ok, "expected migration 3 to exist")
		assert.Equal(t, "add_index_to_users.sql", m.Name, "expected migration 3 with name 'add_index_to_users.sql'")
	})

	t.Run("handles non-existent directory", func(t *testing.T) {
		_, err := LoadMigrationsFromDir("/non/existent/directory")
		require.Error(t, err, "expected error for non-existent directory")
	})

	t.Run("handles empty directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		_, err := LoadMigrationsFromDir(tmpDir)
		require.Error(t, err, "expected error for empty directory")
		assert.Equal(t, "no migrations found", err.Error())
	})

	t.Run("skips subdirectories", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Save current working directory
		oldWd, err := os.Getwd()
		require.NoError(t, err, "failed to get working directory")
		defer os.Chdir(oldWd)

		// Change to the temp directory
		err = os.Chdir(tmpDir)
		require.NoError(t, err, "failed to change directory")

		// Create a subdirectory
		subDir := filepath.Join(tmpDir, "subdir")
		err = os.Mkdir(subDir, 0755)
		require.NoError(t, err, "failed to create subdirectory")

		// Create a migration file in the root
		err = os.WriteFile("1_create_users.sql", []byte("CREATE TABLE users (id INT);"), 0644)
		require.NoError(t, err, "failed to create test file")

		migrations, err := LoadMigrationsFromDir(tmpDir)
		require.NoError(t, err)
		assert.Len(t, migrations, 1, "expected 1 migration (subdirectory should be skipped)")
	})

	t.Run("handles invalid file name format", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Save current working directory
		oldWd, err := os.Getwd()
		require.NoError(t, err, "failed to get working directory")
		defer os.Chdir(oldWd)

		// Change to the temp directory
		err = os.Chdir(tmpDir)
		require.NoError(t, err, "failed to change directory")

		// Create a file with invalid format
		err = os.WriteFile("invalid.sql", []byte("CREATE TABLE users (id INT);"), 0644)
		require.NoError(t, err, "failed to create test file")

		_, err = LoadMigrationsFromDir(tmpDir)
		require.Error(t, err, "expected error for invalid file name format")
	})

	t.Run("handles invalid version number", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Save current working directory
		oldWd, err := os.Getwd()
		require.NoError(t, err, "failed to get working directory")
		defer os.Chdir(oldWd)

		// Change to the temp directory
		err = os.Chdir(tmpDir)
		require.NoError(t, err, "failed to change directory")

		// Create a file with non-numeric version
		err = os.WriteFile("abc_create_users.sql", []byte("CREATE TABLE users (id INT);"), 0644)
		require.NoError(t, err, "failed to create test file")

		_, err = LoadMigrationsFromDir(tmpDir)
		require.Error(t, err, "expected error for invalid version number")
	})

	t.Run("handles zero or negative version", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Save current working directory
		oldWd, err := os.Getwd()
		require.NoError(t, err, "failed to get working directory")
		defer os.Chdir(oldWd)

		// Change to the temp directory
		err = os.Chdir(tmpDir)
		require.NoError(t, err, "failed to change directory")

		testCases := []string{"0_create_table.sql", "-1_create_table.sql"}

		for _, fileName := range testCases {
			err := os.WriteFile(fileName, []byte("CREATE TABLE users (id INT);"), 0644)
			require.NoError(t, err, "failed to create test file %s", fileName)

			_, err = LoadMigrationsFromDir(tmpDir)
			require.Error(t, err, "expected error for version <= 0 in file %s", fileName)

			// Clean up for next iteration
			os.Remove(fileName)
		}
	})
}

func TestLoadMigrationsFromFS(t *testing.T) {
	t.Run("handles empty embedded filesystem", func(t *testing.T) {
		// Create an empty filesystem using fstest.MapFS
		emptyFS := fstest.MapFS{}

		_, err := LoadMigrationsFromFS(emptyFS)
		require.Error(t, err, "expected error for empty filesystem")
	})

	t.Run("successful load from embedded filesystem", func(t *testing.T) {
		// Create an in-memory filesystem using fstest.MapFS
		testFS := fstest.MapFS{
			"1_create_users.sql": &fstest.MapFile{
				Data: []byte(
					`CREATE TABLE users (
						id INT PRIMARY KEY,
						name VARCHAR(255) NOT NULL,
						email VARCHAR(255) UNIQUE NOT NULL,
						created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
					);`,
				),
			},
			"2_create_posts.sql": &fstest.MapFile{
				Data: []byte(
					`CREATE TABLE posts (
						id INT PRIMARY KEY,
						user_id INT NOT NULL,
						title VARCHAR(255) NOT NULL,
						content TEXT,
						created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
						FOREIGN KEY (user_id) REFERENCES users(id)
					);`,
				),
			},
			"3_add_index_to_users.sql": &fstest.MapFile{
				Data: []byte(
					`
					CREATE INDEX idx_users_email ON users(email);
					CREATE INDEX idx_users_created_at ON users(created_at);
					`,
				),
			},
		}

		migrations, err := LoadMigrationsFromFS(testFS)
		require.NoError(t, err, "expected no error when loading from embedded filesystem")
		require.Len(t, migrations, 3, "expected 3 migrations from test filesystem")

		// Verify migrations are loaded correctly (order may vary due to filesystem)
		versionMap := make(map[int]Migration)
		for _, m := range migrations {
			versionMap[m.Version] = m
		}

		// Verify migration 1
		m, ok := versionMap[1]
		require.True(t, ok, "expected migration 1 to exist")
		assert.Equal(t, "create_users.sql", m.Name, "expected migration 1 with name 'create_users.sql'")
		assert.Contains(t, m.SQL, "CREATE TABLE users", "expected migration 1 to contain CREATE TABLE users")

		// Verify migration 2
		m, ok = versionMap[2]
		require.True(t, ok, "expected migration 2 to exist")
		assert.Equal(t, "create_posts.sql", m.Name, "expected migration 2 with name 'create_posts.sql'")
		assert.Contains(t, m.SQL, "CREATE TABLE posts", "expected migration 2 to contain CREATE TABLE posts")

		// Verify migration 3
		m, ok = versionMap[3]
		require.True(t, ok, "expected migration 3 to exist")
		assert.Equal(t, "add_index_to_users.sql", m.Name, "expected migration 3 with name 'add_index_to_users.sql'")
		assert.Contains(t, m.SQL, "CREATE INDEX", "expected migration 3 to contain CREATE INDEX")
	})
}

func TestParseMigrationFileName(t *testing.T) {
	t.Run("valid file namesingle underscore", func(t *testing.T) {
		version, name, err := parseMigrationFileName("1_createusers.sql")
		require.NoError(t, err)
		assert.Equal(t, 1, version)
		assert.Equal(t, "createusers.sql", name)
	})

	t.Run("valid file name multiple underscores", func(t *testing.T) {
		// Multiple underscores in the name part should be allowed
		version, name, err := parseMigrationFileName("5_create_users_table.sql")
		require.NoError(t, err)
		assert.Equal(t, 5, version)
		assert.Equal(t, "create_users_table.sql", name)
	})

	t.Run("valid file name many underscores", func(t *testing.T) {
		// Many underscores should work as long as ID is first
		version, name, err := parseMigrationFileName("10_add_index_to_users_table.sql")
		require.NoError(t, err)
		assert.Equal(t, 10, version)
		assert.Equal(t, "add_index_to_users_table.sql", name)
	})

	t.Run("invalid format no underscore", func(t *testing.T) {
		_, _, err := parseMigrationFileName("1create_users.sql")
		require.Error(t, err, "expected error for invalid format (no underscore)")
	})

	t.Run("invalid version non-numeric", func(t *testing.T) {
		_, _, err := parseMigrationFileName("abc_create_table.sql")
		require.Error(t, err, "expected error for non-numeric version")
	})

	t.Run("invalid version zero", func(t *testing.T) {
		_, _, err := parseMigrationFileName("0_create_table.sql")
		require.Error(t, err, "expected error for zero version")
	})

	t.Run("invalid version negative", func(t *testing.T) {
		_, _, err := parseMigrationFileName("-1_create_table.sql")
		require.Error(t, err, "expected error for negative version")
	})
}
