package gomigrator

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockDriver is a mock implementation of MigrationDriver using testify/mock
type mockDriver struct {
	mock.Mock
	appliedMigrations []Migration
}

func (m *mockDriver) Connect(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockDriver) CreateMigrationsTable(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockDriver) GetCurrentVersion(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func (m *mockDriver) ApplyMigration(ctx context.Context, migration Migration) error {
	m.appliedMigrations = append(m.appliedMigrations, migration)
	args := m.Called(ctx, migration)
	return args.Error(0)
}

func (m *mockDriver) Close(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestRunMigrations(t *testing.T) {
	t.Run("successful migration run", func(t *testing.T) {
		driver := new(mockDriver)
		driver.On("Connect", mock.Anything).Return(nil)
		driver.On("CreateMigrationsTable", mock.Anything).Return(nil)
		driver.On("GetCurrentVersion", mock.Anything).Return(0, nil)
		driver.On("ApplyMigration", mock.Anything, mock.Anything).Return(nil).Times(3)
		driver.On("Close", mock.Anything).Return(nil)

		migrations := []Migration{
			{Version: 1, Name: "create_users", SQL: "CREATE TABLE users (id INT);"},
			{Version: 2, Name: "create_posts", SQL: "CREATE TABLE posts (id INT);"},
			{Version: 3, Name: "add_index", SQL: "CREATE INDEX idx_users_id ON users(id);"},
		}

		ctx := context.Background()
		err := RunMigrations(ctx, driver, migrations)

		require.NoError(t, err)
		driver.AssertExpectations(t)
		assert.Len(t, driver.appliedMigrations, 3, "expected 3 migrations to be applied")

		// Verify migrations are applied in order
		for i, migration := range driver.appliedMigrations {
			expectedVersion := i + 1
			assert.Equal(t, expectedVersion, migration.Version, "expected migration version %d at index %d", expectedVersion, i)
		}
	})

	t.Run("migrations are sorted by version", func(t *testing.T) {
		driver := new(mockDriver)
		driver.On("Connect", mock.Anything).Return(nil)
		driver.On("CreateMigrationsTable", mock.Anything).Return(nil)
		driver.On("GetCurrentVersion", mock.Anything).Return(0, nil)
		driver.On("ApplyMigration", mock.Anything, mock.Anything).Return(nil).Times(3)
		driver.On("Close", mock.Anything).Return(nil)

		// Provide migrations in unsorted order
		migrations := []Migration{
			{Version: 3, Name: "add_index", SQL: "CREATE INDEX idx_users_id ON users(id);"},
			{Version: 1, Name: "create_users", SQL: "CREATE TABLE users (id INT);"},
			{Version: 2, Name: "create_posts", SQL: "CREATE TABLE posts (id INT);"},
		}

		ctx := context.Background()
		err := RunMigrations(ctx, driver, migrations)

		require.NoError(t, err)
		driver.AssertExpectations(t)

		// Verify migrations are applied in sorted order
		for i, migration := range driver.appliedMigrations {
			expectedVersion := i + 1
			assert.Equal(t, expectedVersion, migration.Version, "expected migration version %d at index %d", expectedVersion, i)
		}
	})

	t.Run("skips already applied migrations", func(t *testing.T) {
		driver := new(mockDriver)
		driver.On("Connect", mock.Anything).Return(nil)
		driver.On("CreateMigrationsTable", mock.Anything).Return(nil)
		driver.On("GetCurrentVersion", mock.Anything).Return(1, nil) // Already at version 1
		driver.On("ApplyMigration", mock.Anything, mock.Anything).Return(nil).Times(2)
		driver.On("Close", mock.Anything).Return(nil)

		migrations := []Migration{
			{Version: 1, Name: "create_users", SQL: "CREATE TABLE users (id INT);"},
			{Version: 2, Name: "create_posts", SQL: "CREATE TABLE posts (id INT);"},
			{Version: 3, Name: "add_index", SQL: "CREATE INDEX idx_users_id ON users(id);"},
		}

		ctx := context.Background()
		err := RunMigrations(ctx, driver, migrations)

		require.NoError(t, err)
		driver.AssertExpectations(t)

		// Should only apply migrations 2 and 3 (starting from index 1)
		require.Len(t, driver.appliedMigrations, 2, "expected 2 migrations to be applied")
		assert.Equal(t, 2, driver.appliedMigrations[0].Version, "expected first applied migration to be version 2")
		assert.Equal(t, 3, driver.appliedMigrations[1].Version, "expected second applied migration to be version 3")
	})

	t.Run("handles connect error", func(t *testing.T) {
		driver := new(mockDriver)
		driver.On("Connect", mock.Anything).Return(errors.New("connection failed"))

		migrations := []Migration{
			{Version: 1, Name: "create_users", SQL: "CREATE TABLE users (id INT);"},
		}

		ctx := context.Background()
		err := RunMigrations(ctx, driver, migrations)

		require.Error(t, err)
		assert.Equal(t, "connection failed", err.Error())
		driver.AssertExpectations(t)
		// Close should not be called when Connect fails
		driver.AssertNotCalled(t, "Close", mock.Anything)
	})

	t.Run("handles create table error", func(t *testing.T) {
		driver := new(mockDriver)
		driver.On("Connect", mock.Anything).Return(nil)
		driver.On("CreateMigrationsTable", mock.Anything).Return(errors.New("table creation failed"))
		driver.On("Close", mock.Anything).Return(nil)

		migrations := []Migration{
			{Version: 1, Name: "create_users", SQL: "CREATE TABLE users (id INT);"},
		}

		ctx := context.Background()
		err := RunMigrations(ctx, driver, migrations)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "table creation failed")
		driver.AssertExpectations(t)
		// Close should be called even when CreateMigrationsTable fails
		driver.AssertCalled(t, "Close", mock.Anything)
	})

	t.Run("handles get current version error", func(t *testing.T) {
		driver := new(mockDriver)
		driver.On("Connect", mock.Anything).Return(nil)
		driver.On("CreateMigrationsTable", mock.Anything).Return(nil)
		driver.On("GetCurrentVersion", mock.Anything).Return(0, errors.New("version check failed"))
		driver.On("Close", mock.Anything).Return(nil)

		migrations := []Migration{
			{Version: 1, Name: "create_users", SQL: "CREATE TABLE users (id INT);"},
		}

		ctx := context.Background()
		err := RunMigrations(ctx, driver, migrations)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "version check failed")
		driver.AssertExpectations(t)
	})

	t.Run("handles apply migration error", func(t *testing.T) {
		driver := new(mockDriver)
		driver.On("Connect", mock.Anything).Return(nil)
		driver.On("CreateMigrationsTable", mock.Anything).Return(nil)
		driver.On("GetCurrentVersion", mock.Anything).Return(0, nil)
		driver.On("ApplyMigration", mock.Anything, mock.Anything).Return(errors.New("migration failed"))
		driver.On("Close", mock.Anything).Return(nil)

		migrations := []Migration{
			{Version: 1, Name: "create_users", SQL: "CREATE TABLE users (id INT);"},
			{Version: 2, Name: "create_posts", SQL: "CREATE TABLE posts (id INT);"},
		}

		ctx := context.Background()
		err := RunMigrations(ctx, driver, migrations)

		require.Error(t, err)
		assert.Equal(t, "migration failed", err.Error())
		driver.AssertExpectations(t)
		// Should have attempted to apply the first migration
		assert.Len(t, driver.appliedMigrations, 1, "expected 1 migration to be attempted")
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		driver := new(mockDriver)
		driver.On("Connect", mock.Anything).Return(nil)
		driver.On("CreateMigrationsTable", mock.Anything).Return(nil)
		driver.On("GetCurrentVersion", mock.Anything).Return(0, nil)
		driver.On("Close", mock.Anything).Return(nil)

		migrations := []Migration{
			{Version: 1, Name: "create_users", SQL: "CREATE TABLE users (id INT);"},
			{Version: 2, Name: "create_posts", SQL: "CREATE TABLE posts (id INT);"},
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := RunMigrations(ctx, driver, migrations)

		require.Error(t, err)
		assert.True(t, errors.Is(err, context.Canceled), "expected context.Canceled error")
		driver.AssertExpectations(t)
	})

	t.Run("handles context timeout", func(t *testing.T) {
		driver := new(mockDriver)
		driver.On("Connect", mock.Anything).Return(nil)
		driver.On("CreateMigrationsTable", mock.Anything).Return(nil)
		driver.On("GetCurrentVersion", mock.Anything).Return(0, nil)
		driver.On("Close", mock.Anything).Return(nil)

		migrations := []Migration{
			{Version: 1, Name: "create_users", SQL: "CREATE TABLE users (id INT);"},
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
		defer cancel()
		time.Sleep(time.Millisecond) // Ensure timeout has occurred

		err := RunMigrations(ctx, driver, migrations)

		require.Error(t, err)
		assert.True(t, errors.Is(err, context.DeadlineExceeded), "expected context.DeadlineExceeded error")
		driver.AssertExpectations(t)
	})

	t.Run("handles empty migrations list", func(t *testing.T) {
		driver := new(mockDriver)
		driver.On("Connect", mock.Anything).Return(nil)
		driver.On("CreateMigrationsTable", mock.Anything).Return(nil)
		driver.On("GetCurrentVersion", mock.Anything).Return(0, nil)
		driver.On("Close", mock.Anything).Return(nil)

		migrations := []Migration{}

		ctx := context.Background()
		err := RunMigrations(ctx, driver, migrations)

		require.NoError(t, err, "expected no error for empty migrations")
		driver.AssertExpectations(t)
		assert.Empty(t, driver.appliedMigrations, "expected no migrations to be applied")
	})

	t.Run("handles close error", func(t *testing.T) {
		driver := new(mockDriver)
		driver.On("Connect", mock.Anything).Return(nil)
		driver.On("CreateMigrationsTable", mock.Anything).Return(nil)
		driver.On("GetCurrentVersion", mock.Anything).Return(0, nil)
		driver.On("ApplyMigration", mock.Anything, mock.Anything).Return(nil)
		driver.On("Close", mock.Anything).Return(errors.New("close failed"))

		migrations := []Migration{
			{Version: 1, Name: "create_users", SQL: "CREATE TABLE users (id INT);"},
		}

		ctx := context.Background()
		_ = RunMigrations(ctx, driver, migrations)

		// Close errors are deferred, so they might not be returned
		// This test verifies Close is called even if it errors
		driver.AssertExpectations(t)
		driver.AssertCalled(t, "Close", mock.Anything)
	})
}
