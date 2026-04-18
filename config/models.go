package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type ModelConfig struct {
	ID                 string                 `yaml:"id"`
	Body               map[string]interface{} `yaml:"body"`
	Path               string                 `yaml:"path"`
	ExtraBody          map[string]interface{} `yaml:"extra_body"`
	ChatTemplateKwargs map[string]interface{} `yaml:"chat_template_kwargs"`
}

type Endpoint struct {
	Host   string        `yaml:"host"`
	Models []ModelConfig `yaml:"models"`
}

type HTTPConfig struct {
	Host               string `yaml:"host"`
	Port               int    `yaml:"port"`
	MaxRequestBodySize int    `yaml:"max_request_body_size"` // in bytes, 0 = no limit
	Timeout            int    `yaml:"timeout"`               // in seconds, 0 = default (1 hour)
}

type Config struct {
	Server    HTTPConfig `yaml:"server"`
	Endpoints []Endpoint `yaml:"endpoints"`
}

func Load(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if len(config.Endpoints) == 0 {
		return nil, fmt.Errorf("no endpoints configured")
	}

	if config.Server.Host == "" {
		config.Server.Host = "0.0.0.0"
	}

	if config.Server.Port == 0 {
		config.Server.Port = 8080
	}

	// Default max request body size to 100MB if not configured
	if config.Server.MaxRequestBodySize == 0 {
		config.Server.MaxRequestBodySize = 100 * 1024 * 1024 // 100MB for large context support
	}

	// Default timeout to 1 hour (3600 seconds) if not configured
	// Supports 260K context + long generations with speed degradation
	if config.Server.Timeout == 0 {
		config.Server.Timeout = 3600 // 1 hour
	}

	// Validate endpoints
	seenModelIDs := make(map[string]bool)
	for i, endpoint := range config.Endpoints {
		if endpoint.Host == "" {
			return nil, fmt.Errorf("endpoint %d: host is required", i)
		}
		parsedURL, err := url.Parse(endpoint.Host)
		if err != nil {
			return nil, fmt.Errorf("endpoint %d: invalid host URL %q: %w", i, endpoint.Host, err)
		}
		// Validate URL scheme (prevent SSRF)
		// Trim any whitespace for safety (url.Parse should reject these, but be defensive)
		scheme := strings.TrimSpace(parsedURL.Scheme)
		scheme = strings.ToLower(scheme)
		if scheme != "http" && scheme != "https" {
			return nil, fmt.Errorf("endpoint %d: host URL must use http:// or https:// scheme, got %q", i, parsedURL.Scheme)
		}
		if len(endpoint.Models) == 0 {
			return nil, fmt.Errorf("endpoint %d: at least one model is required", i)
		}
		for j, model := range endpoint.Models {
			if model.ID == "" {
				return nil, fmt.Errorf("endpoint %d, model %d: id is required", i, j)
			}
			if seenModelIDs[model.ID] {
				return nil, fmt.Errorf("duplicate model id %q", model.ID)
			}
			seenModelIDs[model.ID] = true
			if model.Path == "" {
				return nil, fmt.Errorf("endpoint %d, model %d (%s): path is required", i, j, model.ID)
			}
			if !strings.HasPrefix(model.Path, "/") {
				return nil, fmt.Errorf("endpoint %d, model %d (%s): path must start with /", i, j, model.ID)
			}
		}
	}

	return &config, nil
}
