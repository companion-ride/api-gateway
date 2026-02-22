package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/companion/api-gateway/internal/config"
	"github.com/companion/api-gateway/internal/consul"
	"github.com/companion/api-gateway/internal/proxy"
	"github.com/companion/api-gateway/internal/registry"
)

func main() {
	cfg := config.Load()

	consulClient, err := consul.New(cfg.ConsulAddr)
	if err != nil {
		log.Fatalf("failed to connect to consul: %v", err)
	}
	log.Printf("connected to consul at %s", cfg.ConsulAddr)

	registryHandler := registry.NewHandler(consulClient, cfg.GatewaySecret)
	proxyHandler := proxy.New(consulClient)

	mux := http.NewServeMux()

	// Internal endpoints (authorized registration/deregistration).
	mux.Handle("/internal/", registryHandler)

	// Health check for the gateway itself.
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// All /api/* requests are proxied to backend services.
	mux.Handle("/api/", proxyHandler)

	// Logging middleware.
	handler := logMiddleware(mux)

	srv := &http.Server{
		Addr:         ":" + cfg.GatewayPort,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown.
	go func() {
		log.Printf("api-gateway listening on :%s", cfg.GatewayPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}

func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		// Don't log health checks to reduce noise.
		if !strings.HasPrefix(r.URL.Path, "/health") {
			log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
		}
	})
}
