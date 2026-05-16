package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/jp-ryuji/auth-playground/apps/api/internal/config"
	"github.com/jp-ryuji/auth-playground/apps/api/internal/discovery"
	"github.com/jp-ryuji/auth-playground/apps/api/internal/server"
	"github.com/jp-ryuji/auth-playground/apps/api/internal/signup"
)

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), cfg.DiscoveryTimeout)
	defer cancel()

	doc, err := discovery.NewClient(cfg.HydraIssuer, http.DefaultClient).Fetch(ctx)
	if err != nil {
		log.Fatalf("discovery: %v", err)
	}

	store := signup.NewStore(cfg.AuthStateTTL)

	srv := server.New(server.Deps{Doc: doc, Store: store, Cfg: cfg})
	errCh := srv.Start()
	log.Printf("server listening on :%s", cfg.Port)

	sigCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	select {
	case err := <-errCh:
		log.Fatalf("server error: %v", err)
	case <-sigCtx.Done():
		log.Println("shutdown signal received")
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
