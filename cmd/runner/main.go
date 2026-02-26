package main

import (
	"context"
	"errors"
	"log"
	"os/signal"
	"syscall"

	"feishu-codex-runner/internal/config"
	"feishu-codex-runner/internal/orchestrator"
)

func main() {
	cfg, err := config.LoadRuntime()
	if err != nil {
		log.Fatalf("load runtime: %v", err)
	}
	repos, err := config.LoadRepos(cfg.ReposFile)
	if err != nil {
		log.Fatalf("load repos: %v", err)
	}
	allow, err := config.LoadAllowList(cfg.AllowListFile)
	if err != nil {
		log.Fatalf("load allowlist: %v", err)
	}
	app, err := orchestrator.New(cfg, repos, allow)
	if err != nil {
		log.Fatalf("create app: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	log.Printf("runner started, poll interval=%s", cfg.PollInterval)
	if err := app.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("runner stopped: %v", err)
	}
	log.Println("runner exited")
}
