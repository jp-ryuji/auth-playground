package config

import (
	"os"
	"strings"
	"time"
)

type Config struct {
	HydraIssuer      string
	ClientID         string
	RedirectURI      string
	Scopes           []string
	Port             string
	AuthStateTTL     time.Duration
	SecureCookie     bool
	DiscoveryTimeout time.Duration
}

func Load() Config {
	scopes := strings.Fields(os.Getenv("SCOPES"))
	if len(scopes) == 0 {
		scopes = []string{"openid", "offline"}
	}

	authStateTTL, err := time.ParseDuration(os.Getenv("AUTH_STATE_TTL"))
	if err != nil {
		authStateTTL = 10 * time.Minute
	}

	discoveryTimeout, err := time.ParseDuration(os.Getenv("DISCOVERY_TIMEOUT"))
	if err != nil {
		discoveryTimeout = 10 * time.Second
	}

	secureCookieVal := os.Getenv("SECURE_COOKIE")

	return Config{
		HydraIssuer:      envOrDefault("HYDRA_ISSUER", "http://127.0.0.1:4444/"),
		ClientID:         envOrDefault("CLIENT_ID", "auth-playground-rp"),
		RedirectURI:      envOrDefault("REDIRECT_URI", "http://127.0.0.1:8080/auth/callback"),
		Scopes:           scopes,
		Port:             envOrDefault("PORT", "8080"),
		AuthStateTTL:     authStateTTL,
		SecureCookie:     secureCookieVal == "true" || secureCookieVal == "1",
		DiscoveryTimeout: discoveryTimeout,
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
