package signup

import (
	"net/http"

	"github.com/jp-ryuji/auth-playground/apps/api/internal/discovery"
)

const authStateCookieName = "auth_state"

// LoginHandler handles GET /auth/login: generates per-request PKCE, state,
// and nonce; stores them server-side; sets an auth-state cookie binding the
// browser to the pending record; and redirects to Hydra's authorize endpoint.
//
// Only the S256 challenge leaves apps/api — the verifier stays in Store
// (SIGNUP-04). Doc, Store, ClientID, RedirectURI, and Scopes must all be set.
type LoginHandler struct {
	Doc          *discovery.Document
	Store        *Store
	ClientID     string
	RedirectURI  string
	Scopes       []string
	SecureCookie bool // true in production (HTTPS); false in local dev
	CookieMaxAge int  // seconds; 0 defaults to 600 (spec max ≤10 min)
}

func (h *LoginHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	pkce, err := NewPKCE()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	state, err := RandomURLSafe(16)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	nonce, err := RandomURLSafe(16)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if err := h.Store.Save(state, AuthState{State: state, Nonce: nonce, Verifier: pkce.Verifier}); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	authorizeURL, err := BuildAuthorizeURL(h.Doc, AuthorizeParams{
		ClientID:      h.ClientID,
		RedirectURI:   h.RedirectURI,
		Scopes:        h.Scopes,
		State:         state,
		Nonce:         nonce,
		CodeChallenge: pkce.Challenge,
	})
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	maxAge := h.CookieMaxAge
	if maxAge == 0 {
		maxAge = 600
	}
	http.SetCookie(w, &http.Cookie{
		Name:     authStateCookieName,
		Value:    state,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   h.SecureCookie,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, authorizeURL, http.StatusFound)
}
