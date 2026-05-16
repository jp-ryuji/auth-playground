package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/jp-ryuji/auth-playground/apps/api/internal/discovery"
	"github.com/jp-ryuji/auth-playground/apps/api/internal/signup"
)

const hydraIssuer = "http://127.0.0.1:4444/"

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	doc, err := discovery.NewClient(hydraIssuer, http.DefaultClient).Fetch(ctx)
	if err != nil {
		log.Fatalf("discovery: %v", err)
	}

	store := signup.NewStore(10 * time.Minute)

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "ok")
	})
	http.Handle("GET /auth/login", &signup.LoginHandler{
		Doc:         doc,
		Store:       store,
		ClientID:    "auth-playground-rp",
		RedirectURI: "http://127.0.0.1:8080/auth/callback",
		Scopes:      []string{"openid", "offline"},
	})

	fmt.Println("server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
