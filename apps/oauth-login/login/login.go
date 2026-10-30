package login

import (
	"net/http"
	"net/url"

	"github.com/jp-ryuji/auth-playground/apps/oauth-login/hydra"
	"github.com/jp-ryuji/auth-playground/apps/oauth-login/kratos"
)

// Handler handles GET /login?login_challenge=... and GET /login/resume?login_challenge=...
//
// SIGNUP-05: extract only login_challenge from the query string, call Hydra
// Admin to resolve it.
// SIGNUP-06: check for a Kratos session; redirect to Kratos self-service
// registration flow when no session exists.
// SIGNUP-07: when a Kratos session exists, accept the login with subject =
// identity id and redirect to Hydra's redirect_to.
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

	if _, err := h.HydraAdmin.GetLoginRequest(r.Context(), challengeID); err != nil {
		http.Error(w, "failed to resolve login challenge", http.StatusBadGateway)
		return
	}

	session, err := h.KratosPublic.GetSession(r.Context(), r.Header.Get("Cookie"))
	if err != nil {
		http.Error(w, "failed to check Kratos session", http.StatusBadGateway)
		return
	}

	if session == nil {
		h.redirectToKratosRegistration(w, r, challengeID)
		return
	}

	h.acceptLoginAndRedirect(w, r, challengeID, session.Identity.ID)
}

// ResumeHandler handles GET /login/resume?login_challenge=... after Kratos
// self-service flow completes.
type ResumeHandler struct {
	Handler
}

func (h *ResumeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	challengeID := r.URL.Query().Get("login_challenge")
	if challengeID == "" {
		http.Error(w, "missing login_challenge", http.StatusBadRequest)
		return
	}

	if _, err := h.HydraAdmin.GetLoginRequest(r.Context(), challengeID); err != nil {
		http.Error(w, "failed to resolve login challenge", http.StatusBadGateway)
		return
	}

	session, err := h.KratosPublic.GetSession(r.Context(), r.Header.Get("Cookie"))
	if err != nil {
		http.Error(w, "failed to check Kratos session", http.StatusBadGateway)
		return
	}

	if session == nil {
		h.redirectToKratosRegistration(w, r, challengeID)
		return
	}

	h.acceptLoginAndRedirect(w, r, challengeID, session.Identity.ID)
}

func (h *Handler) redirectToKratosRegistration(w http.ResponseWriter, r *http.Request, challengeID string) {
	returnTo := h.OAuthLoginBaseURL + "/login/resume?login_challenge=" + url.QueryEscape(challengeID)
	redirectURL := h.KratosPublic.PublicURL() + "/self-service/registration/browser?return_to=" + url.QueryEscape(returnTo)
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func (h *Handler) acceptLoginAndRedirect(w http.ResponseWriter, r *http.Request, challengeID, subject string) {
	accepted, err := h.HydraAdmin.AcceptLoginRequest(r.Context(), challengeID, subject)
	if err != nil {
		http.Error(w, "failed to accept login challenge", http.StatusBadGateway)
		return
	}

	http.Redirect(w, r, accepted.RedirectTo, http.StatusFound)
}
