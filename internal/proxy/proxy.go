package proxy

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/companion/api-gateway/internal/consul"
)

// Routes maps path prefixes to Consul service names.
var Routes = map[string]string{
	"/api/auth":       "auth-service",
	"/api/location":   "location-service",
	"/api/ws":         "socketio-handler",
	"/api/rides":      "rides-service",
	"/api/pathfinder":  "pathfinder",
	"/api/config":     "config-service",
}

type Proxy struct {
	consul *consul.Client
}

func New(c *consul.Client) *Proxy {
	return &Proxy{consul: c}
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for prefix, serviceName := range Routes {
		if strings.HasPrefix(r.URL.Path, prefix) {
			p.forward(w, r, serviceName, prefix)
			return
		}
	}

	http.Error(w, `{"error":"no matching route"}`, http.StatusNotFound)
}

func (p *Proxy) forward(w http.ResponseWriter, r *http.Request, serviceName, prefix string) {
	addr, port, err := p.consul.Resolve(serviceName)
	if err != nil {
		log.Printf("resolve %s: %v", serviceName, err)
		http.Error(w, fmt.Sprintf(`{"error":"service unavailable: %s"}`, serviceName), http.StatusBadGateway)
		return
	}

	target, _ := url.Parse(fmt.Sprintf("http://%s:%d", addr, port))

	originalHost := r.Host
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			// Strip the gateway prefix, keep the rest of the path.
			req.URL.Path = strings.TrimPrefix(req.URL.Path, prefix)
			if req.URL.Path == "" {
				req.URL.Path = "/"
			}
			// Keep the original host so downstream services build correct redirect URLs.
			req.Host = originalHost
			req.Header.Set("X-Forwarded-Host", originalHost)
			req.Header.Set("X-Forwarded-Prefix", prefix)
		},
	}

	proxy.ServeHTTP(w, r)
}
