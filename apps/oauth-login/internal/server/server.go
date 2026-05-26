package server

import (
	"context"
	"errors"
	"net/http"

	"github.com/jp-ryuji/auth-playground/apps/oauth-login/hydra"
	"github.com/jp-ryuji/auth-playground/apps/oauth-login/internal/config"
	"github.com/jp-ryuji/auth-playground/apps/oauth-login/kratos"
	"github.com/jp-ryuji/auth-playground/apps/oauth-login/login"
)

type Deps struct {
	Cfg config.Config
}

type Server struct {
	httpServer *http.Server
}

func New(deps Deps) *Server {
	mux := http.NewServeMux()
	loginHandler := &login.Handler{
		HydraAdmin:        hydra.NewClient(deps.Cfg.HydraAdminURL, http.DefaultClient),
		KratosPublic:      kratos.NewClient(deps.Cfg.KratosPublicURL, http.DefaultClient),
		OAuthLoginBaseURL: deps.Cfg.OAuthLoginBaseURL,
	}
	mux.Handle("GET /login", loginHandler)
	mux.Handle("GET /login/resume", &login.ResumeHandler{Handler: *loginHandler})

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
