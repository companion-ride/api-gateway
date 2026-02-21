package registry

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/companion/api-gateway/internal/consul"
)

type RegisterRequest struct {
	Name        string `json:"name"`
	Port        int    `json:"port"`
	HealthCheck string `json:"health_check"`
}

type Handler struct {
	consul *consul.Client
	secret string
}

func NewHandler(c *consul.Client, secret string) *Handler {
	return &Handler{consul: c, secret: secret}
}

// ServeHTTP handles POST /internal/register and DELETE /internal/deregister.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !h.authorize(r) {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/internal/register":
		h.register(w, r)
	case r.Method == http.MethodDelete && r.URL.Path == "/internal/deregister":
		h.deregister(w, r)
	default:
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	}
}

func (h *Handler) authorize(r *http.Request) bool {
	auth := r.Header.Get("Authorization")
	return strings.TrimPrefix(auth, "Bearer ") == h.secret
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.Port == 0 {
		http.Error(w, `{"error":"name and port are required"}`, http.StatusBadRequest)
		return
	}
	if req.HealthCheck == "" {
		req.HealthCheck = "/health"
	}

	if err := h.consul.Register(req.Name, "127.0.0.1", req.Port, req.HealthCheck); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "registered", "service": req.Name})
}

func (h *Handler) deregister(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, `{"error":"name query param required"}`, http.StatusBadRequest)
		return
	}

	if err := h.consul.Deregister(name); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deregistered", "service": name})
}
