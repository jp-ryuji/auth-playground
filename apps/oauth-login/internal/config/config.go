package config

import "os"

type Config struct {
	HydraAdminURL string
	Port          string
}

func Load() Config {
	return Config{
		HydraAdminURL: envOrDefault("HYDRA_ADMIN_URL", "http://127.0.0.1:4445"),
		Port:          envOrDefault("PORT", "8090"),
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
