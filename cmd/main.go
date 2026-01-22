package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/nfwGytautas/go-migrator/executor"
)

func main() {
	configPath := "gomigrator.yaml"
	if len(os.Args) >= 2 {
		configPath = os.Args[1]
	}

	cfg, err := executor.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if !executor.Execute(ctx, cfg) {
		os.Exit(1)
	}
}
