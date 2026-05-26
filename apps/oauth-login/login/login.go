package login

import (
	"net/http"
	"net/url"

	"github.com/jp-ryuji/auth-playground/apps/oauth-login/hydra"
	"github.com/jp-ryuji/auth-playground/apps/oauth-login/kratos"
)

// Handler handles GET /login?login_challenge=...
//
// SIGNUP-05: extract only login_challenge from the query string, call Hydra
// Admin to resolve it.
// SIGNUP-06: check for a Kratos session; redirect to Kratos self-service
// registration flow when no session exists.
type Handler struct {
	HydraAdmin        *hydra.Client
	KratosPublic      *kratos.Client
	OAuthLoginBaseURL string
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	challengeID := r.URL.Query().Get("login_challenge")
	if challengeID == "" {
		http.Error(w, "missing login_challenge", http.StatusBadRequest)
		return
	}

	_, err := h.HydraAdmin.GetLoginRequest(r.Context(), challengeID)
	if err != nil {
		http.Error(w, "failed to resolve login challenge", http.StatusBadGateway)
		return
	}

	session, err := h.KratosPublic.GetSession(r.Context(), r.Header.Get("Cookie"))
	if err != nil {
		http.Error(w, "failed to check Kratos session", http.StatusBadGateway)
		return
	}

	if session == nil {
		returnTo := h.OAuthLoginBaseURL + "/login/resume?login_challenge=" + url.QueryEscape(challengeID)
		redirectURL := h.KratosPublic.PublicURL() + "/self-service/registration/browser?return_to=" + url.QueryEscape(returnTo)
		http.Redirect(w, r, redirectURL, http.StatusFound)
		return
	}

	// Placeholder for SIGNUP-07: accept login with Kratos identity as subject.
	http.Error(w, "not implemented", http.StatusNotImplemented)
}
