package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/jp-ryuji/auth-playground/apps/oauth-login/internal/config"
	"github.com/jp-ryuji/auth-playground/apps/oauth-login/internal/server"
)

func main() {
	cfg := config.Load()

	srv := server.New(server.Deps{Cfg: cfg})
	errCh := srv.Start()
	log.Printf("oauth-login listening on :%s", cfg.Port)

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
