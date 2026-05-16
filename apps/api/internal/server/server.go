package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/jp-ryuji/auth-playground/apps/api/internal/config"
	"github.com/jp-ryuji/auth-playground/apps/api/internal/discovery"
	"github.com/jp-ryuji/auth-playground/apps/api/internal/signup"
)

type Deps struct {
	Doc   *discovery.Document
	Store *signup.Store
	Cfg   config.Config
}

type Server struct {
	httpServer *http.Server
}

func New(deps Deps) *Server {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ok")
	})
	mux.Handle("GET /auth/login", &signup.LoginHandler{
		Doc:          deps.Doc,
		Store:        deps.Store,
		ClientID:     deps.Cfg.ClientID,
		RedirectURI:  deps.Cfg.RedirectURI,
		Scopes:       deps.Cfg.Scopes,
		SecureCookie: deps.Cfg.SecureCookie,
	})

	return &Server{
		httpServer: &http.Server{
			Addr:    ":" + deps.Cfg.Port,
			Handler: mux,
		},
	}
}

func (s *Server) Start() <-chan error {
	errCh := make(chan error, 1)
	go func() {
		if err := s.httpServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()
	return errCh
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
