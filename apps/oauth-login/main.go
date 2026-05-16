package main

import (
	"log"
	"net/http"
	"os"

	"github.com/jp-ryuji/auth-playground/apps/oauth-login/hydra"
	"github.com/jp-ryuji/auth-playground/apps/oauth-login/login"
)

func main() {
	adminURL := envOrDefault("HYDRA_ADMIN_URL", "http://127.0.0.1:4445")
	port := envOrDefault("PORT", "8090")

	mux := http.NewServeMux()
	mux.Handle("GET /login", &login.Handler{
		HydraAdmin: hydra.NewClient(adminURL, http.DefaultClient),
	})

	log.Printf("oauth-login listening on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("server: %v", err)
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
