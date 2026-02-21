package consul

import (
	"fmt"

	consulapi "github.com/hashicorp/consul/api"
)

type Client struct {
	consul *consulapi.Client
}

func New(addr string) (*Client, error) {
	cfg := consulapi.DefaultConfig()
	cfg.Address = addr

	client, err := consulapi.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("consul client: %w", err)
	}

	return &Client{consul: client}, nil
}

// Register adds a service to Consul with an HTTP health check.
func (c *Client) Register(name, address string, port int, healthPath string) error {
	reg := &consulapi.AgentServiceRegistration{
		ID:      name,
		Name:    name,
		Address: address,
		Port:    port,
		Check: &consulapi.AgentServiceCheck{
			HTTP:                           fmt.Sprintf("http://%s:%d%s", address, port, healthPath),
			Interval:                       "10s",
			Timeout:                        "3s",
			DeregisterCriticalServiceAfter: "30s",
		},
	}
	return c.consul.Agent().ServiceRegister(reg)
}

// Deregister removes a service from Consul.
func (c *Client) Deregister(serviceID string) error {
	return c.consul.Agent().ServiceDeregister(serviceID)
}

// Resolve returns the address and port for a service name.
// Returns an error if the service is not found or has no healthy instances.
func (c *Client) Resolve(serviceName string) (string, int, error) {
	entries, _, err := c.consul.Health().Service(serviceName, "", true, nil)
	if err != nil {
		return "", 0, fmt.Errorf("consul resolve %s: %w", serviceName, err)
	}
	if len(entries) == 0 {
		return "", 0, fmt.Errorf("no healthy instances of %s", serviceName)
	}

	svc := entries[0].Service
	addr := svc.Address
	if addr == "" {
		addr = entries[0].Node.Address
	}
	return addr, svc.Port, nil
}
