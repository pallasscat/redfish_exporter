package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
)

type Endpoint string

type EndpointConfig struct {
	Username string
	Password string
	Insecure bool
}

type Config map[Endpoint]EndpointConfig

func (c Config) GetEndpointConfig(endpoint string) (EndpointConfig, error) {
	if cfg, ok := c[Endpoint(endpoint)]; ok {
		return cfg, nil
	}

	return EndpointConfig{}, fmt.Errorf("error: endpoint %q not configured", endpoint)
}

func LoadConfig(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %s", err)
	}

	c := &Config{}
	if err := yaml.Unmarshal(b, c); err != nil {
		return nil, fmt.Errorf("error unmarshalling config file: %s", err)
	}

	if len(*c) == 0 {
		return nil, fmt.Errorf("error: config file has no endpoints defined")
	}

	return c, nil
}
