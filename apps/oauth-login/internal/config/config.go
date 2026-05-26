package config

import "os"

type Config struct {
	HydraAdminURL    string
	KratosPublicURL  string
	OAuthLoginBaseURL string
	Port             string
}

func Load() Config {
	return Config{
		HydraAdminURL:    envOrDefault("HYDRA_ADMIN_URL", "http://127.0.0.1:4445"),
		KratosPublicURL:  envOrDefault("KRATOS_PUBLIC_URL", "http://127.0.0.1:4433"),
		OAuthLoginBaseURL: envOrDefault("OAUTH_LOGIN_BASE_URL", "http://127.0.0.1:3000"),
		Port:             envOrDefault("PORT", "3000"),
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
