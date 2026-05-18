package login

import (
	"net/http"

	"github.com/jp-ryuji/auth-playground/apps/oauth-login/hydra"
)

// Handler handles GET /login?login_challenge=...
//
// SIGNUP-05: extract only login_challenge from the query string, call Hydra
// Admin to resolve it, then return 501 as a scaffold placeholder for SIGNUP-06.
type Handler struct {
	HydraAdmin *hydra.Client
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

	// Placeholder for SIGNUP-06: check Kratos session and redirect.
	http.Error(w, "not implemented", http.StatusNotImplemented)
}
