package config

import "os"

type Config struct {
	GatewayPort   string
	ConsulAddr    string
	GatewaySecret string
}

func Load() *Config {
	return &Config{
		GatewayPort:   getEnv("GATEWAY_PORT", "9090"),
		ConsulAddr:    getEnv("CONSUL_ADDR", "localhost:8500"),
		GatewaySecret: getEnv("GATEWAY_SECRET", "supersecret"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
