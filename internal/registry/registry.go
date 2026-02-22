package registry

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"

	"github.com/companion/api-gateway/internal/consul"
)

type RegisterRequest struct {
	Name        string `json:"name"`
	Address     string `json:"address"`
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

// ServeHTTP handles POST /internal/register, DELETE /internal/deregister,
// and GET /internal/resolve.
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
	case r.Method == http.MethodGet && r.URL.Path == "/internal/resolve":
		h.resolve(w, r)
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

	addr := req.Address
	if addr == "" {
		addr, _, _ = net.SplitHostPort(r.RemoteAddr)
	}
	if err := h.consul.Register(req.Name, addr, req.Port, req.HealthCheck); err != nil {
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

func (h *Handler) resolve(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, `{"error":"name query param required"}`, http.StatusBadRequest)
		return
	}

	addr, port, err := h.consul.Resolve(name)
	if err != nil {
		http.Error(w, `{"error":"service not found: `+name+`"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"name": name, "address": addr, "port": port})
}
