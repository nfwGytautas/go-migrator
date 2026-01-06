# go-migrator
A super simple go SQL migrator library

## Why only up?
It is my personal belief that having a flow of only "UP" shields from a large portion of the things that can go wrong in production.

## Usage

### As a `Docker` worker image

```bash
docker pull ghcr.io/apphene/go-migrator:latest
```

#### Docker Compose Example

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:16
    environment:
      POSTGRES_DB: myapp
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
    ports:
      - "5432:5432"

  migrations:
    image: ghcr.io/apphene/go-migrator:latest
    depends_on:
      - postgres
    environment:
      DATABASE_DSN: postgres://postgres:postgres@postgres:5432/myapp?sslmode=disable
      MIGRATIONS_DIR: /migrations
      DRIVER: postgres
      MAX_RETRIES: 5 # Optional
      RETRY_DELAY: 3s # Optional
      TIMEOUT: 30s # Optional
    volumes:
      - ./migrations:/migrations
```

### As a `Go` Package

```bash
go get github.com/apphene/go-migrator
```

#### Example Usage

```go
package main

import (
    "context"
    "log"
    "time"

    gomigrator "github.com/apphene/go-migrator"
    "github.com/apphene/go-migrator/drivers"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // Create a Postgres driver
    dsn := "postgres://user:password@localhost:5432/dbname?sslmode=disable"
    driver := drivers.NewPostgresDriver(dsn)

    // Load migrations from a directory
    migrations, err := gomigrator.LoadMigrationsFromDir("./migrations")
    if err != nil {
        log.Fatalf("Failed to load migrations: %v", err)
    }

    // Run migrations
    if err := gomigrator.RunMigrations(ctx, driver, migrations); err != nil {
        log.Fatalf("Failed to run migrations: %v", err)
    }

    log.Println("Migrations completed successfully")
}
```

#### Migration File Format

Migration files should be named with the format: `<version>_<name>.sql`

Example:
- `1_create_users_table.sql`
- `2_add_email_index.sql`
- `3_create_posts_table.sql`
