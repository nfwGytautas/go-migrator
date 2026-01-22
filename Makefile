help:
	@echo "Usage: make <target>"

run-tests:
	@go test -v -race -coverprofile=coverage.out ./...

run-migrator:
	@docker compose --profile databases up -d
	-@docker compose --profile migrator up --build --force-recreate
	@docker compose --profile "*" down --volumes
